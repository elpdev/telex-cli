package screens

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	mailModeAttachments
	mailReadWidth = 100
)

var mailBoxes = []string{"inbox", "archive", "trash", "sent", "outbox", "drafts"}

var extractArticleURL = articletext.NewExtractor().ExtractURL

type Mail struct {
	store            mailstore.Store
	toggleRead       ToggleReadFunc
	toggleStar       ToggleStarFunc
	archive          MessageActionFunc
	trash            MessageActionFunc
	restore          MessageActionFunc
	sync             SyncFunc
	sendDraft        SendDraftFunc
	updateDraft      UpdateDraftFunc
	deleteDraft      DeleteDraftFunc
	forward          ForwardFunc
	download         DownloadAttachmentFunc
	mailboxes        []mailstore.MailboxMeta
	mailboxIndex     int
	boxIndex         int
	allMessages      []mailstore.CachedMessage
	messages         []mailstore.CachedMessage
	messageIndex     int
	searching        bool
	searchQuery      string
	searchInput      string
	detailScroll     int
	links            []emailtext.Link
	linkIndex        int
	attachmentIndex  int
	savingAttachment bool
	saveDirInput     string
	attachingFile    bool
	attachPathInput  string
	forwarding       bool
	forwardToInput   string
	article          string
	articleURL       string
	articleScroll    int
	mode             mailMode
	loading          bool
	syncing          bool
	confirm          string
	err              error
	status           string
	keys             MailKeyMap
}

type MailKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Previous    key.Binding
	Next        key.Binding
	BoxPrev     key.Binding
	BoxNext     key.Binding
	Open        key.Binding
	OpenHTML    key.Binding
	Links       key.Binding
	Extract     key.Binding
	Compose     key.Binding
	Reply       key.Binding
	Forward     key.Binding
	Send        key.Binding
	Delete      key.Binding
	Attachments key.Binding
	ToggleRead  key.Binding
	ToggleStar  key.Binding
	Archive     key.Binding
	Trash       key.Binding
	Restore     key.Binding
	Copy        key.Binding
	Back        key.Binding
	Refresh     key.Binding
}

type mailLoadedMsg struct {
	mailboxes []mailstore.MailboxMeta
	messages  []mailstore.CachedMessage
	err       error
}

type mailSyncedMsg struct {
	result MailSyncResult
	loaded mailLoadedMsg
	err    error
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

type messageReadToggledMsg struct {
	index int
	path  string
	read  bool
	err   error
}

type messageStarToggledMsg struct {
	index   int
	path    string
	starred bool
	err     error
}

type messageMovedMsg struct {
	index  int
	path   string
	action string
	err    error
}

type draftEditedMsg struct {
	path         string
	existingPath string
	mailbox      mailstore.MailboxMeta
	err          error
}

type forwardDraftCreatedMsg struct {
	remoteID int64
	status   string
	err      error
}

type remoteDraftUpdatedMsg struct {
	remoteID int64
	err      error
}

type draftDeletedMsg struct {
	index int
	path  string
	err   error
}

type draftSentMsg struct {
	index int
	path  string
	err   error
}

type draftAttachmentDetachedMsg struct {
	path string
	err  error
}

type attachmentDownloadedMsg struct {
	path string
	open bool
	err  error
}

type attachmentOpenedMsg struct {
	path string
	err  error
}

type ToggleReadFunc func(context.Context, int64, bool) error
type ToggleStarFunc func(context.Context, int64, bool) error
type MessageActionFunc func(context.Context, int64) error
type SyncFunc func(context.Context) (MailSyncResult, error)
type SendDraftFunc func(context.Context, mailstore.MailboxMeta, mailstore.Draft) error
type UpdateDraftFunc func(context.Context, mailstore.Draft) error
type DeleteDraftFunc func(context.Context, mailstore.Draft) error
type ForwardFunc func(context.Context, int64, mailstore.Draft) (int64, string, error)
type DownloadAttachmentFunc func(context.Context, mailstore.AttachmentMeta) ([]byte, error)

type MailSyncResult struct {
	ActiveMailboxes  int
	SkippedMailboxes int
	OutboxItems      int
	DraftItems       int
	InboxMessages    int
	BodyErrors       int
	InboxErrors      int
}

func NewMail(store mailstore.Store) Mail {
	return NewMailWithActions(store, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func NewMailWithActions(store mailstore.Store, toggleRead ToggleReadFunc, toggleStar ToggleStarFunc, archive MessageActionFunc, trash MessageActionFunc, restore MessageActionFunc, sync SyncFunc, sendDraft SendDraftFunc, updateDraft UpdateDraftFunc, deleteDraft DeleteDraftFunc, forward ForwardFunc, download DownloadAttachmentFunc) Mail {
	return Mail{store: store, toggleRead: toggleRead, toggleStar: toggleStar, archive: archive, trash: trash, restore: restore, sync: sync, sendDraft: sendDraft, updateDraft: updateDraft, deleteDraft: deleteDraft, forward: forward, download: download, keys: DefaultMailKeyMap(), loading: true}
}

func DefaultMailKeyMap() MailKeyMap {
	return MailKeyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "message up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "message down")),
		Previous:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("left/h", "mailbox prev")),
		Next:        key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("right/l", "mailbox next")),
		BoxPrev:     key.NewBinding(key.WithKeys("["), key.WithHelp("[", "box prev")),
		BoxNext:     key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "box next")),
		Open:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		OpenHTML:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open html")),
		Links:       key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "links")),
		Extract:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "extract")),
		Compose:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compose")),
		Reply:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
		Forward:     key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "forward")),
		Send:        key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "send draft")),
		Delete:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete draft")),
		Attachments: key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "attachments")),
		ToggleRead:  key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "read/unread")),
		ToggleStar:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "star/unstar")),
		Archive:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive")),
		Trash:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "trash")),
		Restore:     key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "restore")),
		Copy:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy link")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Refresh:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
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
			m.allMessages = msg.messages
			m.applySearch()
			m.clampSelection()
		}
		return m, nil
	case mailSyncedMsg:
		m.loading = false
		m.syncing = false
		m.err = msg.loaded.err
		if msg.loaded.err == nil {
			m.mailboxes = msg.loaded.mailboxes
			m.allMessages = msg.loaded.messages
			m.applySearch()
			m.clampSelection()
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("Sync failed: %v", msg.err)
			return m, nil
		}
		m.status = syncStatus(msg.result)
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
	case messageReadToggledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not update read state: %v", msg.err)
			return m, nil
		}
		m.updateMessageByPath(msg.path, func(message *mailstore.CachedMessage) { message.Meta.Read = msg.read })
		if msg.read {
			m.status = "Marked read"
		} else {
			m.status = "Marked unread"
		}
		return m, nil
	case messageStarToggledMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not update star state: %v", msg.err)
			return m, nil
		}
		m.updateMessageByPath(msg.path, func(message *mailstore.CachedMessage) { message.Meta.Starred = msg.starred })
		if msg.starred {
			m.status = "Starred"
		} else {
			m.status = "Unstarred"
		}
		return m, nil
	case messageMovedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not %s message: %v", msg.action, msg.err)
			return m, nil
		}
		m.removeMessageByPath(msg.path)
		m.mode = mailModeList
		m.detailScroll = 0
		m.clampSelection()
		switch msg.action {
		case "archive":
			m.status = "Archived"
		case "trash":
			m.status = "Moved to trash"
		case "restore":
			m.status = "Restored"
		}
		return m, nil
	case draftEditedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not save draft: %v", msg.err)
			return m, nil
		}
		draft, err := saveEditedDraft(m.store, msg.mailbox, msg.path, msg.existingPath)
		if err != nil {
			m.status = fmt.Sprintf("Could not save draft: %v", err)
			return m, nil
		}
		m.status = fmt.Sprintf("Draft saved: %s", draft.Meta.ID)
		if msg.existingPath != "" && m.currentBox() == "drafts" {
			loaded := m.load(m.mailboxIndex, m.currentBox())
			m.allMessages = loaded.messages
			m.applySearch()
			m.clampSelection()
		}
		if draft.Meta.RemoteID > 0 && m.updateDraft != nil {
			m.status = fmt.Sprintf("Draft saved locally; syncing remote draft %d...", draft.Meta.RemoteID)
			return m, func() tea.Msg {
				return remoteDraftUpdatedMsg{remoteID: draft.Meta.RemoteID, err: m.updateDraft(context.Background(), *draft)}
			}
		}
		if draft.Meta.DraftKind == "forward" && draft.Meta.SourceMessageID > 0 && m.forward != nil {
			m.status = "Creating reviewed forward draft..."
			return m, func() tea.Msg {
				remoteID, status, err := m.forward(context.Background(), draft.Meta.SourceMessageID, *draft)
				return forwardDraftCreatedMsg{remoteID: remoteID, status: status, err: err}
			}
		}
		return m, nil
	case forwardDraftCreatedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not create forward draft: %v", msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Forward draft created remotely: %d (%s)", msg.remoteID, msg.status)
		return m, nil
	case remoteDraftUpdatedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not sync remote draft %d: %v", msg.remoteID, msg.err)
			return m, nil
		}
		m.status = fmt.Sprintf("Remote draft synced: %d", msg.remoteID)
		return m, nil
	case draftSentMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not send draft: %v", msg.err)
			return m, nil
		}
		m.removeMessageByPath(msg.path)
		m.clampSelection()
		m.status = "Draft sent"
		return m, nil
	case draftDeletedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not delete draft: %v", msg.err)
			return m, nil
		}
		m.removeMessageByPath(msg.path)
		m.clampSelection()
		m.status = "Draft deleted"
		return m, nil
	case draftAttachmentDetachedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not detach attachment: %v", msg.err)
			return m, nil
		}
		loaded := m.load(m.mailboxIndex, m.currentBox())
		m.allMessages = loaded.messages
		m.applySearch()
		m.clampSelection()
		m.mode = mailModeDetail
		m.status = "Attachment detached"
		return m, nil
	case attachmentDownloadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not save attachment: %v", msg.err)
			return m, nil
		}
		if msg.open {
			m.status = "Opening attachment..."
			cmd := exec.Command("xdg-open", msg.path)
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return attachmentOpenedMsg{path: msg.path, err: err} })
		}
		m.status = fmt.Sprintf("Saved attachment: %s", msg.path)
		return m, nil
	case attachmentOpenedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Could not open attachment: %v", msg.err)
		} else {
			m.status = fmt.Sprintf("Opened attachment: %s", msg.path)
		}
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
	if m.mode == mailModeAttachments && len(m.messages) > 0 {
		return style.Render(m.attachmentsView(width, height))
	}
	if m.mode == mailModeDetail && len(m.messages) > 0 {
		return style.Render(m.detailView(width, height))
	}
	return style.Render(m.listView(width, height))
}

func (m Mail) Title() string { return "Mail" }

func (m Mail) KeyBindings() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Previous, m.keys.Next, m.keys.BoxPrev, m.keys.BoxNext, m.keys.Open, m.keys.OpenHTML, m.keys.Links, m.keys.Attachments, m.keys.Extract, m.keys.Compose, m.keys.Reply, m.keys.Forward, m.keys.Send, m.keys.Delete, m.keys.ToggleRead, m.keys.ToggleStar, m.keys.Archive, m.keys.Trash, m.keys.Restore, m.keys.Copy, m.keys.Back, m.keys.Refresh}
}

func (m Mail) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if m.confirm != "" {
		return m.handleConfirmKey(msg)
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.savingAttachment {
		return m.handleAttachmentSaveKey(msg)
	}
	if m.attachingFile {
		return m.handleAttachFileKey(msg)
	}
	if m.forwarding {
		return m.handleForwardKey(msg)
	}
	if m.mode == mailModeArticle {
		return m.handleArticleKey(msg)
	}
	if m.mode == mailModeAttachments {
		return m.handleAttachmentsKey(msg)
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
		if key.Matches(msg, m.keys.Attachments) {
			if len(m.messages[m.messageIndex].Meta.Attachments) == 0 {
				m.status = "No attachments on this message"
				return m, nil
			}
			m.attachmentIndex = 0
			m.mode = mailModeAttachments
			m.status = ""
			return m, nil
		}
		if key.Matches(msg, m.keys.Reply) {
			return m.editReplyDraft()
		}
		if key.Matches(msg, m.keys.Forward) {
			return m.startForward()
		}
		if key.Matches(msg, m.keys.Send) {
			return m.requestConfirm("send-draft", "Send this draft?")
		}
		if key.Matches(msg, m.keys.Extract) {
			return m.editSelectedDraft()
		}
		if key.Matches(msg, m.keys.Delete) {
			return m.requestConfirm("delete-draft", "Delete this draft?")
		}
		if key.Matches(msg, m.keys.ToggleRead) {
			return m.toggleSelectedRead()
		}
		if key.Matches(msg, m.keys.ToggleStar) {
			return m.toggleSelectedStar()
		}
		if key.Matches(msg, m.keys.Archive) {
			return m.moveSelectedMessage("archive")
		}
		if key.Matches(msg, m.keys.Trash) {
			return m.requestConfirm("trash", "Move this message to trash?")
		}
		if key.Matches(msg, m.keys.Restore) {
			return m.moveSelectedMessage("restore")
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
	case key.Matches(msg, m.keys.Refresh):
		if m.sync == nil {
			m.loading = true
			return m, m.loadCmd()
		}
		if m.syncing {
			return m, nil
		}
		m.syncing = true
		m.status = "Syncing mailboxes, outbox, and inbox..."
		return m, m.syncCmd()
	case key.Matches(msg, m.keys.Compose):
		return m.editComposeDraft()
	case key.Matches(msg, m.keys.Send):
		return m.requestConfirm("send-draft", "Send this draft?")
	case key.Matches(msg, m.keys.Extract):
		return m.editSelectedDraft()
	case key.Matches(msg, m.keys.Delete):
		return m.requestConfirm("delete-draft", "Delete this draft?")
	case msg.String() == "/":
		m.searching = true
		m.searchInput = m.searchQuery
		m.status = "Search: " + m.searchInput
		return m, nil
	case key.Matches(msg, m.keys.BoxPrev):
		if m.boxIndex > 0 {
			m.boxIndex--
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
	case key.Matches(msg, m.keys.BoxNext):
		if m.boxIndex < len(mailBoxes)-1 {
			m.boxIndex++
			m.messageIndex = 0
			m.loading = true
			return m, m.loadCmd()
		}
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
	case key.Matches(msg, m.keys.ToggleRead):
		return m.toggleSelectedRead()
	case key.Matches(msg, m.keys.ToggleStar):
		return m.toggleSelectedStar()
	case key.Matches(msg, m.keys.Archive):
		if m.currentBox() == "drafts" {
			return m.startAttachFile()
		}
		return m.moveSelectedMessage("archive")
	case key.Matches(msg, m.keys.Trash):
		return m.requestConfirm("trash", "Move this message to trash?")
	case key.Matches(msg, m.keys.Restore):
		return m.moveSelectedMessage("restore")
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

func (m Mail) handleAttachmentsKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.mode = mailModeDetail
		return m, nil
	}
	attachments := m.messages[m.messageIndex].Meta.Attachments
	if len(attachments) == 0 {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.attachmentIndex > 0 {
			m.attachmentIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.attachmentIndex < len(attachments)-1 {
			m.attachmentIndex++
		}
	case key.Matches(msg, m.keys.Open):
		return m.openAttachment()
	case key.Matches(msg, m.keys.Delete):
		return m.detachSelectedDraftAttachment()
	case key.Matches(msg, m.keys.Copy):
		return m.copyAttachmentURL()
	case key.Matches(msg, m.keys.Send):
		m.savingAttachment = true
		m.saveDirInput = defaultDownloadDir()
		m.status = "Save to: " + m.saveDirInput
	}
	return m, nil
}

func (m Mail) detachSelectedDraftAttachment() (Screen, tea.Cmd) {
	if m.currentBox() != "drafts" {
		m.status = "detach is only available from drafts"
		return m, nil
	}
	message := m.messages[m.messageIndex]
	attachment := message.Meta.Attachments[m.attachmentIndex]
	m.status = "Detaching attachment..."
	return m, func() tea.Msg {
		_, err := mailstore.DetachFileFromDraft(message.Path, attachmentFileLabel(attachment), time.Now())
		return draftAttachmentDetachedMsg{path: message.Path, err: err}
	}
}

func (m Mail) handleAttachmentSaveKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.savingAttachment = false
		m.saveDirInput = ""
		m.status = "Cancelled"
		return m, nil
	case "enter":
		dir := strings.TrimSpace(m.saveDirInput)
		m.savingAttachment = false
		m.saveDirInput = ""
		return m.saveAttachmentTo(dir)
	case "backspace":
		if len(m.saveDirInput) > 0 {
			m.saveDirInput = m.saveDirInput[:len(m.saveDirInput)-1]
		}
		m.status = "Save to: " + m.saveDirInput
		return m, nil
	}
	if msg.Text != "" {
		m.saveDirInput += msg.Text
		m.status = "Save to: " + m.saveDirInput
	}
	return m, nil
}

func (m Mail) handleAttachFileKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.attachingFile = false
		m.attachPathInput = ""
		m.status = "Cancelled"
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.attachPathInput)
		m.attachingFile = false
		m.attachPathInput = ""
		return m.attachFileToSelectedDraft(path)
	case "backspace":
		if len(m.attachPathInput) > 0 {
			m.attachPathInput = m.attachPathInput[:len(m.attachPathInput)-1]
		}
		m.status = "Attach file: " + m.attachPathInput
		return m, nil
	}
	if msg.Text != "" {
		m.attachPathInput += msg.Text
		m.status = "Attach file: " + m.attachPathInput
	}
	return m, nil
}

func (m Mail) handleForwardKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.forwarding = false
		m.forwardToInput = ""
		m.status = "Cancelled"
		return m, nil
	case "enter":
		to := splitDraftAddresses(m.forwardToInput)
		m.forwarding = false
		m.forwardToInput = ""
		return m.createRemoteForwardDraft(to)
	case "backspace":
		if len(m.forwardToInput) > 0 {
			m.forwardToInput = m.forwardToInput[:len(m.forwardToInput)-1]
		}
		m.status = "Forward to: " + m.forwardToInput
		return m, nil
	}
	if msg.Text != "" {
		m.forwardToInput += msg.Text
		m.status = "Forward to: " + m.forwardToInput
	}
	return m, nil
}

func (m Mail) startForward() (Screen, tea.Cmd) {
	if m.forward == nil {
		return m.editForwardDraft(nil)
	}
	m.forwarding = true
	m.forwardToInput = ""
	m.status = "Forward to: "
	return m, nil
}

func (m Mail) createRemoteForwardDraft(to []string) (Screen, tea.Cmd) {
	if len(to) == 0 {
		m.status = "No forward recipients"
		return m, nil
	}
	return m.editForwardDraft(to)
}

func (m Mail) startAttachFile() (Screen, tea.Cmd) {
	if len(m.messages) == 0 || m.currentBox() != "drafts" {
		m.status = "attach is only available from drafts"
		return m, nil
	}
	m.attachingFile = true
	m.attachPathInput = ""
	m.status = "Attach file: "
	return m, nil
}

func (m Mail) attachFileToSelectedDraft(path string) (Screen, tea.Cmd) {
	if path == "" {
		m.status = "No file selected"
		return m, nil
	}
	if len(m.messages) == 0 || m.currentBox() != "drafts" {
		return m, nil
	}
	draftPath := m.messages[m.messageIndex].Path
	m.status = "Attaching file..."
	return m, func() tea.Msg {
		_, err := mailstore.AttachFileToDraft(draftPath, expandHome(path), time.Now())
		return draftEditedMsg{existingPath: draftPath, err: err}
	}
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

func (m Mail) handleSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchInput = ""
		m.status = ""
		return m, nil
	case "enter":
		m.searching = false
		m.searchQuery = strings.TrimSpace(m.searchInput)
		m.messageIndex = 0
		m.applySearch()
		m.clampSelection()
		if m.searchQuery == "" {
			m.status = "Search cleared"
		} else {
			m.status = fmt.Sprintf("Search: %s", m.searchQuery)
		}
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		m.status = "Search: " + m.searchInput
		return m, nil
	}
	if msg.Text != "" {
		m.searchInput += msg.Text
		m.status = "Search: " + m.searchInput
	}
	return m, nil
}

func (m Mail) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		action := m.confirm
		m.confirm = ""
		switch action {
		case "trash":
			return m.moveSelectedMessage("trash")
		case "send-draft":
			return m.sendSelectedDraft()
		case "delete-draft":
			return m.deleteSelectedDraft()
		}
	case "n", "esc":
		m.confirm = ""
		m.status = "Cancelled"
		return m, nil
	}
	return m, nil
}

func (m Mail) requestConfirm(action, prompt string) (Screen, tea.Cmd) {
	m.confirm = action
	m.status = prompt + " Press y to confirm, n to cancel."
	return m, nil
}

func (m Mail) toggleSelectedRead() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if !m.currentBoxSupportsMessageActions() {
		m.status = "Read/unread is only available for message boxes"
		return m, nil
	}
	if m.toggleRead == nil {
		m.status = "Read/unread action is not configured"
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	desiredRead := !message.Meta.Read
	m.status = "Updating read state..."
	return m, func() tea.Msg {
		if err := m.toggleRead(context.Background(), message.Meta.RemoteID, desiredRead); err != nil {
			return messageReadToggledMsg{index: index, path: message.Path, read: desiredRead, err: err}
		}
		if _, err := mailstore.SetCachedMessageRead(message.Path, desiredRead, time.Now()); err != nil {
			return messageReadToggledMsg{index: index, path: message.Path, read: desiredRead, err: err}
		}
		return messageReadToggledMsg{index: index, path: message.Path, read: desiredRead}
	}
}

func (m Mail) toggleSelectedStar() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if !m.currentBoxSupportsMessageActions() {
		m.status = "Star/unstar is only available for message boxes"
		return m, nil
	}
	if m.toggleStar == nil {
		m.status = "Star/unstar action is not configured"
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	desiredStarred := !message.Meta.Starred
	m.status = "Updating star state..."
	return m, func() tea.Msg {
		if err := m.toggleStar(context.Background(), message.Meta.RemoteID, desiredStarred); err != nil {
			return messageStarToggledMsg{index: index, path: message.Path, starred: desiredStarred, err: err}
		}
		if _, err := mailstore.SetCachedMessageStarred(message.Path, desiredStarred, time.Now()); err != nil {
			return messageStarToggledMsg{index: index, path: message.Path, starred: desiredStarred, err: err}
		}
		return messageStarToggledMsg{index: index, path: message.Path, starred: desiredStarred}
	}
}

func (m Mail) moveSelectedMessage(action string) (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	fromBox := m.currentBox()
	moveRemote := m.archive
	toBox := "archive"
	status := "Archiving..."
	switch action {
	case "archive":
		if fromBox != "inbox" {
			m.status = "archive is only available from inbox"
			return m, nil
		}
	case "trash":
		if fromBox != "inbox" {
			m.status = "trash is only available from inbox"
			return m, nil
		}
		moveRemote = m.trash
		toBox = "trash"
		status = "Moving to trash..."
	case "restore":
		if fromBox != "archive" && fromBox != "trash" {
			m.status = "restore is only available from archive or trash"
			return m, nil
		}
		moveRemote = m.restore
		toBox = "inbox"
		status = "Restoring..."
	default:
		m.status = fmt.Sprintf("unknown message action %q", action)
		return m, nil
	}
	if moveRemote == nil {
		m.status = fmt.Sprintf("%s action is not configured", action)
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	mailbox := m.mailboxes[m.mailboxIndex]
	m.status = status
	return m, func() tea.Msg {
		if err := moveRemote(context.Background(), message.Meta.RemoteID); err != nil {
			return messageMovedMsg{index: index, path: message.Path, action: action, err: err}
		}
		mailboxPath, err := m.store.MailboxPath(mailbox.DomainName, mailbox.LocalPart)
		if err != nil {
			return messageMovedMsg{index: index, path: message.Path, action: action, err: err}
		}
		if _, err := mailstore.MoveCachedMessage(mailboxPath, fromBox, toBox, message.Path, time.Now()); err != nil {
			return messageMovedMsg{index: index, path: message.Path, action: action, err: err}
		}
		return messageMovedMsg{index: index, path: message.Path, action: action}
	}
}

func (m Mail) editComposeDraft() (Screen, tea.Cmd) {
	if len(m.mailboxes) == 0 {
		return m, nil
	}
	return m.editDraft(draftTemplate(draftFields{From: m.mailboxes[m.mailboxIndex].Address}), "")
}

func (m Mail) sendSelectedDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.currentBox() != "drafts" {
		m.status = "send is only available from drafts"
		return m, nil
	}
	if m.sendDraft == nil {
		m.status = "send draft action is not configured"
		return m, nil
	}
	index := m.messageIndex
	message := m.messages[index]
	mailbox := m.mailboxes[m.mailboxIndex]
	m.status = "Sending draft..."
	return m, func() tea.Msg {
		draft, err := mailstore.ReadDraft(message.Path)
		if err != nil {
			return draftSentMsg{index: index, path: message.Path, err: err}
		}
		if err := m.sendDraft(context.Background(), mailbox, *draft); err != nil {
			return draftSentMsg{index: index, path: message.Path, err: err}
		}
		return draftSentMsg{index: index, path: message.Path}
	}
}

func (m Mail) editReplyDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 || len(m.mailboxes) == 0 {
		return m, nil
	}
	message := m.messages[m.messageIndex]
	subject := message.Meta.Subject
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "re:") {
		subject = "Re: " + subject
	}
	body := quotedReplyBody(message)
	return m.editDraft(draftTemplate(draftFields{From: m.mailboxes[m.mailboxIndex].Address, To: []string{message.Meta.FromAddress}, Subject: subject, Body: body, SourceMessageID: message.Meta.RemoteID, ConversationID: message.Meta.ConversationID}), "")
}

func (m Mail) editForwardDraft(to []string) (Screen, tea.Cmd) {
	if len(m.messages) == 0 || len(m.mailboxes) == 0 {
		return m, nil
	}
	message := m.messages[m.messageIndex]
	subject := message.Meta.Subject
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "fwd:") {
		subject = "Fwd: " + subject
	}
	return m.editDraft(draftTemplate(draftFields{From: m.mailboxes[m.mailboxIndex].Address, To: to, Subject: subject, Body: quotedForwardBody(message), SourceMessageID: message.Meta.RemoteID, ConversationID: message.Meta.ConversationID, DraftKind: "forward"}), "")
}

func (m Mail) editSelectedDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.currentBox() != "drafts" {
		m.status = "edit is only available from drafts"
		return m, nil
	}
	draft, err := mailstore.ReadDraft(m.messages[m.messageIndex].Path)
	if err != nil {
		m.status = fmt.Sprintf("Could not read draft: %v", err)
		return m, nil
	}
	return m.editDraft(draftTemplate(draftFields{From: draft.Meta.FromAddress, To: draft.Meta.To, CC: draft.Meta.CC, BCC: draft.Meta.BCC, Subject: draft.Meta.Subject, Body: draft.Body, SourceMessageID: draft.Meta.SourceMessageID, ConversationID: draft.Meta.ConversationID}), draft.Path)
}

func (m Mail) editDraft(content, existingPath string) (Screen, tea.Cmd) {
	file, err := os.CreateTemp("", "telex-draft-*.md")
	if err != nil {
		m.status = fmt.Sprintf("Could not create draft file: %v", err)
		return m, nil
	}
	path := file.Name()
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		m.status = fmt.Sprintf("Could not write draft file: %v", err)
		return m, nil
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		m.status = fmt.Sprintf("Could not close draft file: %v", err)
		return m, nil
	}
	cmd, err := editorCommand(path)
	if err != nil {
		_ = os.Remove(path)
		m.status = err.Error()
		return m, nil
	}
	mailbox := m.mailboxes[m.mailboxIndex]
	m.loading = true
	m.status = "Editing draft..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return draftEditedMsg{path: path, existingPath: existingPath, mailbox: mailbox, err: err}
	})
}

func (m Mail) deleteSelectedDraft() (Screen, tea.Cmd) {
	if len(m.messages) == 0 {
		return m, nil
	}
	if m.currentBox() != "drafts" {
		m.status = "delete is only available from drafts"
		return m, nil
	}
	index := m.messageIndex
	path := m.messages[index].Path
	m.status = "Deleting draft..."
	return m, func() tea.Msg {
		draft, err := mailstore.ReadDraft(path)
		if err != nil {
			return draftDeletedMsg{index: index, path: path, err: err}
		}
		if draft.Meta.RemoteID > 0 && m.deleteDraft != nil {
			if err := m.deleteDraft(context.Background(), *draft); err != nil {
				return draftDeletedMsg{index: index, path: path, err: err}
			}
		}
		return draftDeletedMsg{index: index, path: path, err: mailstore.DeleteDraft(path)}
	}
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

func (m Mail) openAttachment() (Screen, tea.Cmd) {
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	path := mailstore.AttachmentCachePath(m.messages[m.messageIndex].Path, attachment)
	if _, err := os.Stat(path); err == nil {
		m.status = "Opening attachment..."
		cmd := exec.Command("xdg-open", path)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return attachmentOpenedMsg{path: path, err: err} })
	}
	return m.downloadAttachment(path, true)
}

func (m Mail) saveAttachmentTo(dir string) (Screen, tea.Cmd) {
	if dir == "" {
		dir = defaultDownloadDir()
	}
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	dest := uniquePath(filepath.Join(expandHome(dir), attachmentSaveName(attachment)))
	cachePath := mailstore.AttachmentCachePath(m.messages[m.messageIndex].Path, attachment)
	if data, err := os.ReadFile(cachePath); err == nil {
		m.status = "Saving attachment..."
		return m, func() tea.Msg { return attachmentDownloadedMsg{path: dest, err: writeAttachmentFile(dest, data)} }
	}
	return m.downloadAttachment(dest, false)
}

func (m Mail) downloadAttachment(path string, open bool) (Screen, tea.Cmd) {
	if m.download == nil {
		m.status = "Attachment download is not configured"
		return m, nil
	}
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	m.status = "Downloading attachment..."
	return m, func() tea.Msg {
		data, err := m.download(context.Background(), attachment)
		if err != nil {
			return attachmentDownloadedMsg{path: path, open: open, err: err}
		}
		return attachmentDownloadedMsg{path: path, open: open, err: writeAttachmentFile(path, data)}
	}
}

func (m Mail) copyAttachmentURL() (Screen, tea.Cmd) {
	attachment := m.messages[m.messageIndex].Meta.Attachments[m.attachmentIndex]
	if attachment.DownloadURL == "" {
		m.status = "No download URL for this attachment"
		return m, nil
	}
	cmd, err := clipboardCommand(attachment.DownloadURL)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "Copying attachment URL..."
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return linkCopyFinishedMsg{url: attachment.DownloadURL, err: err} })
}

func writeAttachmentFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func attachmentSaveName(attachment mailstore.AttachmentMeta) string {
	path := mailstore.AttachmentCachePath("", attachment)
	return filepath.Base(path)
}

func attachmentFileLabel(attachment mailstore.AttachmentMeta) string {
	if attachment.CacheName != "" {
		return attachment.CacheName
	}
	return attachment.Filename
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func defaultDownloadDir() string {
	if xdg := strings.TrimSpace(os.Getenv("XDG_DOWNLOAD_DIR")); xdg != "" {
		return expandHome(xdg)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Downloads")
	}
	return "."
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
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

func editorCommand(path string) (*exec.Cmd, error) {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, fmt.Errorf("editor is not configured")
	}
	args := append(parts[1:], path)
	return exec.Command(parts[0], args...), nil
}

type draftFields struct {
	From            string
	To              []string
	CC              []string
	BCC             []string
	Subject         string
	Body            string
	SourceMessageID int64
	ConversationID  int64
	DraftKind       string
}

func draftTemplate(fields draftFields) string {
	extra := ""
	if fields.SourceMessageID > 0 {
		extra += fmt.Sprintf("X-Telex-Source-Message-ID: %d\n", fields.SourceMessageID)
	}
	if fields.ConversationID > 0 {
		extra += fmt.Sprintf("X-Telex-Conversation-ID: %d\n", fields.ConversationID)
	}
	if fields.DraftKind != "" {
		extra += fmt.Sprintf("X-Telex-Draft-Kind: %s\n", fields.DraftKind)
	}
	return fmt.Sprintf("From: %s\nTo: %s\nCc: %s\nBcc: %s\nSubject: %s\n%s\n%s", fields.From, strings.Join(fields.To, ", "), strings.Join(fields.CC, ", "), strings.Join(fields.BCC, ", "), fields.Subject, extra, fields.Body)
}

func saveEditedDraft(store mailstore.Store, mailbox mailstore.MailboxMeta, path, existingPath string) (*mailstore.Draft, error) {
	defer os.Remove(path)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fields, err := parseDraftFile(string(content))
	if err != nil {
		return nil, err
	}
	input := mailstore.DraftInput{Mailbox: mailbox, Subject: fields.Subject, To: fields.To, CC: fields.CC, BCC: fields.BCC, Body: fields.Body, SourceMessageID: fields.SourceMessageID, ConversationID: fields.ConversationID, DraftKind: fields.DraftKind, Now: time.Now()}
	if existingPath != "" {
		return store.UpdateDraft(existingPath, input)
	}
	return store.CreateDraft(input)
}

func parseDraftFile(content string) (draftFields, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	parts := strings.SplitN(content, "\n\n", 2)
	if len(parts) != 2 {
		return draftFields{}, fmt.Errorf("draft must contain headers, a blank line, then body")
	}
	fields := draftFields{Body: parts[1]}
	for _, line := range strings.Split(parts[0], "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "from":
			fields.From = strings.TrimSpace(value)
		case "to":
			fields.To = splitDraftAddresses(value)
		case "cc":
			fields.CC = splitDraftAddresses(value)
		case "bcc":
			fields.BCC = splitDraftAddresses(value)
		case "subject":
			fields.Subject = strings.TrimSpace(value)
		case "x-telex-source-message-id":
			fields.SourceMessageID = parseDraftInt(value)
		case "x-telex-conversation-id":
			fields.ConversationID = parseDraftInt(value)
		case "x-telex-draft-kind":
			fields.DraftKind = strings.TrimSpace(value)
		}
	}
	if strings.TrimSpace(fields.Subject) == "" {
		return draftFields{}, fmt.Errorf("subject is required")
	}
	return fields, nil
}

func parseDraftInt(value string) int64 {
	var parsed int64
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed)
	return parsed
}

func splitDraftAddresses(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' })
	addresses := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			addresses = append(addresses, part)
		}
	}
	return addresses
}

func quotedReplyBody(message mailstore.CachedMessage) string {
	body := strings.TrimSpace(emailtext.DecodeQuotedPrintable(message.BodyText))
	if body == "" {
		body = strings.TrimSpace(emailtext.DecodeQuotedPrintable(message.BodyHTML))
	}
	if body == "" {
		return "\n\n"
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return "\n\n" + strings.Join(lines, "\n") + "\n"
}

func quotedForwardBody(message mailstore.CachedMessage) string {
	body := strings.TrimSpace(emailtext.DecodeQuotedPrintable(message.BodyText))
	if body == "" {
		body = strings.TrimSpace(emailtext.DecodeQuotedPrintable(message.BodyHTML))
	}
	var b strings.Builder
	b.WriteString("\n\n---------- Forwarded message ---------\n")
	b.WriteString(fmt.Sprintf("From: %s\n", senderLine(message)))
	if len(message.Meta.To) > 0 {
		b.WriteString(fmt.Sprintf("To: %s\n", strings.Join(message.Meta.To, ", ")))
	}
	if len(message.Meta.CC) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(message.Meta.CC, ", ")))
	}
	if !message.Meta.ReceivedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Date: %s\n", message.Meta.ReceivedAt.Format(time.RFC1123)))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\n\n", message.Meta.Subject))
	if body != "" {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func senderLine(message mailstore.CachedMessage) string {
	if strings.TrimSpace(message.Meta.FromName) == "" {
		return message.Meta.FromAddress
	}
	return fmt.Sprintf("%s <%s>", message.Meta.FromName, message.Meta.FromAddress)
}

func (m Mail) loadCmd() tea.Cmd {
	mailboxIndex := m.mailboxIndex
	box := m.currentBox()
	return func() tea.Msg {
		return m.load(mailboxIndex, box)
	}
}

func (m Mail) syncCmd() tea.Cmd {
	mailboxIndex := m.mailboxIndex
	box := m.currentBox()
	return func() tea.Msg {
		result, err := m.sync(context.Background())
		loaded := m.load(mailboxIndex, box)
		return mailSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (m Mail) load(mailboxIndex int, box string) mailLoadedMsg {
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
	messages, err := listCachedBox(mailboxPath, box)
	return mailLoadedMsg{mailboxes: mailboxes, messages: messages, err: err}
}

func syncStatus(result MailSyncResult) string {
	status := fmt.Sprintf("Synced %d mailbox(es), %d inbox message(s)", result.ActiveMailboxes, result.InboxMessages)
	if result.OutboxItems > 0 {
		status = fmt.Sprintf("%s, %d outbox item(s)", status, result.OutboxItems)
	}
	if result.DraftItems > 0 {
		status = fmt.Sprintf("%s, %d remote draft(s)", status, result.DraftItems)
	}
	if result.BodyErrors > 0 || result.InboxErrors > 0 {
		status = fmt.Sprintf("%s with warnings", status)
	}
	return status
}

func listCachedBox(mailboxPath, box string) ([]mailstore.CachedMessage, error) {
	switch box {
	case "inbox", "archive", "trash":
		return mailstore.ListMessages(mailboxPath, box)
	case "sent":
		drafts, err := mailstore.ListSent(mailboxPath)
		if err != nil {
			return nil, err
		}
		return draftsToCachedMessages(drafts), nil
	case "outbox":
		drafts, err := mailstore.ListOutbox(mailboxPath)
		if err != nil {
			return nil, err
		}
		return draftsToCachedMessages(drafts), nil
	case "drafts":
		drafts, err := mailstore.ListDrafts(mailboxPath)
		if err != nil {
			return nil, err
		}
		return draftsToCachedMessages(drafts), nil
	default:
		return nil, fmt.Errorf("unknown mail box %q", box)
	}
}

func draftsToCachedMessages(drafts []mailstore.Draft) []mailstore.CachedMessage {
	messages := make([]mailstore.CachedMessage, 0, len(drafts))
	for _, draft := range drafts {
		messages = append(messages, mailstore.CachedMessage{
			Meta: mailstore.MessageMeta{
				SchemaVersion: draft.Meta.SchemaVersion,
				Kind:          draft.Meta.Kind,
				RemoteID:      draft.Meta.RemoteID,
				DomainID:      draft.Meta.DomainID,
				DomainName:    draft.Meta.DomainName,
				InboxID:       draft.Meta.InboxID,
				Mailbox:       draft.Meta.Kind,
				Status:        draft.Meta.RemoteStatus,
				RemoteError:   draft.Meta.RemoteError,
				Attachments:   draft.Meta.Attachments,
				Subject:       draft.Meta.Subject,
				FromAddress:   draft.Meta.FromAddress,
				To:            draft.Meta.To,
				CC:            draft.Meta.CC,
				Read:          true,
				ReceivedAt:    draft.Meta.UpdatedAt,
				SyncedAt:      draft.Meta.UpdatedAt,
			},
			Path:     draft.Path,
			BodyText: draft.Body,
		})
	}
	return messages
}

func (m *Mail) applySearch() {
	query := strings.ToLower(strings.TrimSpace(m.searchQuery))
	if query == "" {
		m.messages = append([]mailstore.CachedMessage(nil), m.allMessages...)
		return
	}
	m.messages = m.messages[:0]
	for _, message := range m.allMessages {
		if cachedMessageMatches(message, query) {
			m.messages = append(m.messages, message)
		}
	}
}

func cachedMessageMatches(message mailstore.CachedMessage, query string) bool {
	values := []string{
		message.Meta.Subject,
		message.Meta.FromAddress,
		message.Meta.FromName,
		strings.Join(message.Meta.To, " "),
		strings.Join(message.Meta.CC, " "),
		message.Meta.Status,
		message.Meta.RemoteError,
		message.BodyText,
		message.BodyHTML,
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func (m *Mail) updateMessageByPath(path string, update func(*mailstore.CachedMessage)) {
	for i := range m.allMessages {
		if m.allMessages[i].Path == path {
			update(&m.allMessages[i])
		}
	}
	m.applySearch()
}

func (m *Mail) removeMessageByPath(path string) {
	m.allMessages = removeCachedMessageByPath(m.allMessages, path)
	m.applySearch()
}

func removeCachedMessageByPath(messages []mailstore.CachedMessage, path string) []mailstore.CachedMessage {
	for i := range messages {
		if messages[i].Path == path {
			return append(messages[:i], messages[i+1:]...)
		}
	}
	return messages
}

func (m Mail) currentBox() string {
	if m.boxIndex < 0 || m.boxIndex >= len(mailBoxes) {
		return "inbox"
	}
	return mailBoxes[m.boxIndex]
}

func (m Mail) currentBoxSupportsMessageActions() bool {
	switch m.currentBox() {
	case "inbox", "archive", "trash":
		return true
	default:
		return false
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
	box := m.currentBox()
	b.WriteString(fmt.Sprintf("Mailbox %d/%d: %s | Box %d/%d: %s\n", m.mailboxIndex+1, len(m.mailboxes), mailbox.Address, m.boxIndex+1, len(mailBoxes), box))
	b.WriteString("Use h/l to switch mailboxes, [/] to switch boxes, / search, c compose, enter to read, r reload.")
	if box == "inbox" {
		b.WriteString(" a archive, d trash.")
	} else if box == "archive" || box == "trash" {
		b.WriteString(" R restore.")
	} else if box == "drafts" {
		b.WriteString(" a attach, e edit, S send, x delete.")
	}
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	if m.searchQuery != "" {
		b.WriteString(fmt.Sprintf("Filter: %s (%d/%d)\n", m.searchQuery, len(m.messages), len(m.allMessages)))
	}
	b.WriteString("\n")
	if len(m.messages) == 0 {
		b.WriteString(fmt.Sprintf("No cached %s messages for this mailbox. Run `telex sync`.\n", box))
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
		star := " "
		if message.Meta.Starred {
			star = "!"
		}
		line := fmt.Sprintf("%s%s%s %-16s %-48s %s", cursor, read, star, truncate(message.Meta.FromAddress, 16), truncate(message.Meta.Subject, 48), message.Meta.ReceivedAt.Format("Jan 02 15:04"))
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
	if len(message.Meta.CC) > 0 {
		b.WriteString(fmt.Sprintf("CC: %s\n", strings.Join(message.Meta.CC, ", ")))
	}
	b.WriteString(fmt.Sprintf("Box: %s\n", message.Meta.Mailbox))
	if message.Meta.RemoteID > 0 {
		b.WriteString(fmt.Sprintf("Remote ID: %d\n", message.Meta.RemoteID))
	}
	if message.Meta.Status != "" {
		b.WriteString(fmt.Sprintf("Delivery status: %s\n", message.Meta.Status))
	}
	if message.Meta.RemoteError != "" {
		b.WriteString(fmt.Sprintf("Delivery error: %s\n", message.Meta.RemoteError))
	}
	if len(message.Meta.Attachments) > 0 {
		b.WriteString(fmt.Sprintf("Attachments: %d (A to view)\n", len(message.Meta.Attachments)))
	}
	if m.currentBoxSupportsMessageActions() {
		b.WriteString(fmt.Sprintf("Read: %t\n", message.Meta.Read))
		b.WriteString(fmt.Sprintf("Starred: %t\n", message.Meta.Starred))
	}
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

func (m Mail) attachmentsView(width, height int) string {
	message := m.messages[m.messageIndex]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Attachments: %s\n", message.Meta.Subject))
	b.WriteString("enter opens/downloads to cache, S saves to directory, y copies URL, esc returns.\n")
	if m.status != "" {
		b.WriteString(fmt.Sprintf("Status: %s\n", m.status))
	}
	b.WriteString("\n")
	attachments := message.Meta.Attachments
	if len(attachments) == 0 {
		b.WriteString("No attachments on this message.\n")
		return b.String()
	}
	limit := max(1, height-5)
	start := 0
	if m.attachmentIndex >= limit {
		start = m.attachmentIndex - limit + 1
	}
	end := min(len(attachments), start+limit)
	for i := start; i < end; i++ {
		attachment := attachments[i]
		cursor := "  "
		if i == m.attachmentIndex {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%d. %s %s %s", cursor, i+1, attachment.Filename, attachment.ContentType, formatBytes(attachment.ByteSize))
		b.WriteString(truncate(line, width))
		b.WriteByte('\n')
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

func formatBytes(size int64) string {
	switch {
	case size >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	case size > 0:
		return fmt.Sprintf("%d B", size)
	default:
		return ""
	}
}
