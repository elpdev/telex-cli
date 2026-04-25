package screens

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/theme"
)

type HomeNavigateFunc func(screenID string) tea.Cmd

type Home struct {
	mail     mailstore.Store
	calendar calendarstore.Store
	notes    notestore.Store
	drive    drivestore.Store
	theme    theme.Theme
	navigate HomeNavigateFunc

	summary homeSummary
	loaded  bool
	keys    HomeKeyMap
}

type HomeKeyMap struct {
	Refresh  key.Binding
	Mail     key.Binding
	Calendar key.Binding
	Notes    key.Binding
	Drive    key.Binding
}

type homeSummary struct {
	mail     mailCardData
	calendar calendarCardData
	notes    notesCardData
	drive    driveCardData
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

type homeLoadedMsg struct {
	summary homeSummary
}

func DefaultHomeKeyMap() HomeKeyMap {
	return HomeKeyMap{
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Mail:     key.NewBinding(key.WithKeys("m", "1"), key.WithHelp("m/1", "open mail")),
		Calendar: key.NewBinding(key.WithKeys("c", "2"), key.WithHelp("c/2", "open calendar")),
		Notes:    key.NewBinding(key.WithKeys("n", "3"), key.WithHelp("n/3", "open notes")),
		Drive:    key.NewBinding(key.WithKeys("d", "4"), key.WithHelp("d/4", "open drive")),
	}
}

func NewHome(mail mailstore.Store, calendar calendarstore.Store, notes notestore.Store, drive drivestore.Store, t theme.Theme, navigate HomeNavigateFunc) Home {
	return Home{
		mail:     mail,
		calendar: calendar,
		notes:    notes,
		drive:    drive,
		theme:    t,
		navigate: navigate,
		keys:     DefaultHomeKeyMap(),
	}
}

func (h Home) Reconfigure(t theme.Theme) Home {
	h.theme = t
	return h
}

func (h Home) Init() tea.Cmd { return h.loadCmd() }

func (h Home) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case homeLoadedMsg:
		h.summary = msg.summary
		h.loaded = true
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
	case key.Matches(msg, h.keys.Notes):
		return h.routeTo("notes")
	case key.Matches(msg, h.keys.Drive):
		return h.routeTo("drive")
	}
	return h, nil
}

func (h Home) routeTo(id string) (Screen, tea.Cmd) {
	if h.navigate == nil {
		return h, nil
	}
	return h, h.navigate(id)
}

func (h Home) Title() string { return "Home" }

func (h Home) KeyBindings() []key.Binding {
	return []key.Binding{h.keys.Mail, h.keys.Calendar, h.keys.Notes, h.keys.Drive, h.keys.Refresh}
}

func (h Home) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if !h.loaded {
		return style.Render(h.theme.Muted.Render("Loading dashboard…"))
	}

	header := h.renderHeader(width)

	cardWidth := width / 2
	stacked := width < 100
	if stacked {
		cardWidth = width
	}
	if cardWidth < 32 {
		cardWidth = 32
	}

	mailCard := h.renderMailCard(cardWidth)
	calCard := h.renderCalendarCard(cardWidth)
	notesCard := h.renderNotesCard(cardWidth)
	driveCard := h.renderDriveCard(cardWidth)

	var grid string
	if stacked {
		grid = lipgloss.JoinVertical(lipgloss.Left, mailCard, calCard, notesCard, driveCard)
	} else {
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, mailCard, calCard)
		bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, notesCard, driveCard)
		grid = lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
	}

	footer := h.renderFooter()

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", grid, "", footer)
	return style.Render(body)
}

func (h Home) renderHeader(width int) string {
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
		h.theme.HeaderAccent.Render("m c n d") + h.theme.Muted.Render(" jump"),
		h.theme.HeaderAccent.Render("r") + h.theme.Muted.Render(" refresh"),
	}
	return strings.Join(parts, h.theme.Muted.Render("  •  "))
}

func (h Home) renderMailCard(width int) string {
	d := h.summary.mail
	body := &strings.Builder{}
	body.WriteString(cardTitleRow(h.theme, "MAIL", "m / 1", innerWidth(width)))
	body.WriteString("\n")

	switch {
	case d.err != nil:
		body.WriteString(h.theme.Warn.Render("⚠ mail cache error — see Logs"))
	case !d.hasMailboxes:
		body.WriteString(h.theme.Muted.Render("No mailbox yet — sync from Mail."))
	default:
		counts := []string{fmt.Sprintf("%d unread", d.unread)}
		if d.drafts > 0 {
			counts = append(counts, fmt.Sprintf("%d drafts", d.drafts))
		}
		if d.outbox > 0 {
			counts = append(counts, fmt.Sprintf("%d outbox", d.outbox))
		}
		body.WriteString(h.theme.HeaderAccent.Render(strings.Join(counts, "  ")))
	}
	body.WriteString("\n\n")

	if d.err == nil && d.hasMailboxes {
		if len(d.recent) == 0 {
			body.WriteString(h.theme.Muted.Render("Inbox empty."))
		}
		for _, m := range d.recent {
			subject := m.subject
			if subject == "" {
				subject = "(no subject)"
			}
			label := subject
			if m.from != "" {
				label = m.from + " — " + subject
			}
			body.WriteString(h.renderRow(label, humanAgo(time.Since(m.received))+" ago", innerWidth(width), m.unread))
			body.WriteString("\n")
		}
	}

	return cardFrame(h.theme, width).Render(body.String())
}

func (h Home) renderCalendarCard(width int) string {
	d := h.summary.calendar
	body := &strings.Builder{}
	body.WriteString(cardTitleRow(h.theme, "CALENDAR", "c / 2", innerWidth(width)))
	body.WriteString("\n")

	switch {
	case d.err != nil:
		body.WriteString(h.theme.Warn.Render("⚠ calendar cache error — see Logs"))
	case d.syncedAt.IsZero():
		body.WriteString(h.theme.Muted.Render("No calendar yet — sync from Calendar."))
	default:
		counts := fmt.Sprintf("%d today  %d this week", d.today, d.thisWeek)
		body.WriteString(h.theme.HeaderAccent.Render(counts))
	}
	body.WriteString("\n\n")

	if d.err == nil && !d.syncedAt.IsZero() {
		if len(d.upcoming) == 0 {
			body.WriteString(h.theme.Muted.Render("No upcoming events."))
		}
		now := time.Now()
		for _, occ := range d.upcoming {
			when := formatWhen(occ.StartsAt, occ.AllDay, now)
			body.WriteString(h.renderRow(occ.Title, when, innerWidth(width), false))
			body.WriteString("\n")
		}
	}

	return cardFrame(h.theme, width).Render(body.String())
}

func (h Home) renderNotesCard(width int) string {
	d := h.summary.notes
	body := &strings.Builder{}
	body.WriteString(cardTitleRow(h.theme, "NOTES", "n / 3", innerWidth(width)))
	body.WriteString("\n")

	switch {
	case d.err != nil:
		body.WriteString(h.theme.Warn.Render("⚠ notes cache error — see Logs"))
	case d.syncedAt.IsZero() && d.total == 0:
		body.WriteString(h.theme.Muted.Render("No notes yet — sync from Notes."))
	default:
		counts := fmt.Sprintf("%d notes  %d folders", d.total, d.folders)
		body.WriteString(h.theme.HeaderAccent.Render(counts))
	}
	body.WriteString("\n\n")

	if d.err == nil {
		if len(d.recent) == 0 && d.total > 0 {
			body.WriteString(h.theme.Muted.Render("No recent edits."))
		}
		for _, n := range d.recent {
			title := n.title
			if title == "" {
				title = "(untitled)"
			}
			body.WriteString(h.renderRow(title, humanAgo(time.Since(n.updated))+" ago", innerWidth(width), false))
			body.WriteString("\n")
		}
	}

	return cardFrame(h.theme, width).Render(body.String())
}

func (h Home) renderDriveCard(width int) string {
	d := h.summary.drive
	body := &strings.Builder{}
	body.WriteString(cardTitleRow(h.theme, "DRIVE", "d / 4", innerWidth(width)))
	body.WriteString("\n")

	switch {
	case d.err != nil:
		body.WriteString(h.theme.Warn.Render("⚠ drive cache error — see Logs"))
	case d.syncedAt.IsZero() && d.files == 0:
		body.WriteString(h.theme.Muted.Render("No files yet — sync from Drive."))
	default:
		counts := fmt.Sprintf("%d files  %s", d.files, humanBytes(d.bytes))
		body.WriteString(h.theme.HeaderAccent.Render(counts))
	}
	body.WriteString("\n\n")

	if d.err == nil {
		if len(d.recent) == 0 && d.files > 0 {
			body.WriteString(h.theme.Muted.Render("No recent uploads."))
		}
		for _, f := range d.recent {
			body.WriteString(h.renderRow(f.name, humanAgo(time.Since(f.updated))+" ago", innerWidth(width), false))
			body.WriteString("\n")
		}
	}

	return cardFrame(h.theme, width).Render(body.String())
}

func (h Home) renderRow(left, right string, width int, accent bool) string {
	bullet := h.theme.Muted.Render("• ")
	if accent {
		bullet = h.theme.HeaderAccent.Render("• ")
	}
	rightStyled := h.theme.Muted.Render(right)
	rightW := lipgloss.Width(rightStyled)
	bulletW := lipgloss.Width(bullet)
	avail := width - bulletW - rightW - 1
	if avail < 4 {
		avail = 4
	}
	leftTrunc := truncateLine(left, avail)
	leftW := lipgloss.Width(leftTrunc)
	pad := width - bulletW - leftW - rightW
	if pad < 1 {
		pad = 1
	}
	leftStyled := h.theme.Text.Render(leftTrunc)
	if accent {
		leftStyled = h.theme.Title.Render(leftTrunc)
	}
	return bullet + leftStyled + strings.Repeat(" ", pad) + rightStyled
}

func cardTitleRow(t theme.Theme, title, hint string, width int) string {
	titleStyled := t.Title.Render(title)
	hintStyled := t.HeaderAccent.Render("(" + hint + ")")
	pad := width - lipgloss.Width(titleStyled) - lipgloss.Width(hintStyled)
	if pad < 1 {
		pad = 1
	}
	return titleStyled + strings.Repeat(" ", pad) + hintStyled
}

func cardFrame(t theme.Theme, totalWidth int) lipgloss.Style {
	border := t.Border.GetForeground()
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(totalWidth - 4)
}

func innerWidth(totalWidth int) int { return totalWidth - 4 }

func truncateLine(s string, max int) string {
	if max <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > max {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
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
	return func() tea.Msg {
		return homeLoadedMsg{summary: collectHomeSummary(mail, cal, notes, drive)}
	}
}

func collectHomeSummary(mail mailstore.Store, cal calendarstore.Store, notes notestore.Store, drive drivestore.Store) homeSummary {
	s := homeSummary{
		mail:     collectMailCard(mail),
		calendar: collectCalendarCard(cal),
		notes:    collectNotesCard(notes),
		drive:    collectDriveCard(drive),
	}
	for _, t := range []time.Time{s.mail.syncedAt, s.calendar.syncedAt, s.notes.syncedAt, s.drive.syncedAt} {
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
