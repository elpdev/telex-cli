package screens

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/emailtext"
	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

type TasksSyncFunc func(context.Context) (TasksSyncResult, error)
type CreateTaskProjectFunc func(context.Context, tasks.ProjectInput) (*tasks.Project, error)
type CreateTaskCardFunc func(context.Context, int64, tasks.CardInput) (*tasks.Card, error)
type UpdateTaskCardFunc func(context.Context, int64, int64, tasks.CardInput) (*tasks.Card, error)
type DeleteTaskCardFunc func(context.Context, int64, int64) error

type TasksSyncResult struct {
	Projects int
	Boards   int
	Cards    int
}

type Tasks struct {
	store         taskstore.Store
	sync          TasksSyncFunc
	createProject CreateTaskProjectFunc
	createCard    CreateTaskCardFunc
	updateCard    UpdateTaskCardFunc
	deleteCard    DeleteTaskCardFunc
	projects      []taskstore.CachedProject
	project       *taskstore.CachedProject
	board         *taskstore.CachedBoard
	cards         []taskstore.CachedCard
	rows          []taskRow
	rowList       list.Model
	index         int
	detail        *taskstore.CachedCard
	detailScroll  int
	filter        string
	filtering     bool
	confirm       string
	loading       bool
	syncing       bool
	err           error
	status        string
	keys          TasksKeyMap
}

type TasksKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Search  key.Binding
	Project key.Binding
	New     key.Binding
	Edit    key.Binding
	Delete  key.Binding
}

type taskRow struct {
	Kind    string
	Name    string
	Project *taskstore.CachedProject
	Column  *tasks.BoardColumn
	Card    *taskstore.CachedCard
	Missing bool
}

type taskListItem struct{ row taskRow }

func (i taskListItem) FilterValue() string { return i.row.Name }

type tasksLoadedMsg struct {
	projects []taskstore.CachedProject
	project  *taskstore.CachedProject
	board    *taskstore.CachedBoard
	cards    []taskstore.CachedCard
	err      error
}

type tasksSyncedMsg struct {
	result TasksSyncResult
	loaded tasksLoadedMsg
	err    error
}

type taskActionFinishedMsg struct {
	status string
	loaded tasksLoadedMsg
	err    error
}

type TasksActionMsg struct{ Action string }

func NewTasks(store taskstore.Store, sync TasksSyncFunc) Tasks {
	return Tasks{store: store, sync: sync, loading: true, keys: DefaultTasksKeyMap(), rowList: newTaskList(nil, 0, 0, 0)}
}

func (t Tasks) WithActions(createProject CreateTaskProjectFunc, createCard CreateTaskCardFunc, updateCard UpdateTaskCardFunc, deleteCard DeleteTaskCardFunc) Tasks {
	t.createProject = createProject
	t.createCard = createCard
	t.updateCard = updateCard
	t.deleteCard = deleteCard
	return t
}

func DefaultTasksKeyMap() TasksKeyMap {
	return TasksKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync tasks")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Project: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "projects")),
		New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new card/project")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit card")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete card")),
	}
}

func (t Tasks) Init() tea.Cmd { return t.loadCmd(0) }

func (t Tasks) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tasksLoadedMsg:
		t.loading = false
		t.err = msg.err
		if msg.err == nil {
			t.projects = msg.projects
			t.project = msg.project
			t.board = msg.board
			t.cards = msg.cards
			t.rows = t.buildRows()
			t.clampIndex()
			t.syncList()
		}
		return t, nil
	case tasksSyncedMsg:
		t.syncing = false
		t.err = msg.err
		if msg.err == nil {
			t.status = fmt.Sprintf("Synced %d project(s), %d board(s), %d card(s)", msg.result.Projects, msg.result.Boards, msg.result.Cards)
			t.projects = msg.loaded.projects
			t.project = msg.loaded.project
			t.board = msg.loaded.board
			t.cards = msg.loaded.cards
			t.rows = t.buildRows()
			t.clampIndex()
			t.syncList()
		} else {
			t.status = ""
		}
		return t, nil
	case taskActionFinishedMsg:
		t.loading = false
		t.err = msg.err
		if msg.err != nil {
			t.status = fmt.Sprintf("Tasks action failed: %v", msg.err)
			return t, nil
		}
		t.status = msg.status
		t.projects = msg.loaded.projects
		t.project = msg.loaded.project
		t.board = msg.loaded.board
		t.cards = msg.loaded.cards
		t.rows = t.buildRows()
		t.clampIndex()
		t.syncList()
		return t, nil
	case TasksActionMsg:
		return t.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t Tasks) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if t.loading {
		return style.Render("Loading local tasks cache...")
	}
	if t.err != nil {
		return style.Render(fmt.Sprintf("Tasks cache error: %v\n\nRun `telex tasks sync` or press S to populate Tasks.", t.err))
	}
	var b strings.Builder
	b.WriteString("Tasks")
	if t.project != nil {
		b.WriteString(" / " + t.project.Meta.Name)
	}
	b.WriteString("\n")
	if t.status != "" {
		b.WriteString(t.status + "\n")
	}
	if t.syncing {
		b.WriteString("Syncing remote Tasks...\n")
	}
	if t.filtering {
		b.WriteString("Filter: " + t.filter + "\n")
	}
	if t.confirm != "" {
		b.WriteString(t.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if t.detail != nil {
		headerLines := strings.Count(b.String(), "\n")
		b.WriteString(t.detailView(width, max(1, height-headerLines)))
		return style.Render(b.String())
	}
	rows := t.visibleRows()
	if len(rows) == 0 {
		b.WriteString("No cached task projects found. Press S to sync or n to create a project.\n")
		return style.Render(b.String())
	}
	headerLines := strings.Count(b.String(), "\n")
	bodyHeight := max(1, height-headerLines)
	listWidth, previewWidth := tasksPaneWidths(width)
	listCol := t.renderList(rows, listWidth, bodyHeight)
	if previewWidth <= 0 {
		b.WriteString(listCol)
		return style.Render(b.String())
	}
	previewCol := t.renderPreview(rows, previewWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(listWidth).Render(listCol), "  ", lipgloss.NewStyle().Width(previewWidth).Render(previewCol))
	b.WriteString(body)
	return style.Render(b.String())
}

func tasksPaneWidths(width int) (int, int) {
	if width < 64 {
		return width, 0
	}
	listWidth := width * 4 / 10
	if listWidth < 28 {
		listWidth = 28
	}
	if listWidth > 50 {
		listWidth = 50
	}
	previewWidth := width - listWidth - 2
	if previewWidth < 24 {
		return width, 0
	}
	return listWidth, previewWidth
}

func (t Tasks) renderList(rows []taskRow, width, height int) string {
	t.ensureList(rows)
	t.rowList.SetSize(width, height)
	return t.rowList.View()
}

func (t Tasks) renderPreview(rows []taskRow, width int) string {
	if t.index < 0 || t.index >= len(rows) {
		return ""
	}
	row := rows[t.index]
	var b strings.Builder
	switch {
	case row.Project != nil:
		b.WriteString(row.Project.Meta.Name + "\n")
		b.WriteString(fmt.Sprintf("Project %d\n", row.Project.Meta.RemoteID))
	case row.Column != nil:
		b.WriteString(row.Column.Name + "\n")
		b.WriteString(fmt.Sprintf("%d linked card(s)\n", len(row.Column.Cards)))
	case row.Card != nil:
		b.WriteString(row.Card.Meta.Title + "\n")
		if updated := formatNotesRelative(row.Card.Meta.RemoteUpdatedAt); updated != "" {
			b.WriteString("Updated " + updated + "\n")
		}
		b.WriteString(strings.Repeat("─", width) + "\n")
		rendered, err := emailtext.RenderMarkdown(row.Card.Body, width)
		if err != nil {
			b.WriteString(row.Card.Body)
		} else {
			b.WriteString(rendered)
		}
	case row.Missing:
		b.WriteString(row.Name + "\nMissing linked card\n")
	}
	return b.String()
}

func (t Tasks) Title() string { return "Tasks" }

func (t Tasks) KeyBindings() []key.Binding {
	return []key.Binding{t.keys.Up, t.keys.Down, t.keys.Open, t.keys.Back, t.keys.Refresh, t.keys.Sync, t.keys.Search, t.keys.Project, t.keys.New, t.keys.Edit, t.keys.Delete}
}

func (t Tasks) Selection() TasksSelection {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		return TasksSelection{Kind: "task-card", HasItem: false}
	}
	return TasksSelection{Kind: "task-card", Subject: row.Card.Meta.Title, HasItem: true}
}

type TasksSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

func (t Tasks) handleAction(action string) (Screen, tea.Cmd) {
	if t.confirm != "" || t.filtering {
		return t, nil
	}
	switch action {
	case "sync":
		if t.sync == nil || t.syncing {
			return t, nil
		}
		t.syncing = true
		t.status = ""
		return t, t.syncCmd()
	case "new-card":
		return t, t.createCardCmd()
	case "new-project":
		return t, t.createProjectCmd()
	case "edit-card":
		return t, t.editCardCmd()
	case "delete-card":
		if row, ok := t.selectedRow(); ok && row.Card != nil {
			t.confirm = "Delete " + row.Card.Meta.Title + "?"
		}
	case "search":
		t.filtering = true
		t.filter = ""
		t.index = 0
		t.syncList()
	case "projects":
		t.project = nil
		t.board = nil
		t.cards = nil
		t.rows = t.buildRows()
		t.index = 0
		t.syncList()
	}
	return t, nil
}

func (t Tasks) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if t.confirm != "" {
		return t.handleConfirmKey(msg)
	}
	if t.filtering {
		return t.handleFilterKey(msg)
	}
	if t.detail != nil {
		if key.Matches(msg, t.keys.Back) {
			t.detail = nil
			t.detailScroll = 0
			return t, nil
		}
		if key.Matches(msg, t.keys.Edit) {
			return t, t.editCachedCardCmd(*t.detail)
		}
		if key.Matches(msg, t.keys.Up) && t.detailScroll > 0 {
			t.detailScroll--
		} else if key.Matches(msg, t.keys.Down) {
			t.detailScroll++
		}
		return t, nil
	}
	rows := t.visibleRows()
	if key.Matches(msg, t.keys.Open) {
		if row, ok := t.selectedRow(); ok {
			if row.Project != nil {
				return t, t.loadCmd(row.Project.Meta.RemoteID)
			}
			if row.Card != nil {
				card := *row.Card
				t.detail = &card
				t.detailScroll = 0
			}
		}
		return t, nil
	}
	switch {
	case key.Matches(msg, t.keys.Back), key.Matches(msg, t.keys.Project):
		return t.handleAction("projects")
	case key.Matches(msg, t.keys.Refresh):
		return t, t.loadCmd(t.currentProjectID())
	case key.Matches(msg, t.keys.Sync):
		return t.handleAction("sync")
	case key.Matches(msg, t.keys.Search):
		return t.handleAction("search")
	case key.Matches(msg, t.keys.New):
		if t.project == nil {
			return t.handleAction("new-project")
		}
		return t.handleAction("new-card")
	case key.Matches(msg, t.keys.Edit):
		return t.handleAction("edit-card")
	case key.Matches(msg, t.keys.Delete):
		return t.handleAction("delete-card")
	default:
		t.ensureList(rows)
		updated, cmd := t.rowList.Update(msg)
		t.rowList = updated
		t.index = t.rowList.GlobalIndex()
		t.clampIndex()
		return t, cmd
	}
}

func (t Tasks) loadCmd(projectID int64) tea.Cmd { return func() tea.Msg { return t.load(projectID) } }

func (t Tasks) load(projectID int64) tasksLoadedMsg {
	projects, err := t.store.ListProjects()
	if err != nil {
		return tasksLoadedMsg{err: err}
	}
	var project *taskstore.CachedProject
	if projectID == 0 && len(projects) == 1 {
		projectID = projects[0].Meta.RemoteID
	}
	if projectID > 0 {
		for i := range projects {
			if projects[i].Meta.RemoteID == projectID {
				project = &projects[i]
				break
			}
		}
	}
	if project == nil {
		return tasksLoadedMsg{projects: projects}
	}
	board, boardErr := t.store.ReadBoard(project.Meta.RemoteID)
	if boardErr != nil && !os.IsNotExist(boardErr) {
		return tasksLoadedMsg{err: boardErr}
	}
	cards, err := t.store.ListCards(project.Meta.RemoteID)
	return tasksLoadedMsg{projects: projects, project: project, board: board, cards: cards, err: err}
}

func (t Tasks) syncCmd() tea.Cmd {
	projectID := t.currentProjectID()
	return func() tea.Msg {
		result, err := t.sync(context.Background())
		loaded := t.load(projectID)
		if err == nil {
			err = loaded.err
		}
		return tasksSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (t Tasks) createProjectCmd() tea.Cmd {
	if t.createProject == nil {
		t.status = "Create project is not configured"
		return nil
	}
	return func() tea.Msg {
		input, err := editTaskProjectTemplate("Untitled")
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		project, err := t.createProject(context.Background(), input)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(project.ID)
		return taskActionFinishedMsg{status: "Created " + project.Name, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) createCardCmd() tea.Cmd {
	if t.createCard == nil {
		t.status = "Create card is not configured"
		return nil
	}
	projectID := t.currentProjectID()
	if projectID == 0 {
		t.status = "Open a project before creating a card"
		return nil
	}
	return func() tea.Msg {
		input, err := editTaskCardTemplate("Untitled", "")
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		card, err := t.createCard(context.Background(), projectID, input)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(projectID)
		return taskActionFinishedMsg{status: "Created " + card.Title, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) editCardCmd() tea.Cmd {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		t.status = "Select a card to edit"
		return nil
	}
	return t.editCachedCardCmd(*row.Card)
}

func (t Tasks) editCachedCardCmd(cached taskstore.CachedCard) tea.Cmd {
	if t.updateCard == nil {
		t.status = "Edit card is not configured"
		return nil
	}
	projectID := cached.Meta.ProjectID
	return func() tea.Msg {
		input, err := editTaskCardTemplate(cached.Meta.Title, cached.Body)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		card, err := t.updateCard(context.Background(), projectID, cached.Meta.RemoteID, input)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(projectID)
		return taskActionFinishedMsg{status: "Updated " + card.Title, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) deleteCardCmd() tea.Cmd {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		return nil
	}
	if t.deleteCard == nil {
		return func() tea.Msg { return taskActionFinishedMsg{err: fmt.Errorf("delete card is not configured")} }
	}
	card := *row.Card
	return func() tea.Msg {
		if err := t.deleteCard(context.Background(), card.Meta.ProjectID, card.Meta.RemoteID); err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(card.Meta.ProjectID)
		return taskActionFinishedMsg{status: "Deleted " + card.Meta.Title, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.filtering = false
		t.filter = ""
		t.index = 0
	case "enter":
		t.filtering = false
	case "backspace":
		if len(t.filter) > 0 {
			t.filter = t.filter[:len(t.filter)-1]
		}
		t.index = 0
	default:
		if msg.Text != "" {
			t.filter += msg.Text
			t.index = 0
		}
	}
	t.syncList()
	return t, nil
}

func (t Tasks) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		t.confirm = ""
		return t, t.deleteCardCmd()
	case "n", "N", "esc":
		t.confirm = ""
		t.status = "Cancelled"
	}
	return t, nil
}

func (t Tasks) buildRows() []taskRow {
	if t.project == nil {
		rows := make([]taskRow, 0, len(t.projects))
		for i := range t.projects {
			project := &t.projects[i]
			rows = append(rows, taskRow{Kind: "project", Name: project.Meta.Name, Project: project})
		}
		return rows
	}
	byID := map[int64]*taskstore.CachedCard{}
	for i := range t.cards {
		byID[t.cards[i].Meta.RemoteID] = &t.cards[i]
	}
	rows := []taskRow{}
	if t.board != nil {
		for i := range t.board.Columns {
			column := &t.board.Columns[i]
			rows = append(rows, taskRow{Kind: "column", Name: column.Name, Column: column})
			for _, link := range column.Cards {
				if link.Card != nil {
					if card := byID[link.Card.ID]; card != nil {
						rows = append(rows, taskRow{Kind: "card", Name: card.Meta.Title, Card: card})
						continue
					}
				}
				rows = append(rows, taskRow{Kind: "missing", Name: link.Title, Missing: true})
			}
		}
	}
	if len(rows) == 0 {
		for i := range t.cards {
			card := &t.cards[i]
			rows = append(rows, taskRow{Kind: "card", Name: card.Meta.Title, Card: card})
		}
	}
	return rows
}

func (t Tasks) visibleRows() []taskRow {
	filter := strings.ToLower(strings.TrimSpace(t.filter))
	if filter == "" {
		return t.rows
	}
	out := make([]taskRow, 0, len(t.rows))
	for _, row := range t.rows {
		if strings.Contains(strings.ToLower(row.Name), filter) || row.Card != nil && strings.Contains(strings.ToLower(row.Card.Body), filter) {
			out = append(out, row)
		}
	}
	return out
}

func (t *Tasks) clampIndex() {
	if t.index >= len(t.visibleRows()) {
		t.index = maxNotesIndex(len(t.visibleRows()))
	}
	if t.index < 0 {
		t.index = 0
	}
}

func (t *Tasks) ensureList(rows []taskRow) {
	if len(t.rowList.Items()) == len(rows) {
		t.rowList.Select(t.clampedIndex(rows))
		return
	}
	t.syncList()
}

func (t *Tasks) syncList() {
	rows := t.visibleRows()
	t.index = t.clampedIndex(rows)
	t.rowList = newTaskList(rows, t.index, t.rowList.Width(), t.rowList.Height())
}

func (t Tasks) clampedIndex(rows []taskRow) int {
	if t.index < 0 || len(rows) == 0 {
		return 0
	}
	if t.index >= len(rows) {
		return len(rows) - 1
	}
	return t.index
}

func newTaskList(rows []taskRow, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, taskListItem{row: row})
	}
	m := list.New(items, taskListDelegate{}, width, height)
	m.SetShowTitle(false)
	m.SetShowFilter(false)
	m.SetFilteringEnabled(false)
	m.SetShowStatusBar(false)
	m.SetShowHelp(false)
	m.DisableQuitKeybindings()
	if len(items) > 0 {
		if selected < 0 {
			selected = 0
		}
		if selected >= len(items) {
			selected = len(items) - 1
		}
		m.Select(selected)
	}
	return m
}

type taskListDelegate struct{}

func (d taskListDelegate) Height() int                         { return 1 }
func (d taskListDelegate) Spacing() int                        { return 0 }
func (d taskListDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d taskListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	taskItem, ok := item.(taskListItem)
	if !ok {
		return
	}
	_, _ = io.WriteString(w, formatTaskRow(taskItem.row, index == m.Index(), m.Width()))
}

func formatTaskRow(row taskRow, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "> "
	}
	glyph := "  "
	switch row.Kind {
	case "project":
		glyph = "▸ "
	case "column":
		glyph = "# "
	case "card":
		glyph = "- "
	case "missing":
		glyph = "! "
	}
	return cursor + glyph + truncate(row.Name, max(0, width-4))
}

func (t Tasks) selectedRow() (taskRow, bool) {
	rows := t.visibleRows()
	if len(rows) == 0 || t.index < 0 || t.index >= len(rows) {
		return taskRow{}, false
	}
	return rows[t.index], true
}

func (t Tasks) currentProjectID() int64 {
	if t.project == nil {
		return 0
	}
	return t.project.Meta.RemoteID
}

func (t Tasks) detailView(width, height int) string {
	if t.detail == nil {
		return ""
	}
	bodyWidth := notesBodyWidth(width)
	var head strings.Builder
	head.WriteString(t.detail.Meta.Title + "\n")
	if updated := formatNotesRelative(t.detail.Meta.RemoteUpdatedAt); updated != "" {
		head.WriteString("Updated " + updated + "\n")
	}
	head.WriteString(strings.Repeat("─", bodyWidth) + "\n")
	rendered, err := emailtext.RenderMarkdown(t.detail.Body, bodyWidth)
	body := rendered
	if err != nil {
		body = fmt.Sprintf("Markdown render error: %v", err)
	}
	bodyLines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	visibleBodyHeight := max(1, height-strings.Count(head.String(), "\n")-2)
	scroll := t.detailScroll
	if scroll < 0 {
		scroll = 0
	}
	maxScroll := max(0, len(bodyLines)-visibleBodyHeight)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := min(len(bodyLines), scroll+visibleBodyHeight)
	visible := bodyLines[scroll:end]
	var b strings.Builder
	b.WriteString(head.String())
	b.WriteString(strings.Join(visible, "\n"))
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("─", bodyWidth) + "\n")
	b.WriteString(detailFooterHint(scroll, len(bodyLines), visibleBodyHeight) + "\n")
	return b.String()
}

func editTaskProjectTemplate(name string) (tasks.ProjectInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-task-project-%d.md", time.Now().UnixNano()))
	defer os.Remove(path)
	if err := os.WriteFile(path, []byte("Name: "+name+"\n"), 0o600); err != nil {
		return tasks.ProjectInput{}, err
	}
	if err := runTaskEditor(path); err != nil {
		return tasks.ProjectInput{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return tasks.ProjectInput{}, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "name:") {
			if parsed := strings.TrimSpace(line[len("Name:"):]); parsed != "" {
				return tasks.ProjectInput{Name: parsed}, nil
			}
		}
	}
	return tasks.ProjectInput{Name: "Untitled"}, nil
}

func editTaskCardTemplate(title, body string) (tasks.CardInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-task-card-%d.md", time.Now().UnixNano()))
	defer os.Remove(path)
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return tasks.CardInput{}, err
	}
	if err := runTaskEditor(path); err != nil {
		return tasks.CardInput{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return tasks.CardInput{}, err
	}
	return parseTaskCardTemplate(string(content)), nil
}

func parseTaskCardTemplate(content string) tasks.CardInput {
	lines := strings.Split(content, "\n")
	title := "Untitled"
	start := 0
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
		if parsed := strings.TrimSpace(lines[0][len("Title:"):]); parsed != "" {
			title = parsed
		}
		start = 1
		if len(lines) > 1 && strings.TrimSpace(lines[1]) == "" {
			start = 2
		}
	}
	return tasks.CardInput{Title: title, Body: strings.Join(lines[start:], "\n")}
}

func runTaskEditor(path string) error {
	editor := tasksEditorCommand()
	if editor == "" {
		return fmt.Errorf("set TELEX_TASKS_EDITOR, VISUAL, or EDITOR to edit tasks")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("set TELEX_TASKS_EDITOR, VISUAL, or EDITOR to edit tasks")
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func tasksEditorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("TELEX_TASKS_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}
