package screens

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/emailtext"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
)

type NotesSyncFunc func(context.Context) (NotesSyncResult, error)
type CreateNoteFunc func(context.Context, notes.NoteInput) (*notes.Note, error)
type UpdateNoteFunc func(context.Context, int64, notes.NoteInput) (*notes.Note, error)
type DeleteNoteFunc func(context.Context, int64) error

type NotesSyncResult struct {
	Folders int
	Notes   int
}

type Notes struct {
	store   notestore.Store
	sync    NotesSyncFunc
	create  CreateNoteFunc
	update  UpdateNoteFunc
	delete  DeleteNoteFunc
	tree    *notes.FolderTree
	folder  *notes.FolderTree
	notes   []notestore.CachedNote
	rows    []noteRow
	index   int
	detail  *notestore.CachedNote
	filter  string
	editing bool
	confirm string
	loading bool
	syncing bool
	err     error
	status  string
	keys    NotesKeyMap
}

type NotesKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Search  key.Binding
	New     key.Binding
	Edit    key.Binding
	Delete  key.Binding
}

type noteRow struct {
	Kind   string
	Name   string
	Folder *notes.FolderTree
	Note   *notestore.CachedNote
}

type notesLoadedMsg struct {
	tree   *notes.FolderTree
	folder *notes.FolderTree
	notes  []notestore.CachedNote
	err    error
}

type notesSyncedMsg struct {
	result NotesSyncResult
	loaded notesLoadedMsg
	err    error
}

type noteActionFinishedMsg struct {
	status string
	loaded notesLoadedMsg
	err    error
}

type NotesActionMsg struct{ Action string }

func NewNotes(store notestore.Store, sync NotesSyncFunc) Notes {
	return Notes{store: store, sync: sync, loading: true, keys: DefaultNotesKeyMap()}
}

func (n Notes) WithActions(create CreateNoteFunc, update UpdateNoteFunc, delete DeleteNoteFunc) Notes {
	n.create = create
	n.update = update
	n.delete = delete
	return n
}

func DefaultNotesKeyMap() NotesKeyMap {
	return NotesKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync notes")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new note")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit note")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete note")),
	}
}

func (n Notes) Init() tea.Cmd { return n.loadCmd(0) }

func (n Notes) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case notesLoadedMsg:
		n.loading = false
		n.err = msg.err
		if msg.err == nil {
			n.tree = msg.tree
			n.folder = msg.folder
			n.notes = msg.notes
			n.rows = n.buildRows()
			n.clampIndex()
		}
		return n, nil
	case notesSyncedMsg:
		n.syncing = false
		n.err = msg.err
		if msg.err == nil {
			n.status = fmt.Sprintf("Synced %d folder(s), %d note(s)", msg.result.Folders, msg.result.Notes)
			n.tree = msg.loaded.tree
			n.folder = msg.loaded.folder
			n.notes = msg.loaded.notes
			n.rows = n.buildRows()
			n.clampIndex()
		} else {
			n.status = ""
		}
		return n, nil
	case noteActionFinishedMsg:
		n.loading = false
		n.err = msg.err
		if msg.err != nil {
			n.status = fmt.Sprintf("Notes action failed: %v", msg.err)
			return n, nil
		}
		n.status = msg.status
		n.tree = msg.loaded.tree
		n.folder = msg.loaded.folder
		n.notes = msg.loaded.notes
		n.rows = n.buildRows()
		n.clampIndex()
		return n, nil
	case NotesActionMsg:
		return n.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return n.handleKey(msg)
	}
	return n, nil
}

func (n Notes) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if n.loading {
		return style.Render("Loading local notes cache...")
	}
	if n.err != nil {
		return style.Render(fmt.Sprintf("Notes cache error: %v\n\nRun `telex notes sync` or press S to populate Notes.", n.err))
	}
	var b strings.Builder
	b.WriteString("Notes")
	if n.folder != nil {
		b.WriteString(" / " + n.breadcrumb())
	}
	b.WriteString("\n")
	if n.status != "" {
		b.WriteString(n.status + "\n")
	}
	if n.syncing {
		b.WriteString("Syncing remote Notes...\n")
	}
	if n.editing {
		b.WriteString("Filter: " + n.filter + "\n")
	}
	if n.confirm != "" {
		b.WriteString(n.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if n.detail != nil {
		b.WriteString(n.detailView(width))
		return style.Render(b.String())
	}
	rows := n.visibleRows()
	if len(rows) == 0 {
		b.WriteString("No cached notes found. Press S to sync or n to create a note.\n")
		return style.Render(b.String())
	}
	for i, row := range rows {
		cursor := "  "
		if i == n.index {
			cursor = "> "
		}
		kind := "note"
		extra := ""
		if row.Kind == "folder" {
			kind = "dir "
			extra = fmt.Sprintf(" (%d)", row.Folder.NoteCount)
		}
		b.WriteString(fmt.Sprintf("%s%s  %s%s\n", cursor, kind, row.Name, extra))
	}
	return style.Render(b.String())
}

func (n Notes) Title() string { return "Notes" }

func (n Notes) KeyBindings() []key.Binding {
	return []key.Binding{n.keys.Up, n.keys.Down, n.keys.Open, n.keys.Back, n.keys.Refresh, n.keys.Sync, n.keys.Search, n.keys.New, n.keys.Edit, n.keys.Delete}
}

func (n Notes) Selection() NotesSelection {
	row, ok := n.selectedRow()
	if !ok || row.Note == nil {
		return NotesSelection{Kind: "note", HasItem: false}
	}
	return NotesSelection{Kind: "note", Subject: row.Note.Meta.Title, HasItem: true}
}

type NotesSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

func (n Notes) handleAction(action string) (Screen, tea.Cmd) {
	if n.confirm != "" || n.editing {
		return n, nil
	}
	switch action {
	case "sync":
		if n.sync == nil || n.syncing {
			return n, nil
		}
		n.syncing = true
		n.status = ""
		return n, n.syncCmd()
	case "new":
		return n, n.createCmd()
	case "edit":
		return n, n.editCmd()
	case "delete":
		if row, ok := n.selectedRow(); ok && row.Note != nil {
			n.confirm = "Delete " + row.Note.Meta.Title + "?"
		}
	case "search":
		n.editing = true
		n.filter = ""
		n.index = 0
	}
	return n, nil
}

func (n Notes) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if n.confirm != "" {
		return n.handleConfirmKey(msg)
	}
	if n.editing {
		return n.handleFilterKey(msg)
	}
	if n.detail != nil {
		if key.Matches(msg, n.keys.Back) {
			n.detail = nil
		}
		if key.Matches(msg, n.keys.Edit) {
			return n, n.editCachedCmd(*n.detail)
		}
		return n, nil
	}
	rows := n.visibleRows()
	switch {
	case key.Matches(msg, n.keys.Up):
		if n.index > 0 {
			n.index--
		}
	case key.Matches(msg, n.keys.Down):
		if n.index < len(rows)-1 {
			n.index++
		}
	case key.Matches(msg, n.keys.Open):
		if len(rows) == 0 {
			return n, nil
		}
		row := rows[n.index]
		if row.Folder != nil {
			n.index = 0
			return n, n.loadCmd(row.Folder.ID)
		}
		if row.Note != nil {
			note := *row.Note
			n.detail = &note
		}
	case key.Matches(msg, n.keys.Back):
		if n.folder != nil && n.folder.ParentID != nil {
			n.index = 0
			return n, n.loadCmd(*n.folder.ParentID)
		}
	case key.Matches(msg, n.keys.Refresh):
		return n, n.loadCmd(n.currentFolderID())
	case key.Matches(msg, n.keys.Sync):
		if n.sync == nil || n.syncing {
			return n, nil
		}
		n.syncing = true
		n.status = ""
		return n, n.syncCmd()
	case key.Matches(msg, n.keys.Search):
		n.editing = true
		n.filter = ""
		n.index = 0
	case key.Matches(msg, n.keys.New):
		return n, n.createCmd()
	case key.Matches(msg, n.keys.Edit):
		return n, n.editCmd()
	case key.Matches(msg, n.keys.Delete):
		if row, ok := n.selectedRow(); ok && row.Note != nil {
			n.confirm = "Delete " + row.Note.Meta.Title + "?"
		}
	}
	return n, nil
}

func (n Notes) loadCmd(folderID int64) tea.Cmd {
	return func() tea.Msg { return n.load(folderID) }
}

func (n Notes) load(folderID int64) notesLoadedMsg {
	tree, err := n.store.FolderTree()
	if err != nil {
		return notesLoadedMsg{err: err}
	}
	folder := findNotesFolder(tree, folderID)
	if folder == nil {
		folder = tree
	}
	cached, err := n.store.ListNotes(folder.ID)
	return notesLoadedMsg{tree: tree, folder: folder, notes: cached, err: err}
}

func (n Notes) syncCmd() tea.Cmd {
	folderID := n.currentFolderID()
	return func() tea.Msg {
		result, err := n.sync(context.Background())
		loaded := n.load(folderID)
		if err == nil {
			err = loaded.err
		}
		return notesSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (n Notes) createCmd() tea.Cmd {
	if n.create == nil {
		n.status = "Create is not configured"
		return nil
	}
	folderID := n.currentFolderID()
	return func() tea.Msg {
		input, err := editNoteTemplate("Untitled", "")
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		if folderID > 0 {
			input.FolderID = &folderID
		}
		created, err := n.create(context.Background(), input)
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		loaded := n.load(folderID)
		return noteActionFinishedMsg{status: "Created " + created.Title, loaded: loaded, err: loaded.err}
	}
}

func (n Notes) editCmd() tea.Cmd {
	row, ok := n.selectedRow()
	if !ok || row.Note == nil {
		n.status = "Select a note to edit"
		return nil
	}
	return n.editCachedCmd(*row.Note)
}

func (n Notes) editCachedCmd(cached notestore.CachedNote) tea.Cmd {
	if n.update == nil {
		n.status = "Edit is not configured"
		return nil
	}
	folderID := n.currentFolderID()
	return func() tea.Msg {
		input, err := editNoteTemplate(cached.Meta.Title, cached.Body)
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		if cached.Meta.FolderID > 0 {
			input.FolderID = &cached.Meta.FolderID
		}
		updated, err := n.update(context.Background(), cached.Meta.RemoteID, input)
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		loaded := n.load(folderID)
		return noteActionFinishedMsg{status: "Updated " + updated.Title, loaded: loaded, err: loaded.err}
	}
}

func (n Notes) deleteCmd() tea.Cmd {
	row, ok := n.selectedRow()
	if !ok || row.Note == nil {
		return nil
	}
	if n.delete == nil {
		return func() tea.Msg { return noteActionFinishedMsg{err: fmt.Errorf("delete is not configured")} }
	}
	folderID := n.currentFolderID()
	note := *row.Note
	return func() tea.Msg {
		if err := n.delete(context.Background(), note.Meta.RemoteID); err != nil {
			return noteActionFinishedMsg{err: err}
		}
		loaded := n.load(folderID)
		return noteActionFinishedMsg{status: "Deleted " + note.Meta.Title, loaded: loaded, err: loaded.err}
	}
}

func (n Notes) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		n.editing = false
		n.filter = ""
		n.index = 0
	case "enter":
		n.editing = false
	case "backspace":
		if len(n.filter) > 0 {
			n.filter = n.filter[:len(n.filter)-1]
		}
		n.index = 0
	default:
		if msg.Text != "" {
			n.filter += msg.Text
			n.index = 0
		}
	}
	return n, nil
}

func (n Notes) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		n.confirm = ""
		return n, n.deleteCmd()
	case "n", "N", "esc":
		n.confirm = ""
		n.status = "Cancelled"
	}
	return n, nil
}

func (n Notes) buildRows() []noteRow {
	if n.folder == nil {
		return nil
	}
	rows := make([]noteRow, 0, len(n.folder.Children)+len(n.notes))
	for i := range n.folder.Children {
		folder := &n.folder.Children[i]
		rows = append(rows, noteRow{Kind: "folder", Name: folder.Name, Folder: folder})
	}
	for i := range n.notes {
		note := &n.notes[i]
		rows = append(rows, noteRow{Kind: "note", Name: note.Meta.Title, Note: note})
	}
	return rows
}

func (n Notes) visibleRows() []noteRow {
	filter := strings.ToLower(strings.TrimSpace(n.filter))
	if filter == "" {
		return n.rows
	}
	out := make([]noteRow, 0, len(n.rows))
	for _, row := range n.rows {
		if strings.Contains(strings.ToLower(row.Name), filter) {
			out = append(out, row)
		}
	}
	return out
}

func (n *Notes) clampIndex() {
	if n.index >= len(n.visibleRows()) {
		n.index = maxNotesIndex(len(n.visibleRows()))
	}
}

func (n Notes) selectedRow() (noteRow, bool) {
	rows := n.visibleRows()
	if len(rows) == 0 || n.index < 0 || n.index >= len(rows) {
		return noteRow{}, false
	}
	return rows[n.index], true
}

func (n Notes) currentFolderID() int64 {
	if n.folder == nil {
		return 0
	}
	return n.folder.ID
}

func (n Notes) breadcrumb() string {
	if n.tree == nil || n.folder == nil {
		return ""
	}
	paths := notesFolderPath(n.tree, n.folder.ID, nil)
	if len(paths) == 0 {
		return n.folder.Name
	}
	return strings.Join(paths, " / ")
}

func (n Notes) detailView(width int) string {
	if n.detail == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(n.detail.Meta.Title + "\n")
	b.WriteString(fmt.Sprintf("ID: %d\nFolder: %s\nUpdated: %s\nPath: %s\n\n", n.detail.Meta.RemoteID, formatNotesID(n.detail.Meta.FolderID), n.detail.Meta.RemoteUpdatedAt.Format(time.RFC3339), n.detail.Path))
	rendered, err := emailtext.RenderMarkdown(n.detail.Body, notesBodyWidth(width))
	if err != nil {
		b.WriteString(fmt.Sprintf("Markdown render error: %v", err))
	} else {
		b.WriteString(rendered)
	}
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func notesBodyWidth(width int) int {
	if width < 24 {
		return 20
	}
	return width - 4
}

func findNotesFolder(tree *notes.FolderTree, id int64) *notes.FolderTree {
	if tree == nil {
		return nil
	}
	if id == 0 || tree.ID == id {
		return tree
	}
	for i := range tree.Children {
		if found := findNotesFolder(&tree.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}

func notesFolderPath(tree *notes.FolderTree, id int64, path []string) []string {
	if tree == nil {
		return nil
	}
	path = append(path, tree.Name)
	if tree.ID == id {
		return path
	}
	for i := range tree.Children {
		if found := notesFolderPath(&tree.Children[i], id, append([]string{}, path...)); len(found) > 0 {
			return found
		}
	}
	return nil
}

func editNoteTemplate(title, body string) (notes.NoteInput, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("telex-note-%d.md", time.Now().UnixNano()))
	defer os.Remove(path)
	if err := os.WriteFile(path, []byte(fmt.Sprintf("Title: %s\n\n%s", title, body)), 0o600); err != nil {
		return notes.NoteInput{}, err
	}
	editor := notesEditorCommand()
	if editor == "" {
		return notes.NoteInput{}, fmt.Errorf("set TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit notes")
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return notes.NoteInput{}, fmt.Errorf("set TELEX_NOTES_EDITOR, VISUAL, or EDITOR to edit notes")
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return notes.NoteInput{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return notes.NoteInput{}, err
	}
	return parseNoteTemplate(string(content)), nil
}

func notesEditorCommand() string {
	if editor := strings.TrimSpace(os.Getenv("TELEX_NOTES_EDITOR")); editor != "" {
		return editor
	}
	if editor := strings.TrimSpace(os.Getenv("VISUAL")); editor != "" {
		return editor
	}
	return strings.TrimSpace(os.Getenv("EDITOR"))
}

func parseNoteTemplate(content string) notes.NoteInput {
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
	return notes.NoteInput{Title: title, Body: strings.Join(lines[start:], "\n")}
}

func formatNotesID(id int64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}

func maxNotesIndex(length int) int {
	if length <= 0 {
		return 0
	}
	return length - 1
}
