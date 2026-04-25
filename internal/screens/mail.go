package screens

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/articletext"
	"github.com/elpdev/telex-cli/internal/emailtext"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

type mailMode int

const (
	mailModeList mailMode = iota
	mailModeDetail
	mailModeLinks
	mailModeArticle
	mailReadWidth = 100
)

var extractArticleURL = articletext.NewExtractor().ExtractURL

type Mail struct {
	store         mailstore.Store
	mailboxes     []mailstore.MailboxMeta
	mailboxIndex  int
	messages      []mailstore.CachedMessage
	messageIndex  int
	detailScroll  int
	links         []emailtext.Link
	linkIndex     int
	article       string
	articleURL    string
	articleScroll int
	mode          mailMode
	loading       bool
	err           error
	status        string
	keys          MailKeyMap
}

type MailKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Previous key.Binding
	Next     key.Binding
	Open     key.Binding
	OpenHTML key.Binding
	Links    key.Binding
	Extract  key.Binding
	Copy     key.Binding
	Back     key.Binding
	Refresh  key.Binding
}

type mailLoadedMsg struct {
	mailboxes []mailstore.MailboxMeta
	messages  []mailstore.CachedMessage
	err       error
}

type htmlOpenFinishedMsg struct {
	path string
	err  error
}

type linkOpenFinishedMsg struct {
	url string
	err error
}

type linkCopyFinishedMsg struct {
	url string
	err error
}

type articleExtractedMsg struct {
	url     string
	article string
	err     error
}

func NewMail(store mailstore.Store) Mail {
	return Mail{store: store, keys: DefaultMailKeyMap(), loading: true}
}

func DefaultMailKeyMap() MailKeyMap {
	return MailKeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "message up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "message down")),
		Previous: key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("left/h", "mailbox prev")),
		Next:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("right/l", "mailbox next")),
		Open:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		OpenHTML: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open html")),
		Links:    key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "links")),
		Extract:  key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "extract")),
		Copy:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy link")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
	}
}

func (m Mail) Init() tea.Cmd { return m.loadCmd() }

func (m Mail) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case mailLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.status = ""
		if msg.err == nil {
			m.mailboxes = msg.mailboxes
			m.messages = msg.messages
			m.clampSelection()
		}
		return m, nil
	case htmlOpenFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not open HTML: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Opened HTML: %s", msg.path)
		}
		return m, nil
	case linkOpenFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not open link: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Opened link: %s", msg.url)
		}
		return m, nil
	case linkCopyFinishedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not copy link: %v", msg.err)
		} else {
			m.status = "Copied link"
		}
		return m, nil
	case articleExtractedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not extract article: %v", msg.err)
			return m, nil
		}
		m.article = msg.article
		m.articleURL = msg.url
		m.articleScroll = 0
		m.status = ""
		m.mode = mailModeArticle
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Mail) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if m.loading {
		return style.Render("Loading local mail cache...")
	}
	if m.err != nil {
		return style.Render(fmt.Sprintf("Mail cache error: %v\n\nRun `telex sync` to create local mail data.", m.err))
	}
	if len(m.mailboxes) == 0 {
		return style.Render("No synced mailboxes found.\n\nRun `telex sync` to populate the local mail cache.")
	}
	if m.mode == mailModeArticle && len(m.messages) > 0 {
		return style.Render(m.articleView(width, height))
	}
	if m.mode == mailModeLinks && len(m.messages) > 0 {
		return style.Render(m.linksView(width, height))
	}
	if m.mode == mailModeDetail && len(m.messages) > 0 {
		return style.Render(m.detailView(width, height))
	}
	return style.Render(m.listView(width, height))
}

func (m Mail) Title() string { return "Mail" }

func (m Mail) KeyBindings() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Previous, m.keys.Next, m.keys.Open, m.keys.OpenHTML, m.keys.Links, m.keys.Extract, m.keys.Copy, m.keys.Back, m.keys.Refresh}
}

func (m Mail) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Refresh) {
		m.loading = true
		return m, m.loadCmd()
	}
	if m.mode == mailModeArticle {
		return m.handleArticleKey(msg)
	}
	if m.mode == mailModeLinks {
		return m.handleLinksKey(msg)
	}
	if m.mode == mailModeDetail {
		if key.Matches(msg, m.keys.Back) {
			m.mode = mailModeList
			m.detailScroll = 0
			m.status = ""
			return m, nil
		}
		if key.Matches(msg, m.keys.OpenHTML) {
			return m.openHTML()
		}
		if key.Matches(msg, m.keys.Links) {
			m.links = emailtext.Links(m.messages[m.messageIndex].BodyText, m.messages[m.messageIndex].BodyHTML)
			m.linkIndex = 0
			m.mode = mailModeLinks
			if len(m.links) == 0 {
				m.status = "No links found in this message"
			}
			return m, nil
		}
		maxScroll := m.maxDetailScroll()
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		case key.Matches(msg, m.keys.Down):
			if m.detailScroll < maxScroll {
				m.detailScroll++
			}
		}
		return m, nil
	}
	if len(m.mailboxes) == 0 {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Previous):
		if m.mailboxIndex > 0 {
			m.mailboxIndex--
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.Next):
		if m.mailboxIndex < len(m.mailboxes)-1 {
			m.mailboxIndex++
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.Up):
		if m.messageIndex > 0 {
			m.messageIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.messageIndex < len(m.messages)-1 {
			m.messageIndex++
		}
	case key.Matches(msg, m.keys.Open):
		if len(m.messages) > 0 {
			m.mode = mailModeDetail
			m.detailScroll = 0
			m.status = ""
		}
	case key.Matches(msg, m.keys.Back):
		return m, nil
	}
	return m, nil
}

func (m Mail) handleLinksKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = mailModeDetail
		return m, nil
	}
	if len(m.links) == 0 {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.linkIndex > 0 {
			m.linkIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.linkIndex < len(m.links)-1 {
			m.linkIndex++
		}
	case key.Matches(msg, m.keys.Open):
		return m.openLink()
	case key.Matches(msg, m.keys.Copy):
		return m.copyLink()
	case key.Matches(msg, m.keys.Extract):
		return m.extractLink()
	}
	return m, nil
}

func (m Mail) handleArticleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = mailModeLinks
		return m, nil
	}
	maxScroll := m.maxArticleScroll()
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.articleScroll > 0 {
			m.articleScroll--
		}
	case key.Matches(msg, m.keys.Down):
		if m.articleScroll < maxScroll {
			m.articleScroll++
		}
	case key.Matches(msg, m.keys.Open):
		return m.openArticleURL()
	case key.Matches(msg, m.keys.Copy):
		return m.copyArticleURL()
	}
	return m, nil
}

func (m Mail) openHTML() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	path := filepath.Join(m.messages[m.messageIndex].Path, "body.html")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			m.status = "No cached HTML body for this message"
			return m, nil
		}
		m.status = fmt.Sprintf("Could not read HTML body: %v", err)
		return m, nil
	}
	m.status = "Opening HTML in browser..."
	cmd := exec.Command("xdg-open", path)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return htmlOpenFinishedMsg{path: path, err: err}
	})
}

func (m Mail) openLink() (Screen, tea.Cmd) {
	if len(m.links) == 0 {
		return m, nil
	}
	url := m.links[m.linkIndex].URL
	m.status = "Opening link in browser..."
	cmd := exec.Command("xdg-open", url)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkOpenFinishedMsg{url: url, err: err}
	})
}

func (m Mail) copyLink() (Screen, tea.Cmd) {
	if len(m.links) == 0 {
		return m, nil
	}
	url := m.links[m.linkIndex].URL
	cmd, err := clipboardCommand(url)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Copying link..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkCopyFinishedMsg{url: url, err: err}
	})
}

func (m Mail) extractLink() (Screen, tea.Cmd) {
	if len(m.links) == 0 {
		return m, nil
	}
	url := m.links[m.linkIndex].URL
	m.status = "Extracting article..."
	return m, func() tea.Msg {
		article, err := extractArticleURL(context.Background(), url)
		return articleExtractedMsg{url: url, article: article, err: err}
	}
}

func (m Mail) openArticleURL() (Screen, tea.Cmd) {
	if m.articleURL == "" {
		return m, nil
	}
	url := m.articleURL
	m.status = "Opening article in browser..."
	cmd := exec.Command("xdg-open", url)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkOpenFinishedMsg{url: url, err: err}
	})
}

func (m Mail) copyArticleURL() (Screen, tea.Cmd) {
	if m.articleURL == "" {
		return m, nil
	}
	cmd, err := clipboardCommand(m.articleURL)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Copying article URL..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return linkCopyFinishedMsg{url: m.articleURL, err: err}
	})
}

func clipboardCommand(value string) (*exec.Cmd, error) {
	for _, candidate := range []struct {
		name string
		args []string
	}{
		{name: "wl-copy"},
		{name: "xclip", args: []string{"-selection", "clipboard"}},
		{name: "xsel", args: []string{"--clipboard", "--input"}},
	} {
		if _, err := exec.LookPath(candidate.name); err == nil {
			cmd := exec.Command(candidate.name, candidate.args...)
			cmd.Stdin = strings.NewReader(value)
			return cmd, nil
		}
	}
	return nil, fmt.Errorf("no clipboard command found: install wl-copy, xclip, or xsel")
}

func (m Mail) loadCmd() tea.Cmd {
	mailboxIndex := m.mailboxIndex
	return func() tea.Msg {
		mailboxes, err := m.store.ListMailboxes()
		if err != nil {
			return mailLoadedMsg{err: err}
		}
		if len(mailboxes) == 0 {
			return mailLoadedMsg{mailboxes: mailboxes}
		}
		if mailboxIndex >= len(mailboxes) {
			mailboxIndex = len(mailboxes) - 1
		}
		mailboxPath, err := m.store.MailboxPath(mailboxes[mailboxIndex].DomainName, mailboxes[mailboxIndex].LocalPart)
		if err != nil {
			return mailLoadedMsg{mailboxes: mailboxes, err: err}
		}
		messages, err := mailstore.ListInbox(mailboxPath)
		return mailLoadedMsg{mailboxes: mailboxes, messages: messages, err: err}
	}
}

func (m *Mail) clampSelection() {
	if m.mailboxIndex >= len(m.mailboxes) {
		m.mailboxIndex = max(0, len(m.mailboxes)-1)
	}
	if m.messageIndex >= len(m.messages) {
		m.messageIndex = max(0, len(m.messages)-1)
	}
	if len(m.messages) == 0 {
		m.mode = mailModeList
		m.detailScroll = 0
	}
}

func (m Mail) listView(width, height int) string {
	var b strings.Builder
	mailbox := m.mailboxes[m.mailboxIndex]
	b.WriteString(fmt.Sprintf("Mailbox %d/%d: %s\n", m.mailboxIndex+1, len(m.mailboxes), mailbox.Address))
	b.WriteString("Use h/l to switch mailboxes, enter to read, r to reload.\n\n")
	if len(m.messages) == 0 {
		b.WriteString("No cached inbox messages for this mailbox. Run `telex sync`.\n")
		return b.String()
	}
	limit := max(1, height-4)
	start := 0
	if m.messageIndex >= limit {
		start = m.messageIndex - limit + 1
	}
	end := min(len(m.messages), start+limit)
	for i := start; i < end; i++ {
		message := m.messages[i]
		cursor := "  "
		if i == m.messageIndex {
			cursor = "> "
		}
		read := " "
		if !message.Meta.Read {
			read = "*"
		}
		line := fmt.Sprintf("%s%s %-16s %-48s %s", cursor, read, truncate(message.Meta.FromAddress, 16), truncate(message.Meta.Subject, 48), message.Meta.ReceivedAt.Format("Jan 02 15:04"))
		b.WriteString(truncate(line, width))
		b.WriteByte('\n')
	}
	return b.String()
}

func (m Mail) detailView(width, height int) string {
	message := m.messages[m.messageIndex]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Subject: %s\n", message.Meta.Subject))
	b.WriteString(fmt.Sprintf("From: %s\n", message.Meta.FromAddress))
	b.WriteString(fmt.Sprintf("To: %s\n", strings.Join(message.Meta.To, ", ")))
	b.WriteString(fmt.Sprintf("Received: %s\n", message.Meta.ReceivedAt.Format("2006-01-02 15:04")))
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	bodyWidth := min(width, mailReadWidth)
	body, err := emailtext.Render(message.BodyText, message.BodyHTML, bodyWidth)
	if err != nil {
		body = fmt.Sprintf("(could not render body: %v)", err)
	}
	lines := strings.Split(body, "\n")
	limit := max(1, height-7)
	maxScroll := max(0, len(lines)-limit)
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
	end := min(len(lines), m.detailScroll+limit)
	for i := m.detailScroll; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	if len(lines) > limit {
		b.WriteString(fmt.Sprintf("\n%d/%d lines", end, len(lines)))
	}
	return b.String()
}

func (m Mail) linksView(width, height int) string {
	message := m.messages[m.messageIndex]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Links: %s\n", message.Meta.Subject))
	b.WriteString("enter opens, e extracts, y copies, esc returns.\n")
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	if len(m.links) == 0 {
		b.WriteString("No links found in this message.\n")
		return b.String()
	}
	limit := max(1, height-5)
	start := 0
	if m.linkIndex >= limit {
		start = m.linkIndex - limit + 1
	}
	end := min(len(m.links), start+limit)
	for i := start; i < end; i++ {
		link := m.links[i]
		cursor := "  "
		if i == m.linkIndex {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s (%s)", cursor, link.Text, link.URL)
		b.WriteString(truncate(line, width))
		b.WriteByte('\n')
	}
	return b.String()
}

func (m Mail) articleView(width, height int) string {
	var b strings.Builder
	b.WriteString("Article reader\n")
	if m.articleURL != "" {
		b.WriteString(fmt.Sprintf("URL: %s\n", m.articleURL))
	}
	b.WriteString("enter opens, y copies, esc returns.\n")
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	bodyWidth := min(width, mailReadWidth)
	article, err := emailtext.RenderMarkdown(m.article, bodyWidth)
	if err != nil {
		article = m.article
	}
	lines := strings.Split(article, "\n")
	limit := max(1, height-6)
	maxScroll := max(0, len(lines)-limit)
	if m.articleScroll > maxScroll {
		m.articleScroll = maxScroll
	}
	end := min(len(lines), m.articleScroll+limit)
	for i := m.articleScroll; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	if len(lines) > limit {
		b.WriteString(fmt.Sprintf("\n%d/%d lines", end, len(lines)))
	}
	return b.String()
}

func (m Mail) maxDetailScroll() int {
	if len(m.messages) == 0 {
		return 0
	}
	body, err := emailtext.Render(m.messages[m.messageIndex].BodyText, m.messages[m.messageIndex].BodyHTML, mailReadWidth)
	if err != nil || strings.TrimSpace(body) == "" {
		return 0
	}
	return max(0, len(strings.Split(body, "\n"))-1)
}

func (m Mail) maxArticleScroll() int {
	if strings.TrimSpace(m.article) == "" {
		return 0
	}
	article, err := emailtext.RenderMarkdown(m.article, mailReadWidth)
	if err != nil {
		article = m.article
	}
	return max(0, len(strings.Split(article, "\n"))-1)
}

func truncate(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "~"
}
