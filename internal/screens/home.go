package screens

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/hackernews/pkg/hn"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/components/card"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/taskstore"
	"github.com/elpdev/telex-cli/internal/theme"
)

type HomeNavigateFunc func(screenID string) tea.Cmd

// NewsFetcher returns up to limit top stories. Implementations typically wrap
// hn.Client.TopStories. May be nil — the news card then renders an empty state.
type NewsFetcher func(ctx context.Context, limit int) ([]hn.Item, error)

const (
	homeGridCols     = 2
	homeStackedBelow = 100
	homeNewsLimit    = 3
	homeNewsTimeout  = 4 * time.Second
)

type Home struct {
	mail     mailstore.Store
	calendar calendarstore.Store
	notes    notestore.Store
	drive    drivestore.Store
	tasks    taskstore.Store
	contacts contactstore.Store
	news     NewsFetcher
	theme    theme.Theme
	navigate HomeNavigateFunc

	summary    homeSummary
	cards      []card.Model
	cardIDs    []string
	focusedIdx int
	loaded     bool
	keys       HomeKeyMap
}

type HomeKeyMap struct {
	Refresh    key.Binding
	NextCard   key.Binding
	PrevCard   key.Binding
	OpenCard   key.Binding
	ClearFocus key.Binding
	Mail       key.Binding
	Calendar   key.Binding
	Notes      key.Binding
	Drive      key.Binding
	Contacts   key.Binding
	Tasks      key.Binding
	News       key.Binding
}

type homeSummary struct {
	mail     mailCardData
	calendar calendarCardData
	notes    notesCardData
	drive    driveCardData
	tasks    tasksCardData
	contacts contactsCardData
	news     newsCardData
	lastSync time.Time
}

type mailCardData struct {
	hasMailboxes bool
	unread       int
	drafts       int
	outbox       int
	recent       []mailRecent
	syncedAt     time.Time
	err          error
}

type mailRecent struct {
	subject  string
	from     string
	received time.Time
	unread   bool
}

type calendarCardData struct {
	today    int
	thisWeek int
	upcoming []calendarstore.OccurrenceMeta
	syncedAt time.Time
	err      error
}

type notesCardData struct {
	total    int
	folders  int
	recent   []notesRecent
	syncedAt time.Time
	err      error
}

type notesRecent struct {
	title   string
	updated time.Time
}

type driveCardData struct {
	files    int
	bytes    int64
	recent   []driveRecent
	syncedAt time.Time
	err      error
}

type driveRecent struct {
	name    string
	updated time.Time
}

type tasksCardData struct {
	projects int
	cards    int
	recent   []tasksRecent
	syncedAt time.Time
	err      error
}

type tasksRecent struct {
	title   string
	project string
	updated time.Time
}

type contactsCardData struct {
	total    int
	comms    int
	recent   []contactsRecent
	syncedAt time.Time
	err      error
}

type contactsRecent struct {
	who     string
	subject string
	when    time.Time
}

type newsCardData struct {
	loaded  bool
	recent  []newsRecent
	fetched time.Time
	err     error
}

type newsRecent struct {
	title    string
	score    int
	comments int
	posted   time.Time
}

type homeLoadedMsg struct {
	summary homeSummary
}

type homeNewsLoadedMsg struct {
	news newsCardData
}

func DefaultHomeKeyMap() HomeKeyMap {
	return HomeKeyMap{
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		NextCard:   key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", "next card")),
		PrevCard:   key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("shift+tab", "prev card")),
		OpenCard:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open focused")),
		ClearFocus: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear focus")),
		Mail:       key.NewBinding(key.WithKeys("m", "1"), key.WithHelp("m/1", "open mail")),
		Calendar:   key.NewBinding(key.WithKeys("c", "2"), key.WithHelp("c/2", "open calendar")),
		Contacts:   key.NewBinding(key.WithKeys("o", "3"), key.WithHelp("o/3", "open contacts")),
		Notes:      key.NewBinding(key.WithKeys("n", "4"), key.WithHelp("n/4", "open notes")),
		Tasks:      key.NewBinding(key.WithKeys("t", "5"), key.WithHelp("t/5", "open tasks")),
		Drive:      key.NewBinding(key.WithKeys("d", "6"), key.WithHelp("d/6", "open drive")),
		News:       key.NewBinding(key.WithKeys("w", "7"), key.WithHelp("w/7", "open news")),
	}
}

func NewHome(
	mail mailstore.Store,
	calendar calendarstore.Store,
	notes notestore.Store,
	drive drivestore.Store,
	tasks taskstore.Store,
	contacts contactstore.Store,
	news NewsFetcher,
	t theme.Theme,
	navigate HomeNavigateFunc,
) Home {
	h := Home{
		mail:       mail,
		calendar:   calendar,
		notes:      notes,
		drive:      drive,
		tasks:      tasks,
		contacts:   contacts,
		news:       news,
		theme:      t,
		navigate:   navigate,
		focusedIdx: -1,
		keys:       DefaultHomeKeyMap(),
	}
	h.cards, h.cardIDs = h.buildCards()
	return h
}

func (h Home) Reconfigure(t theme.Theme) Home {
	h.theme = t
	h.cards, h.cardIDs = h.buildCards()
	return h
}

func (h Home) Init() tea.Cmd { return h.loadCmd() }

func (h Home) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case homeLoadedMsg:
		// Preserve any already-loaded news data so a refresh of the main
		// summary doesn't clobber a successful news fetch from earlier.
		news := h.summary.news
		h.summary = msg.summary
		if news.loaded {
			h.summary.news = news
		}
		h.loaded = true
		h.cards, h.cardIDs = h.buildCards()
		return h, nil
	case homeNewsLoadedMsg:
		h.summary.news = msg.news
		h.cards, h.cardIDs = h.buildCards()
		return h, nil
	case tea.KeyPressMsg:
		return h.handleKey(msg)
	}
	return h, nil
}

func (h Home) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, h.keys.Refresh):
		return h, h.loadCmd()
	case key.Matches(msg, h.keys.Mail):
		return h.routeTo("mail")
	case key.Matches(msg, h.keys.Calendar):
		return h.routeTo("calendar")
	case key.Matches(msg, h.keys.Contacts):
		return h.routeTo("contacts")
	case key.Matches(msg, h.keys.Notes):
		return h.routeTo("notes")
	case key.Matches(msg, h.keys.Tasks):
		return h.routeTo("tasks")
	case key.Matches(msg, h.keys.Drive):
		return h.routeTo("drive")
	case key.Matches(msg, h.keys.News):
		return h.routeTo("news")
	case key.Matches(msg, h.keys.NextCard):
		return h.moveFocus(1), nil
	case key.Matches(msg, h.keys.PrevCard):
		return h.moveFocus(-1), nil
	case key.Matches(msg, h.keys.ClearFocus):
		if h.focusedIdx >= 0 {
			return h.setFocus(-1), nil
		}
	case key.Matches(msg, h.keys.OpenCard):
		if h.focusedIdx >= 0 && h.focusedIdx < len(h.cardIDs) {
			return h.routeTo(h.cardIDs[h.focusedIdx])
		}
	}
	return h, nil
}

func (h Home) routeTo(id string) (Screen, tea.Cmd) {
	if h.navigate == nil {
		return h, nil
	}
	return h, h.navigate(id)
}

func (h Home) moveFocus(delta int) Home {
	if len(h.cards) == 0 {
		return h
	}
	next := h.focusedIdx + delta
	if h.focusedIdx < 0 {
		if delta > 0 {
			next = 0
		} else {
			next = len(h.cards) - 1
		}
	}
	if next < 0 {
		next = len(h.cards) - 1
	}
	if next >= len(h.cards) {
		next = 0
	}
	return h.setFocus(next)
}

func (h Home) setFocus(idx int) Home {
	h.focusedIdx = idx
	for i := range h.cards {
		if i == idx {
			h.cards[i] = h.cards[i].Focus()
		} else {
			h.cards[i] = h.cards[i].Blur()
		}
	}
	return h
}

func (h Home) Title() string { return "Home" }

func (h Home) KeyBindings() []key.Binding {
	return []key.Binding{h.keys.Mail, h.keys.Calendar, h.keys.Contacts, h.keys.Notes, h.keys.Tasks, h.keys.Drive, h.keys.News, h.keys.NextCard, h.keys.OpenCard, h.keys.Refresh}
}

func (h Home) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if !h.loaded {
		return style.Render(h.theme.Muted.Render("Loading dashboard…"))
	}

	header := h.renderHeader()

	stacked := width < homeStackedBelow
	cardWidth := width / homeGridCols
	if stacked {
		cardWidth = width
	}
	if cardWidth < 32 {
		cardWidth = 32
	}

	sized := make([]card.Model, len(h.cards))
	orphan := !stacked && len(h.cards)%homeGridCols == 1
	for i, c := range h.cards {
		w := cardWidth
		if orphan && i == len(h.cards)-1 {
			w = width
		}
		sized[i] = c.WithWidth(w)
	}

	var grid string
	if stacked {
		parts := make([]string, len(sized))
		for i, c := range sized {
			parts[i] = c.View()
		}
		grid = lipgloss.JoinVertical(lipgloss.Left, parts...)
	} else {
		rows := []string{}
		for i := 0; i < len(sized); i += homeGridCols {
			end := i + homeGridCols
			if end > len(sized) {
				end = len(sized)
			}
			rowViews := make([]string, end-i)
			for j := i; j < end; j++ {
				rowViews[j-i] = sized[j].View()
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowViews...))
		}
		grid = lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", grid, "", h.renderFooter())
	return style.Render(body)
}

func (h Home) renderHeader() string {
	title := h.theme.Title.Render("Telex")
	var sub string
	if h.summary.lastSync.IsZero() {
		sub = h.theme.Muted.Render("No data cached yet — open a module and press S to sync.")
	} else {
		sub = h.theme.Muted.Render(fmt.Sprintf("Last sync %s ago", humanAgo(time.Since(h.summary.lastSync))))
	}
	return title + "  " + sub
}

func (h Home) renderFooter() string {
	parts := []string{
		h.theme.HeaderAccent.Render("ctrl+k") + h.theme.Muted.Render(" palette"),
		h.theme.HeaderAccent.Render("?") + h.theme.Muted.Render(" help"),
		h.theme.HeaderAccent.Render("tab") + h.theme.Muted.Render(" focus"),
		h.theme.HeaderAccent.Render("enter") + h.theme.Muted.Render(" open"),
		h.theme.HeaderAccent.Render("m c o n t d w") + h.theme.Muted.Render(" jump"),
		h.theme.HeaderAccent.Render("r") + h.theme.Muted.Render(" refresh"),
	}
	return strings.Join(parts, h.theme.Muted.Render("  •  "))
}

func (h Home) buildCards() ([]card.Model, []string) {
	cards := []card.Model{
		h.makeMailCard(),
		h.makeCalendarCard(),
		h.makeContactsCard(),
		h.makeNotesCard(),
		h.makeTasksCard(),
		h.makeDriveCard(),
		h.makeNewsCard(),
	}
	ids := []string{"mail", "calendar", "contacts", "notes", "tasks", "drive", "news"}
	for i := range cards {
		if i == h.focusedIdx {
			cards[i] = cards[i].Focus()
		}
	}
	return cards, ids
}

func (h Home) makeMailCard() card.Model {
	d := h.summary.mail
	c := card.New(h.theme).WithTitle("MAIL").WithKeyHint("m / 1")
	switch {
	case d.err != nil:
		c = c.WithError("mail cache error — see Logs")
	case !d.hasMailboxes:
		c = c.WithEmpty("No mailbox yet — sync from Mail.")
	default:
		counts := []string{fmt.Sprintf("%d unread", d.unread)}
		if d.drafts > 0 {
			counts = append(counts, fmt.Sprintf("%d drafts", d.drafts))
		}
		if d.outbox > 0 {
			counts = append(counts, fmt.Sprintf("%d outbox", d.outbox))
		}
		c = c.WithCounts(counts...)
		if len(d.recent) == 0 {
			c = c.WithEmpty("Inbox empty.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, m := range d.recent {
				subject := m.subject
				if subject == "" {
					subject = "(no subject)"
				}
				left := subject
				if m.from != "" {
					left = m.from + " — " + subject
				}
				rows = append(rows, card.Row{
					Left:   left,
					Right:  humanAgo(time.Since(m.received)) + " ago",
					Accent: m.unread,
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeCalendarCard() card.Model {
	d := h.summary.calendar
	c := card.New(h.theme).WithTitle("CALENDAR").WithKeyHint("c / 2")
	switch {
	case d.err != nil:
		c = c.WithError("calendar cache error — see Logs")
	case d.syncedAt.IsZero():
		c = c.WithEmpty("No calendar yet — sync from Calendar.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d today", d.today), fmt.Sprintf("%d this week", d.thisWeek))
		if len(d.upcoming) == 0 {
			c = c.WithEmpty("No upcoming events.")
		} else {
			now := time.Now()
			rows := make([]card.Row, 0, len(d.upcoming))
			for _, occ := range d.upcoming {
				rows = append(rows, card.Row{
					Left:  occ.Title,
					Right: formatWhen(occ.StartsAt, occ.AllDay, now),
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeNotesCard() card.Model {
	d := h.summary.notes
	c := card.New(h.theme).WithTitle("NOTES").WithKeyHint("n / 4")
	switch {
	case d.err != nil:
		c = c.WithError("notes cache error — see Logs")
	case d.syncedAt.IsZero() && d.total == 0:
		c = c.WithEmpty("No notes yet — sync from Notes.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d notes", d.total), fmt.Sprintf("%d folders", d.folders))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent edits.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, n := range d.recent {
				title := n.title
				if title == "" {
					title = "(untitled)"
				}
				rows = append(rows, card.Row{
					Left:  title,
					Right: humanAgo(time.Since(n.updated)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeDriveCard() card.Model {
	d := h.summary.drive
	c := card.New(h.theme).WithTitle("DRIVE").WithKeyHint("d / 6")
	switch {
	case d.err != nil:
		c = c.WithError("drive cache error — see Logs")
	case d.syncedAt.IsZero() && d.files == 0:
		c = c.WithEmpty("No files yet — sync from Drive.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d files", d.files), humanBytes(d.bytes))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent uploads.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, f := range d.recent {
				rows = append(rows, card.Row{
					Left:  f.name,
					Right: humanAgo(time.Since(f.updated)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeContactsCard() card.Model {
	d := h.summary.contacts
	c := card.New(h.theme).WithTitle("CONTACTS").WithKeyHint("o / 3")
	switch {
	case d.err != nil:
		c = c.WithError("contacts cache error — see Logs")
	case d.syncedAt.IsZero() && d.total == 0:
		c = c.WithEmpty("No contacts yet — sync from Contacts.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d contacts", d.total), fmt.Sprintf("%d comms", d.comms))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent activity.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, r := range d.recent {
				who := r.who
				if who == "" {
					who = "(unknown)"
				}
				left := who
				if r.subject != "" {
					left = who + " — " + r.subject
				}
				rows = append(rows, card.Row{
					Left:  left,
					Right: humanAgo(time.Since(r.when)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeTasksCard() card.Model {
	d := h.summary.tasks
	c := card.New(h.theme).WithTitle("TASKS").WithKeyHint("t / 5")
	switch {
	case d.err != nil:
		c = c.WithError("tasks cache error — see Logs")
	case d.syncedAt.IsZero() && d.projects == 0:
		c = c.WithEmpty("No tasks yet — sync from Tasks.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d projects", d.projects), fmt.Sprintf("%d cards", d.cards))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent updates.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, r := range d.recent {
				title := r.title
				if title == "" {
					title = "(untitled)"
				}
				left := title
				if r.project != "" {
					left = r.project + " — " + title
				}
				rows = append(rows, card.Row{
					Left:  left,
					Right: humanAgo(time.Since(r.updated)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeNewsCard() card.Model {
	d := h.summary.news
	c := card.New(h.theme).WithTitle("NEWS").WithKeyHint("w / 7")
	switch {
	case d.err != nil:
		c = c.WithError("news fetch failed — check connection")
	case !d.loaded:
		c = c.WithEmpty("Loading top stories…")
	case len(d.recent) == 0:
		c = c.WithEmpty("No top stories.")
	default:
		c = c.WithCounts("Top stories", "fetched "+humanAgo(time.Since(d.fetched))+" ago")
		rows := make([]card.Row, 0, len(d.recent))
		for _, r := range d.recent {
			title := r.title
			if title == "" {
				title = "(untitled)"
			}
			rows = append(rows, card.Row{
				Left:  title,
				Right: fmt.Sprintf("↑%d", r.score),
			})
		}
		c = c.WithRows(rows)
	}
	return c
}

func humanAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
}

func humanBytes(n int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func formatWhen(t time.Time, allDay bool, now time.Time) string {
	if allDay {
		if sameDay(t, now) {
			return "today"
		}
		if sameDay(t, now.AddDate(0, 0, 1)) {
			return "tomorrow"
		}
		return t.Format("Mon Jan 2")
	}
	if sameDay(t, now) {
		return t.Format("15:04")
	}
	if sameDay(t, now.AddDate(0, 0, 1)) {
		return "tmrw " + t.Format("15:04")
	}
	return t.Format("Mon 15:04")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func (h Home) loadCmd() tea.Cmd {
	mail := h.mail
	cal := h.calendar
	notes := h.notes
	drive := h.drive
	tasks := h.tasks
	contacts := h.contacts
	summaryCmd := func() tea.Msg {
		return homeLoadedMsg{summary: collectHomeSummary(mail, cal, notes, drive, tasks, contacts)}
	}
	return tea.Batch(summaryCmd, h.newsLoadCmd())
}

func (h Home) newsLoadCmd() tea.Cmd {
	fetcher := h.news
	return func() tea.Msg {
		return homeNewsLoadedMsg{news: collectNewsCard(fetcher)}
	}
}

func collectHomeSummary(
	mail mailstore.Store,
	cal calendarstore.Store,
	notes notestore.Store,
	drive drivestore.Store,
	tasks taskstore.Store,
	contacts contactstore.Store,
) homeSummary {
	s := homeSummary{
		mail:     collectMailCard(mail),
		calendar: collectCalendarCard(cal),
		notes:    collectNotesCard(notes),
		drive:    collectDriveCard(drive),
		tasks:    collectTasksCard(tasks),
		contacts: collectContactsCard(contacts),
	}
	for _, t := range []time.Time{
		s.mail.syncedAt,
		s.calendar.syncedAt,
		s.notes.syncedAt,
		s.drive.syncedAt,
		s.tasks.syncedAt,
		s.contacts.syncedAt,
	} {
		if t.After(s.lastSync) {
			s.lastSync = t
		}
	}
	return s
}

func collectMailCard(store mailstore.Store) mailCardData {
	boxes, err := store.ListMailboxes()
	if err != nil {
		return mailCardData{err: err}
	}
	data := mailCardData{hasMailboxes: len(boxes) > 0}
	var inbox []mailstore.CachedMessage
	for _, box := range boxes {
		if box.SyncedAt.After(data.syncedAt) {
			data.syncedAt = box.SyncedAt
		}
		path, perr := store.MailboxPath(box.DomainName, box.LocalPart)
		if perr != nil {
			continue
		}
		messages, _ := mailstore.ListInbox(path)
		for _, m := range messages {
			if !m.Meta.Read {
				data.unread++
			}
		}
		inbox = append(inbox, messages...)
		drafts, _ := mailstore.ListMessages(path, "drafts")
		data.drafts += len(drafts)
		outbox, _ := mailstore.ListMessages(path, "outbox")
		data.outbox += len(outbox)
	}
	sort.Slice(inbox, func(i, j int) bool {
		return inbox[i].Meta.ReceivedAt.After(inbox[j].Meta.ReceivedAt)
	})
	limit := 3
	if len(inbox) < limit {
		limit = len(inbox)
	}
	for _, m := range inbox[:limit] {
		from := m.Meta.FromName
		if from == "" {
			from = m.Meta.FromAddress
		}
		data.recent = append(data.recent, mailRecent{
			subject:  m.Meta.Subject,
			from:     from,
			received: m.Meta.ReceivedAt,
			unread:   !m.Meta.Read,
		})
	}
	return data
}

func collectCalendarCard(store calendarstore.Store) calendarCardData {
	occ, err := store.ListOccurrences()
	if err != nil {
		return calendarCardData{err: err}
	}
	data := calendarCardData{}
	now := time.Now()
	weekEnd := now.AddDate(0, 0, 7)
	for _, o := range occ {
		if o.SyncedAt.After(data.syncedAt) {
			data.syncedAt = o.SyncedAt
		}
		if sameDay(o.StartsAt, now) {
			data.today++
		}
		if !o.StartsAt.Before(now) && o.StartsAt.Before(weekEnd) {
			data.thisWeek++
		}
		if !o.StartsAt.Before(now) && len(data.upcoming) < 3 {
			data.upcoming = append(data.upcoming, o)
		}
	}
	return data
}

func collectNotesCard(store notestore.Store) notesCardData {
	data := notesCardData{}
	if total, folders, err := store.Counts(); err == nil {
		data.total = total
		data.folders = folders
	} else {
		data.err = err
	}
	all, err := store.AllNotes()
	if err != nil {
		if data.err == nil {
			data.err = err
		}
		return data
	}
	for _, n := range all {
		if n.Meta.SyncedAt.After(data.syncedAt) {
			data.syncedAt = n.Meta.SyncedAt
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Meta.RemoteUpdatedAt.After(all[j].Meta.RemoteUpdatedAt)
	})
	limit := 3
	if len(all) < limit {
		limit = len(all)
	}
	for _, n := range all[:limit] {
		data.recent = append(data.recent, notesRecent{title: n.Meta.Title, updated: n.Meta.RemoteUpdatedAt})
	}
	return data
}

func collectDriveCard(store drivestore.Store) driveCardData {
	files, err := store.AllFiles()
	if err != nil {
		return driveCardData{err: err}
	}
	data := driveCardData{files: len(files)}
	for _, f := range files {
		data.bytes += f.ByteSize
		if f.SyncedAt.After(data.syncedAt) {
			data.syncedAt = f.SyncedAt
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].RemoteUpdatedAt.After(files[j].RemoteUpdatedAt)
	})
	limit := 3
	if len(files) < limit {
		limit = len(files)
	}
	for _, f := range files[:limit] {
		data.recent = append(data.recent, driveRecent{name: f.Filename, updated: f.RemoteUpdatedAt})
	}
	return data
}

func collectTasksCard(store taskstore.Store) tasksCardData {
	projects, err := store.ListProjects()
	if err != nil {
		return tasksCardData{err: err}
	}
	data := tasksCardData{projects: len(projects)}
	type recent struct {
		title   string
		project string
		updated time.Time
	}
	var all []recent
	for _, p := range projects {
		if p.Meta.SyncedAt.After(data.syncedAt) {
			data.syncedAt = p.Meta.SyncedAt
		}
		cards, _ := store.ListCards(p.Meta.RemoteID)
		data.cards += len(cards)
		for _, c := range cards {
			all = append(all, recent{title: c.Meta.Title, project: p.Meta.Name, updated: c.Meta.RemoteUpdatedAt})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].updated.After(all[j].updated) })
	limit := 3
	if len(all) < limit {
		limit = len(all)
	}
	for _, r := range all[:limit] {
		data.recent = append(data.recent, tasksRecent{title: r.title, project: r.project, updated: r.updated})
	}
	return data
}

func collectContactsCard(store contactstore.Store) contactsCardData {
	contacts, err := store.ListContacts()
	if err != nil {
		return contactsCardData{err: err}
	}
	data := contactsCardData{total: len(contacts)}
	type recent struct {
		who     string
		subject string
		when    time.Time
	}
	var all []recent
	for _, ct := range contacts {
		if ct.Meta.SyncedAt.After(data.syncedAt) {
			data.syncedAt = ct.Meta.SyncedAt
		}
		data.comms += len(ct.Communications)
		for _, comm := range ct.Communications {
			all = append(all, recent{who: ct.Meta.DisplayName, subject: comm.Subject, when: comm.OccurredAt})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].when.After(all[j].when) })
	if len(all) == 0 && len(contacts) > 0 {
		sorted := append([]contactstore.CachedContact(nil), contacts...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Meta.RemoteUpdatedAt.After(sorted[j].Meta.RemoteUpdatedAt)
		})
		for i, ct := range sorted {
			if i >= 3 {
				break
			}
			all = append(all, recent{who: ct.Meta.DisplayName, subject: "added", when: ct.Meta.RemoteUpdatedAt})
		}
	}
	limit := 3
	if len(all) < limit {
		limit = len(all)
	}
	for _, r := range all[:limit] {
		data.recent = append(data.recent, contactsRecent{who: r.who, subject: r.subject, when: r.when})
	}
	return data
}

func collectNewsCard(fetcher NewsFetcher) newsCardData {
	if fetcher == nil {
		return newsCardData{loaded: true, fetched: time.Now()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), homeNewsTimeout)
	defer cancel()
	items, err := fetcher(ctx, homeNewsLimit)
	if err != nil {
		return newsCardData{loaded: true, fetched: time.Now(), err: err}
	}
	data := newsCardData{loaded: true, fetched: time.Now()}
	for _, item := range items {
		data.recent = append(data.recent, newsRecent{
			title:    item.Title,
			score:    item.Score,
			comments: item.Descendants,
			posted:   item.CreatedAt(),
		})
	}
	return data
}
