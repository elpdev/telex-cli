package app

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	helpbubble "charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/debug"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/drivesync"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailsend"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/mailsync"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/telex-cli/internal/theme"
)

const defaultScreen = "home"

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type Model struct {
	width  int
	height int

	activeScreen string
	screens      map[string]screens.Screen
	screenOrder  []string
	initialized  map[string]bool

	showSidebar        bool
	showHelp           bool
	showCommandPalette bool

	focus         FocusArea
	sidebarCursor string
	keys          KeyMap
	help          helpbubble.Model

	commands       *commands.Registry
	commandPalette commands.PaletteModel

	theme theme.Theme
	logs  *debug.Log
	meta  BuildInfo

	configPath string
	dataPath   string
	prefsPath  string
	instance   string
	client     *api.Client
}

func New(meta BuildInfo) Model {
	return assembleModel(meta, "", "", "", &config.UIPrefs{})
}

func NewWithDataPath(meta BuildInfo, dataPath string) Model {
	return NewWithPaths(meta, "", dataPath)
}

func NewWithPaths(meta BuildInfo, configPath, dataPath string) Model {
	prefsPath := config.PrefsPathFor(configPath)
	prefs, err := config.LoadPrefs(prefsPath)
	if err != nil {
		prefs = &config.UIPrefs{}
	}
	return assembleModel(meta, configPath, dataPath, prefsPath, prefs)
}

func assembleModel(meta BuildInfo, configPath, dataPath, prefsPath string, prefs *config.UIPrefs) Model {
	log := debug.NewLog()
	log.Info("App started")
	chosen := theme.Phosphor()
	if prefs.Theme != "" {
		if t, ok := themeByName(prefs.Theme); ok {
			chosen = t
		}
	}
	sidebar := true
	if prefs.SidebarVisible != nil {
		sidebar = *prefs.SidebarVisible
	}

	m := Model{
		activeScreen: defaultScreen,
		screens:      make(map[string]screens.Screen),
		initialized:  make(map[string]bool),
		showSidebar:  sidebar,
		focus:        FocusMain,
		keys:         DefaultKeyMap(),
		help:         helpbubble.New(),
		commands:     commands.NewRegistry(),
		theme:        chosen,
		logs:         log,
		meta:         meta,
		configPath:   configPath,
		dataPath:     dataPath,
		prefsPath:    prefsPath,
		instance:     loadInstance(configPath),
	}

	m.registerScreens()
	m.registerCommands()
	m.commandPalette = commands.NewPaletteModel(m.commands, theme.BuiltIns())
	return m
}

func themeByName(name string) (theme.Theme, bool) {
	for _, t := range theme.BuiltIns() {
		if t.Name == name {
			return t, true
		}
	}
	return theme.Theme{}, false
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{func() tea.Msg { return tea.RequestWindowSize() }}
	for id, screen := range m.screens {
		if id == "news" || strings.HasPrefix(id, hackerNewsPrefix) {
			continue
		}
		cmds = append(cmds, screen.Init())
	}
	return tea.Batch(cmds...)
}

func (m Model) devBuild() bool { return m.meta.Version == "dev" }

func (m *Model) registerScreens() {
	m.screens["home"] = m.buildHome()
	m.screens["mail"] = m.buildMailScreen()
	for _, scope := range aggregateMailScreens() {
		m.screens[scope.id] = m.buildAggregateMailScreen(scope)
	}
	m.screens["mail-admin"] = screens.NewMailAdmin(m.loadMailAdmin).WithActions(m.saveDomain, m.deleteDomain, m.validateDomainOutbound, m.saveInbox, m.deleteInbox, m.inboxPipeline)
	m.screens["calendar"] = screens.NewCalendar(calendarstore.New(m.dataPath), m.syncCalendar).WithActions(m.createCalendarEvent, m.updateCalendarEvent, m.deleteCalendarEvent).WithCalendarActions(m.createCalendar, m.updateCalendar, m.deleteCalendar).WithImportICS(m.importCalendarICS).WithInvitationActions(m.showCalendarInvitation, m.syncCalendarInvitation, m.respondCalendarInvitation)
	m.screens["contacts"] = screens.NewContacts(contactstore.New(m.dataPath), m.syncContacts).WithActions(m.deleteContact, m.loadContactNote, m.updateContactNote, m.loadContactCommunications)
	m.screens["drive"] = screens.NewDrive(drivestore.New(m.dataPath), m.syncDrive).WithActions(m.downloadDriveFile, m.openDriveFile, m.uploadDriveFile, m.createDriveFolder, m.renameDriveFile, m.renameDriveFolder, m.deleteDriveFile, m.deleteDriveFolder)
	m.screens["notes"] = screens.NewNotes(notestore.New(m.dataPath), m.syncNotes).WithActions(m.createNote, m.updateNote, m.deleteNote)
	m.screens["settings"] = m.buildSettings()
	if m.devBuild() {
		m.screens["logs"] = screens.NewLogs(m.logs)
	}
	m.registerHackerNewsModule()
	m.screens["news"] = m.buildNews()
	m.refreshScreenOrder()
}

type aggregateMailScreen struct {
	id         string
	title      string
	box        string
	unreadOnly bool
}

func aggregateMailScreens() []aggregateMailScreen {
	return []aggregateMailScreen{
		{id: "mail-unread", title: "Unread", box: "inbox", unreadOnly: true},
		{id: "mail-inbox", title: "Inbox", box: "inbox"},
		{id: "mail-sent", title: "Sent", box: "sent"},
		{id: "mail-drafts", title: "Drafts", box: "drafts"},
		{id: "mail-outbox", title: "Outbox", box: "outbox"},
		{id: "mail-junk", title: "Junk", box: "junk"},
		{id: "mail-archive", title: "Archive", box: "archive"},
		{id: "mail-trash", title: "Trash", box: "trash"},
	}
}

func (m *Model) buildMailScreen() screens.Mail {
	return screens.NewMailWithActions(mailstore.New(m.dataPath), m.toggleMessageRead, m.toggleMessageStar, m.archiveMessage, m.trashMessage, m.restoreMessage, m.syncMail, m.sendDraft, m.updateDraft, m.deleteDraft, m.forwardMessage, m.downloadAttachment, m.searchMail).WithConversationActions(m.conversationTimeline, m.conversationBody).WithJunkActions(m.junkMessage, m.notJunkMessage).WithSenderPolicyActions(m.blockSender, m.unblockSender, m.blockDomain, m.unblockDomain, m.trustSender, m.untrustSender)
}

func (m *Model) buildAggregateMailScreen(scope aggregateMailScreen) screens.Mail {
	return screens.NewAggregateMailWithActions(mailstore.New(m.dataPath), scope.title, scope.box, scope.unreadOnly, m.toggleMessageRead, m.toggleMessageStar, m.archiveMessage, m.trashMessage, m.restoreMessage, m.syncMail, m.sendDraft, m.updateDraft, m.deleteDraft, m.forwardMessage, m.downloadAttachment, m.searchMail).WithConversationActions(m.conversationTimeline, m.conversationBody).WithJunkActions(m.junkMessage, m.notJunkMessage).WithSenderPolicyActions(m.blockSender, m.unblockSender, m.blockDomain, m.unblockDomain, m.trustSender, m.untrustSender)
}

func (m *Model) buildHome() screens.Home {
	if existing, ok := m.screens["home"].(screens.Home); ok {
		return existing.Reconfigure(m.theme)
	}
	navigate := func(id string) tea.Cmd {
		return func() tea.Msg { return routeMsg{ScreenID: id} }
	}
	return screens.NewHome(
		mailstore.New(m.dataPath),
		calendarstore.New(m.dataPath),
		notestore.New(m.dataPath),
		drivestore.New(m.dataPath),
		m.theme,
		navigate,
	)
}

func (m *Model) buildNews() screens.News {
	tabs := []screens.NewsTab{
		{ID: "hn-top", Label: "Top"},
		{ID: "hn-new", Label: "New"},
		{ID: "hn-best", Label: "Best"},
		{ID: "hn-ask", Label: "Ask"},
		{ID: "hn-show", Label: "Show"},
		{ID: "hn-jobs", Label: "Jobs"},
		{ID: "hn-saved", Label: "Saved"},
	}
	initFn := func(id string) tea.Cmd { return m.initScreen(id) }
	news := screens.NewNews(tabs, m.theme, m.screens, initFn)
	if existing, ok := m.screens["news"].(screens.News); ok {
		news = news.SetActiveIndex(existing.ActiveIndex())
	}
	return news
}

func (m *Model) buildSettings() screens.Settings {
	state := m.computeSettingsState()
	actions := m.settingsActions()
	if existing, ok := m.screens["settings"].(screens.Settings); ok {
		return existing.Reconfigure(state, m.theme, theme.BuiltIns(), actions)
	}
	return screens.NewSettings(state, m.theme, theme.BuiltIns(), actions)
}

func (m *Model) computeSettingsState() screens.SettingsState {
	configFile, tokenFile := config.Paths(m.configPath)

	driveSync := config.DriveSyncFull
	if cfg, err := config.LoadFrom(configFile); err == nil && cfg != nil {
		driveSync = cfg.DriveSyncMode()
	}

	authStatus := "Not signed in"
	signedIn := false
	if tc, err := config.LoadTokenFrom(tokenFile); err == nil && tc != nil && tc.Token != "" {
		if tc.Valid() {
			signedIn = true
			authStatus = "Signed in · expires in " + humanDuration(time.Until(tc.ExpiresAt))
		} else {
			authStatus = "Token expired"
		}
	}

	return screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Instance:       m.instance,
		AuthStatus:     authStatus,
		SignedIn:       signedIn,
		DataDir:        m.dataPath,
		ConfigDir:      m.configDirPath(),
		CacheSize:      computeCacheSize(m.dataPath),
		DriveSyncMode:  driveSync,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	}
}

func (m *Model) settingsActions() screens.SettingsActions {
	return screens.SettingsActions{
		OpenPath: func(path string) tea.Cmd {
			return func() tea.Msg {
				if path == "" {
					return nil
				}
				_ = startDetached(exec.Command("xdg-open", path))
				return nil
			}
		},
		OpenURL: func(target string) tea.Cmd {
			return func() tea.Msg {
				if target == "" {
					return nil
				}
				_ = startDetached(exec.Command("xdg-open", target))
				return nil
			}
		},
		OpenMailAdmin: func() tea.Cmd {
			return func() tea.Msg { return routeMsg{"mail-admin"} }
		},
		SignOut: func() tea.Cmd {
			return func() tea.Msg { return settingsSignOutMsg{} }
		},
	}
}

func (m *Model) configDirPath() string {
	if m.configPath != "" {
		return filepath.Dir(filepath.Clean(m.configPath))
	}
	return config.Dir()
}

func computeCacheSize(dataPath string) int64 {
	if dataPath == "" {
		return 0
	}
	var size int64
	_ = filepath.WalkDir(dataPath, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func humanDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func (m *Model) toggleMessageStar(ctx context.Context, id int64, starred bool) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if starred {
		_, err = service.StarMessage(ctx, id)
	} else {
		_, err = service.UnstarMessage(ctx, id)
	}
	return err
}

func (m *Model) loadMailAdmin(ctx context.Context) ([]mail.Domain, []mail.Inbox, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, nil, err
	}
	domains, _, err := service.ListDomains(ctx, mail.DomainListParams{ListParams: mail.ListParams{Page: 1, PerPage: 100}, Sort: "name"})
	if err != nil {
		return nil, nil, err
	}
	inboxes, _, err := service.ListInboxes(ctx, mail.InboxListParams{ListParams: mail.ListParams{Page: 1, PerPage: 250}, Count: "all", Sort: "address"})
	if err != nil {
		return nil, nil, err
	}
	return domains, inboxes, nil
}

func (m *Model) saveDomain(ctx context.Context, id *int64, input mail.DomainInput) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if id == nil {
		_, err = service.CreateDomain(ctx, input)
		return err
	}
	_, err = service.UpdateDomain(ctx, *id, input)
	return err
}

func (m *Model) deleteDomain(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	return service.DeleteDomain(ctx, id)
}

func (m *Model) validateDomainOutbound(ctx context.Context, id int64) (*mail.DomainOutboundValidation, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	return service.ValidateDomainOutbound(ctx, id, nil)
}

func (m *Model) saveInbox(ctx context.Context, id *int64, input mail.InboxInput) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if id == nil {
		_, err = service.CreateInbox(ctx, input)
		return err
	}
	_, err = service.UpdateInbox(ctx, *id, input)
	return err
}

func (m *Model) deleteInbox(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	return service.DeleteInbox(ctx, id)
}

func (m *Model) inboxPipeline(ctx context.Context, id int64) (*mail.InboxPipeline, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	return service.InboxPipeline(ctx, id)
}

func (m *Model) toggleMessageRead(ctx context.Context, id int64, read bool) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	if read {
		_, err = service.MarkMessageRead(ctx, id)
	} else {
		_, err = service.MarkMessageUnread(ctx, id)
	}
	return err
}

func (m *Model) archiveMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.ArchiveMessage(ctx, id)
	return err
}

func (m *Model) trashMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.TrashMessage(ctx, id)
	return err
}

func (m *Model) junkMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.JunkMessage(ctx, id)
	return err
}

func (m *Model) notJunkMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.NotJunkMessage(ctx, id)
	return err
}

func (m *Model) restoreMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.RestoreMessage(ctx, id)
	return err
}

func (m *Model) blockSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.BlockSender(ctx, id)
	return err
}

func (m *Model) unblockSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UnblockSender(ctx, id)
	return err
}

func (m *Model) blockDomain(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.BlockDomain(ctx, id)
	return err
}

func (m *Model) unblockDomain(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UnblockDomain(ctx, id)
	return err
}

func (m *Model) trustSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.TrustSender(ctx, id)
	return err
}

func (m *Model) untrustSender(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UntrustSender(ctx, id)
	return err
}

func (m *Model) syncMail(ctx context.Context) (screens.MailSyncResult, error) {
	service, err := m.mailService()
	if err != nil {
		return screens.MailSyncResult{}, err
	}
	result, err := mailsync.Run(ctx, mailstore.New(m.dataPath), service, "")
	return screens.MailSyncResult{
		ActiveMailboxes:  result.ActiveMailboxes,
		SkippedMailboxes: result.SkippedMailboxes,
		OutboxItems:      result.OutboxItems,
		DraftItems:       result.DraftItems,
		InboxMessages:    result.InboxMessages,
		BodyErrors:       result.BodyErrors,
		InboxErrors:      result.InboxErrors,
	}, err
}

func (m *Model) sendDraft(ctx context.Context, mailbox mailstore.MailboxMeta, draft mailstore.Draft) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = mailsend.SendDraft(ctx, mailstore.New(m.dataPath), service, mailbox, draft)
	return err
}

func (m *Model) updateDraft(ctx context.Context, draft mailstore.Draft) error {
	if draft.Meta.RemoteID == 0 {
		return nil
	}
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.UpdateOutboundMessage(ctx, draft.Meta.RemoteID, outboundInputFromDraft(draft))
	return err
}

func (m *Model) deleteDraft(ctx context.Context, draft mailstore.Draft) error {
	if draft.Meta.RemoteID == 0 {
		return nil
	}
	service, err := m.mailService()
	if err != nil {
		return err
	}
	return service.DeleteOutboundMessage(ctx, draft.Meta.RemoteID)
}

func outboundInputFromDraft(draft mailstore.Draft) *mail.OutboundMessageInput {
	domainID := draft.Meta.DomainID
	inboxID := draft.Meta.InboxID
	return &mail.OutboundMessageInput{
		DomainID:        &domainID,
		InboxID:         &inboxID,
		SourceMessageID: int64Ptr(draft.Meta.SourceMessageID),
		ConversationID:  int64Ptr(draft.Meta.ConversationID),
		ToAddresses:     draft.Meta.To,
		CCAddresses:     draft.Meta.CC,
		BCCAddresses:    draft.Meta.BCC,
		Subject:         draft.Meta.Subject,
		Body:            draft.Body,
	}
}

func int64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func (m *Model) forwardMessage(ctx context.Context, id int64, draft mailstore.Draft) (int64, string, error) {
	service, err := m.mailService()
	if err != nil {
		return 0, "", err
	}
	outbound, err := service.Forward(ctx, id, draft.Meta.To)
	if err != nil {
		return 0, "", err
	}
	outbound, err = service.UpdateOutboundMessage(ctx, outbound.ID, outboundInputFromDraft(draft))
	if err != nil {
		return 0, "", err
	}
	store := mailstore.New(m.dataPath)
	mailboxes, _ := store.ListMailboxes()
	for _, mailbox := range mailboxes {
		if mailbox.InboxID == outbound.InboxID || mailbox.DomainID == outbound.DomainID {
			_, _ = store.StoreRemoteDraft(mailbox, *outbound, time.Now())
			break
		}
	}
	return outbound.ID, outbound.Status, nil
}

func (m *Model) downloadAttachment(ctx context.Context, attachment mailstore.AttachmentMeta) ([]byte, error) {
	if attachment.DownloadURL == "" {
		return nil, fmt.Errorf("attachment has no download URL")
	}
	if _, err := m.mailService(); err != nil {
		return nil, err
	}
	body, _, err := m.client.Download(ctx, attachment.DownloadURL)
	return body, err
}

func (m *Model) searchMail(ctx context.Context, params screens.MailSearchParams) ([]mailstore.CachedMessage, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	messages, _, err := service.ListMessages(ctx, mail.MessageListParams{
		ListParams: mail.ListParams{Page: params.Page, PerPage: params.PerPage},
		InboxID:    params.InboxID,
		Mailbox:    params.Mailbox,
		Query:      params.Query,
		Sort:       params.Sort,
	})
	if err != nil {
		return nil, err
	}
	results := make([]mailstore.CachedMessage, 0, len(messages))
	for _, message := range messages {
		results = append(results, cachedRemoteMessage(message))
	}
	return results, nil
}

func (m *Model) conversationTimeline(ctx context.Context, id int64) ([]screens.ConversationEntry, error) {
	service, err := m.mailService()
	if err != nil {
		return nil, err
	}
	entries, err := service.ConversationTimeline(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]screens.ConversationEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, screens.ConversationEntry{
			Kind:           entry.Kind,
			RecordID:       entry.RecordID,
			OccurredAt:     entry.OccurredAt,
			Sender:         entry.Sender,
			Recipients:     entry.Recipients,
			Summary:        entry.Summary,
			Status:         entry.Status,
			Subject:        entry.Subject,
			ConversationID: entry.ConversationID,
		})
	}
	return out, nil
}

func (m *Model) conversationBody(ctx context.Context, entry screens.ConversationEntry) (string, error) {
	service, err := m.mailService()
	if err != nil {
		return "", err
	}
	if entry.Kind == "outbound" {
		message, err := service.ShowOutboundMessage(ctx, entry.RecordID)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(message.BodyText) != "" {
			return message.BodyText, nil
		}
		return message.BodyHTML, nil
	}
	body, err := service.MessageBody(ctx, entry.RecordID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(body.Text) != "" {
		return body.Text, nil
	}
	return body.HTML, nil
}

func (m *Model) syncDrive(ctx context.Context) (screens.DriveSyncResult, error) {
	service, cfg, err := m.driveService()
	if err != nil {
		return screens.DriveSyncResult{}, err
	}
	result, err := drivesync.Run(ctx, drivestore.New(m.dataPath), service, cfg.DriveSyncMode())
	return screens.DriveSyncResult{Folders: result.Folders, Files: result.Files, DownloadedFiles: result.DownloadedFiles, DownloadFailures: result.DownloadFailures}, err
}

func (m *Model) downloadDriveFile(ctx context.Context, meta drivestore.FileMeta) ([]byte, error) {
	service, _, err := m.driveService()
	if err != nil {
		return nil, err
	}
	remote, err := service.ShowFile(ctx, meta.RemoteID)
	if err != nil {
		return nil, err
	}
	return service.DownloadFile(ctx, *remote)
}

func (m *Model) openDriveFile(path string) error {
	opener := os.Getenv("OPENER")
	if opener != "" {
		return startDetached(openerCommand(opener, path))
	}
	if textFile(path) {
		if editor := os.Getenv("VISUAL"); editor != "" {
			if cmd := terminalCommand(editor, path); cmd != nil {
				return startDetached(cmd)
			}
		}
		if editor := os.Getenv("EDITOR"); editor != "" {
			if cmd := terminalCommand(editor, path); cmd != nil {
				return startDetached(cmd)
			}
		}
	}
	return startDetached(exec.Command("xdg-open", path))
}

func openerCommand(opener, path string) *exec.Cmd {
	parts := strings.Fields(opener)
	if len(parts) == 0 {
		return exec.Command("xdg-open", path)
	}
	return exec.Command(parts[0], append(parts[1:], path)...)
}

func terminalCommand(editor, path string) *exec.Cmd {
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return nil
	}
	terminal := os.Getenv("TERMINAL")
	if terminal != "" {
		if cmd := terminalCommandFor(terminal, editorParts, path); cmd != nil {
			return cmd
		}
	}
	for _, candidate := range []string{"ghostty", "alacritty", "kitty"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return terminalCommandFor(candidate, editorParts, path)
		}
	}
	return nil
}

func terminalCommandFor(terminal string, editorParts []string, path string) *exec.Cmd {
	terminalParts := strings.Fields(terminal)
	if len(terminalParts) == 0 {
		return nil
	}
	args := append([]string{}, terminalParts[1:]...)
	args = append(args, "-e")
	args = append(args, editorParts...)
	args = append(args, path)
	return exec.Command(terminalParts[0], args...)
}

func startDetached(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func textFile(path string) bool {
	lower := strings.ToLower(path)
	for _, suffix := range []string{".md", ".markdown", ".txt", ".text", ".log", ".csv", ".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".cfg"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func (m *Model) uploadDriveFile(ctx context.Context, path string, folderID *int64) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.UploadFile(ctx, path, folderID); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) createDriveFolder(ctx context.Context, input drive.FolderInput) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.CreateFolder(ctx, input); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) renameDriveFile(ctx context.Context, id int64, input drive.FileInput) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.UpdateFile(ctx, id, input); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) renameDriveFolder(ctx context.Context, id int64, input drive.FolderInput) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.UpdateFolder(ctx, id, input); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) deleteDriveFile(ctx context.Context, id int64) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if err := service.DeleteFile(ctx, id); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) deleteDriveFolder(ctx context.Context, id int64) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if err := service.DeleteFolder(ctx, id); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) syncNotes(ctx context.Context) (screens.NotesSyncResult, error) {
	service, err := m.notesService()
	if err != nil {
		return screens.NotesSyncResult{}, err
	}
	result, err := runNotesSync(ctx, notestore.New(m.dataPath), service)
	return screens.NotesSyncResult{Folders: result.Folders, Notes: result.Notes}, err
}

func (m *Model) syncContacts(ctx context.Context) (screens.ContactsSyncResult, error) {
	service, err := m.contactsService()
	if err != nil {
		return screens.ContactsSyncResult{}, err
	}
	result, err := runContactsSync(ctx, contactstore.New(m.dataPath), service)
	return screens.ContactsSyncResult{Contacts: result.Contacts, Notes: result.Notes}, err
}

func (m *Model) deleteContact(ctx context.Context, id int64) error {
	service, err := m.contactsService()
	if err != nil {
		return err
	}
	if err := service.DeleteContact(ctx, id); err != nil {
		return err
	}
	return contactstore.New(m.dataPath).DeleteContact(id)
}

func (m *Model) loadContactNote(ctx context.Context, id int64) (*contacts.ContactNote, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	note, err := service.ContactNote(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreContactNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) updateContactNote(ctx context.Context, id int64, input contacts.ContactNoteInput) (*contacts.ContactNote, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	note, err := service.UpdateContactNote(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreContactNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) loadContactCommunications(ctx context.Context, id int64) ([]contacts.ContactCommunication, error) {
	service, err := m.contactsService()
	if err != nil {
		return nil, err
	}
	communications, _, err := service.ContactCommunications(ctx, id, contacts.ListParams{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	if err := contactstore.New(m.dataPath).StoreCommunications(id, communications); err != nil {
		return nil, err
	}
	return communications, nil
}

func (m *Model) createNote(ctx context.Context, input notes.NoteInput) (*notes.Note, error) {
	service, err := m.notesService()
	if err != nil {
		return nil, err
	}
	note, err := service.CreateNote(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := notestore.New(m.dataPath).StoreNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) updateNote(ctx context.Context, id int64, input notes.NoteInput) (*notes.Note, error) {
	service, err := m.notesService()
	if err != nil {
		return nil, err
	}
	note, err := service.UpdateNote(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := notestore.New(m.dataPath).StoreNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) deleteNote(ctx context.Context, id int64) error {
	service, err := m.notesService()
	if err != nil {
		return err
	}
	if err := service.DeleteNote(ctx, id); err != nil {
		return err
	}
	return notestore.New(m.dataPath).DeleteNote(id)
}

func (m *Model) syncCalendar(ctx context.Context, from, to string) (screens.CalendarSyncResult, error) {
	service, err := m.calendarService()
	if err != nil {
		return screens.CalendarSyncResult{}, err
	}
	result, err := runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{From: from, To: to})
	return screens.CalendarSyncResult{Calendars: result.Calendars, Events: result.Events, Occurrences: result.Occurrences}, err
}

func (m *Model) createCalendar(ctx context.Context, input calendar.CalendarInput) (*calendar.Calendar, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	created, err := service.CreateCalendar(ctx, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreCalendar(*created, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return created, err
}

func (m *Model) updateCalendar(ctx context.Context, id int64, input calendar.CalendarInput) (*calendar.Calendar, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	updated, err := service.UpdateCalendar(ctx, id, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreCalendar(*updated, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return updated, err
}

func (m *Model) deleteCalendar(ctx context.Context, id int64) error {
	service, err := m.calendarService()
	if err != nil {
		return err
	}
	if err := service.DeleteCalendar(ctx, id); err != nil {
		return err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.DeleteCalendar(id); err != nil {
		return err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return err
}

func (m *Model) importCalendarICS(ctx context.Context, calendarID int64, path string) (*calendar.ImportResult, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	result, err := service.ImportICS(ctx, calendarID, path)
	if err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{})
	return result, err
}

func (m *Model) showCalendarInvitation(ctx context.Context, messageID int64) (*calendar.Invitation, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	invite, err := service.ShowInvitation(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if invite.CalendarEvent != nil {
		if err := calendarstore.New(m.dataPath).StoreEvent(*invite.CalendarEvent, time.Now()); err != nil {
			return nil, err
		}
	}
	return invite, nil
}

func (m *Model) syncCalendarInvitation(ctx context.Context, messageID int64) (*calendar.Invitation, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	invite, err := service.SyncInvitation(ctx, messageID)
	if err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{})
	return invite, err
}

func (m *Model) respondCalendarInvitation(ctx context.Context, messageID int64, input calendar.InvitationInput) (*calendar.Invitation, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	invite, err := service.UpdateInvitation(ctx, messageID, input)
	if err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{})
	return invite, err
}

func (m *Model) createCalendarEvent(ctx context.Context, input calendar.CalendarEventInput) (*calendar.CalendarEvent, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	event, err := service.CreateEvent(ctx, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreEvent(*event, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return event, err
}

func (m *Model) updateCalendarEvent(ctx context.Context, id int64, input calendar.CalendarEventInput) (*calendar.CalendarEvent, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	event, err := service.UpdateEvent(ctx, id, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreEvent(*event, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return event, err
}

func (m *Model) deleteCalendarEvent(ctx context.Context, id int64) error {
	service, err := m.calendarService()
	if err != nil {
		return err
	}
	if err := service.DeleteEvent(ctx, id); err != nil {
		return err
	}
	return calendarstore.New(m.dataPath).DeleteEvent(id)
}

type calendarSyncResult struct {
	Calendars   int
	Events      int
	Occurrences int
}

type calendarSyncOptions struct {
	From       string
	To         string
	CalendarID int64
}

func runCalendarSync(ctx context.Context, store calendarstore.Store, service *calendar.Service, opts calendarSyncOptions) (calendarSyncResult, error) {
	syncedAt := time.Now()
	calendars, _, err := service.ListCalendars(ctx, calendar.ListParams{Page: 1, PerPage: 100})
	if err != nil {
		return calendarSyncResult{}, err
	}
	var result calendarSyncResult
	for _, item := range calendars {
		if opts.CalendarID > 0 && item.ID != opts.CalendarID {
			continue
		}
		if err := store.StoreCalendar(item, syncedAt); err != nil {
			return calendarSyncResult{}, err
		}
		result.Calendars++
		page := 1
		for {
			events, pagination, err := service.ListEvents(ctx, calendar.EventListParams{ListParams: calendar.ListParams{Page: page, PerPage: 100}, CalendarID: item.ID, Sort: "starts_at"})
			if err != nil {
				return calendarSyncResult{}, err
			}
			for _, event := range events {
				messages, err := service.EventMessages(ctx, event.ID)
				if err != nil {
					return calendarSyncResult{}, err
				}
				event.Messages = messages
				if err := store.StoreEvent(event, syncedAt); err != nil {
					return calendarSyncResult{}, err
				}
				result.Events++
			}
			if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
				break
			}
			page++
		}
	}
	from, to := calendarDefaultRange(opts.From, opts.To)
	occurrences, err := service.ListOccurrences(ctx, calendar.OccurrenceListParams{CalendarID: opts.CalendarID, StartsFrom: from, EndsTo: to})
	if err != nil {
		return calendarSyncResult{}, err
	}
	if err := store.StoreOccurrences(occurrences, syncedAt); err != nil {
		return calendarSyncResult{}, err
	}
	result.Occurrences = len(occurrences)
	return result, nil
}

func calendarDefaultRange(from, to string) (string, string) {
	now := time.Now()
	if from == "" {
		from = now.Format("2006-01-02")
	}
	if to == "" {
		to = now.AddDate(0, 0, 30).Format("2006-01-02")
	}
	return from, to
}

type notesSyncResult struct {
	Folders int
	Notes   int
}

func runNotesSync(ctx context.Context, store notestore.Store, service *notes.Service) (notesSyncResult, error) {
	tree, err := service.NotesTree(ctx)
	if err != nil {
		return notesSyncResult{}, err
	}
	syncedAt := time.Now()
	if err := store.StoreTree(tree, syncedAt); err != nil {
		return notesSyncResult{}, err
	}
	var result notesSyncResult
	if err := syncNotesFolder(ctx, store, service, *tree, syncedAt, &result); err != nil {
		return notesSyncResult{}, err
	}
	return result, nil
}

func syncNotesFolder(ctx context.Context, store notestore.Store, service *notes.Service, folder notes.FolderTree, syncedAt time.Time, result *notesSyncResult) error {
	result.Folders++
	page := 1
	for {
		cached, pagination, err := service.ListNotes(ctx, notes.ListNotesParams{ListParams: notes.ListParams{Page: page, PerPage: 100}, FolderID: &folder.ID, Sort: "filename"})
		if err != nil {
			return err
		}
		for _, note := range cached {
			if err := store.StoreNote(note, syncedAt); err != nil {
				return err
			}
			result.Notes++
		}
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
			break
		}
		page++
	}
	for _, child := range folder.Children {
		if err := syncNotesFolder(ctx, store, service, child, syncedAt, result); err != nil {
			return err
		}
	}
	return nil
}

func cachedRemoteMessage(message mail.Message) mailstore.CachedMessage {
	return mailstore.CachedMessage{
		Meta: mailstore.MessageMeta{
			SchemaVersion:  mailstore.SchemaVersion,
			Kind:           "remote-message",
			RemoteID:       message.ID,
			ConversationID: message.ConversationID,
			InboxID:        message.InboxID,
			Mailbox:        message.SystemState,
			Status:         message.Status,
			Subject:        message.Subject,
			FromAddress:    message.FromAddress,
			FromName:       message.FromName,
			To:             message.ToAddresses,
			CC:             message.CCAddresses,
			Read:           message.Read,
			Starred:        message.Starred,
			SenderBlocked:  message.SenderBlocked,
			SenderTrusted:  message.SenderTrusted,
			DomainBlocked:  message.DomainBlocked,
			Labels:         remoteLabelMetas(message.Labels),
			Attachments:    remoteAttachmentMetas(message.Attachments),
			ReceivedAt:     message.ReceivedAt,
			SyncedAt:       time.Now(),
		},
		Path:     fmt.Sprintf("remote:%d", message.ID),
		BodyText: message.TextBody,
	}
}

func remoteLabelMetas(labels []mail.Label) []mailstore.LabelMeta {
	metas := make([]mailstore.LabelMeta, 0, len(labels))
	for _, label := range labels {
		metas = append(metas, mailstore.LabelMeta{ID: label.ID, Name: label.Name, Color: label.Color})
	}
	return metas
}

func remoteAttachmentMetas(attachments []mail.Attachment) []mailstore.AttachmentMeta {
	metas := make([]mailstore.AttachmentMeta, 0, len(attachments))
	for _, attachment := range attachments {
		metas = append(metas, mailstore.AttachmentMeta{
			ID:          attachment.ID,
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			ByteSize:    attachment.ByteSize,
			Previewable: attachment.Previewable,
			PreviewKind: attachment.PreviewKind,
			PreviewURL:  attachment.PreviewURL,
			DownloadURL: attachment.DownloadURL,
		})
	}
	return metas
}

func (m *Model) mailService() (*mail.Service, error) {
	if m.client == nil {
		configFile, tokenFile := config.Paths(m.configPath)
		cfg, err := config.LoadFrom(configFile)
		if err != nil {
			return nil, err
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		m.client = api.NewClient(cfg, tokenFile)
	}
	return mail.NewService(m.client), nil
}

func (m *Model) driveService() (*drive.Service, *config.Config, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return drive.NewService(m.client), cfg, nil
}

func (m *Model) notesService() (*notes.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return notes.NewService(m.client), nil
}

func (m *Model) contactsService() (*contacts.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return contacts.NewService(m.client), nil
}

func (m *Model) calendarService() (*calendar.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return calendar.NewService(m.client), nil
}

func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		if id == "mail-admin" || isAggregateMailScreen(id) || isHackerNewsScreen(id) {
			continue
		}
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"home", "mail", "calendar", "contacts", "notes", "drive", "news", "settings", "logs"}
	ordered := make([]string, 0, len(m.screenOrder))
	seen := make(map[string]bool)
	for _, id := range preferred {
		if _, ok := m.screens[id]; ok {
			ordered = append(ordered, id)
			seen[id] = true
		}
	}
	for _, id := range m.screenOrder {
		if !seen[id] {
			ordered = append(ordered, id)
		}
	}
	m.screenOrder = ordered
}

func isAggregateMailScreen(id string) bool {
	for _, scope := range aggregateMailScreens() {
		if id == scope.id {
			return true
		}
	}
	return false
}

func isMailScreen(id string) bool {
	return id == "mail" || isAggregateMailScreen(id)
}

func (m *Model) registerCommands() {
	route := func(id string) func() tea.Cmd {
		return func() tea.Cmd { return func() tea.Msg { return routeMsg{id} } }
	}
	mailAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.MailActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"mail"} }, actionMsg)
		}
	}
	driveAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.DriveActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"drive"} }, actionMsg)
		}
	}
	calendarAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.CalendarActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"calendar"} }, actionMsg)
		}
	}
	mailAdminAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.MailAdminActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"mail-admin"} }, actionMsg)
		}
	}
	notesAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.NotesActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"notes"} }, actionMsg)
		}
	}
	contactsAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.ContactsActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"contacts"} }, actionMsg)
		}
	}
	onMail := func(ctx commands.Context) bool { return isMailScreen(ctx.ActiveScreen) }
	onMailAdmin := func(ctx commands.Context) bool { return ctx.ActiveScreen == "mail-admin" }
	onMailOrAdmin := func(ctx commands.Context) bool {
		return isMailScreen(ctx.ActiveScreen) || ctx.ActiveScreen == "mail-admin"
	}
	onCalendarAgenda := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar-event"
	}
	onCalendarItem := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar-event" && ctx.Selection.HasItems
	}
	onCalendarInvitation := func(ctx commands.Context) bool {
		return onCalendarItem(ctx) && ctx.Selection.HasInvitation
	}
	onCalendarManagement := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar"
	}
	onCalendarCalendar := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar" && ctx.Selection.HasItems
	}
	onDrive := func(ctx commands.Context) bool { return ctx.ActiveScreen == "drive" }
	onNotes := func(ctx commands.Context) bool { return ctx.ActiveScreen == "notes" }
	onContacts := func(ctx commands.Context) bool { return ctx.ActiveScreen == "contacts" }
	onContactItem := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "contacts" && ctx.Selection != nil && ctx.Selection.Kind == "contact" && ctx.Selection.HasItems
	}
	onNotesItem := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "notes" && ctx.Selection != nil && ctx.Selection.Kind == "note" && ctx.Selection.HasItems
	}
	onMailDrafts := func(ctx commands.Context) bool {
		return isMailScreen(ctx.ActiveScreen) && ctx.Selection != nil && ctx.Selection.IsDraft && ctx.Selection.HasItems
	}
	onMailMessages := func(ctx commands.Context) bool {
		return isMailScreen(ctx.ActiveScreen) && ctx.Selection != nil && ctx.Selection.Kind == "message" && ctx.Selection.HasItems
	}
	subjectDescribe := func(prefix string) func(commands.Context) string {
		return func(ctx commands.Context) string {
			if ctx.Selection != nil && ctx.Selection.Subject != "" {
				return prefix + " · " + ctx.Selection.Subject
			}
			return prefix
		}
	}

	// Navigation
	m.commands.Register(commands.Command{ID: "go-home", Module: commands.ModuleGlobal, Title: "Go to Home", Keywords: []string{"home", "start"}, Run: route("home")})
	m.commands.Register(commands.Command{ID: "go-mail", Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open Mail", Description: "Unread across all mailboxes", Keywords: []string{"mail", "email", "unread", "inbox"}, Pinned: true, Run: route("mail-unread")})
	m.commands.Register(commands.Command{ID: "go-mailboxes", Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open Mailboxes", Description: "Browse one mailbox at a time", Keywords: []string{"mail", "email", "mailboxes", "accounts"}, Pinned: true, Run: route("mail")})
	for _, scope := range aggregateMailScreens() {
		m.commands.Register(commands.Command{ID: "go-" + scope.id, Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open " + scope.title, Keywords: []string{"mail", "email", strings.ToLower(scope.title)}, Run: route(scope.id)})
	}
	m.commands.Register(commands.Command{ID: "go-mail-admin", Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open Mail Admin", Description: "Manage domains and inboxes", Keywords: []string{"mail", "admin", "domains", "inboxes"}, Run: route("mail-admin")})
	m.commands.Register(commands.Command{ID: "go-calendar", Module: commands.ModuleCalendar, Title: "Open Calendar", Keywords: []string{"calendar", "events", "agenda"}, Pinned: true, Run: route("calendar")})
	m.commands.Register(commands.Command{ID: "go-contacts", Module: commands.ModuleContacts, Title: "Open Contacts", Keywords: []string{"contacts", "crm", "people"}, Pinned: true, Run: route("contacts")})
	m.commands.Register(commands.Command{ID: "go-notes", Module: commands.ModuleNotes, Title: "Open Notes", Keywords: []string{"notes", "markdown", "memo"}, Pinned: true, Run: route("notes")})
	m.commands.Register(commands.Command{ID: "go-drive", Module: commands.ModuleDrive, Title: "Open Drive", Description: "Local Drive mirror", Keywords: []string{"drive", "files", "documents"}, Pinned: true, Run: route("drive")})
	m.commands.Register(commands.Command{ID: "go-settings", Module: commands.ModuleSettings, Title: "Open Settings", Keywords: []string{"settings", "config"}, Pinned: true, Run: route("settings")})
	m.registerHackerNewsCommands()
	if m.devBuild() {
		m.commands.Register(commands.Command{ID: "go-logs", Module: commands.ModuleGlobal, Title: "Open Logs", Description: "Debug event log", Keywords: []string{"logs", "debug", "events"}, Run: route("logs")})
	}

	// Mail — module-level
	m.commands.Register(commands.Command{ID: "mail-sync", Module: commands.ModuleMail, Title: "Sync mailbox", Description: "Pull latest messages, drafts, outbox", Keywords: []string{"sync", "refresh"}, Available: onMailOrAdmin, Run: mailAction("sync", true)})
	m.commands.Register(commands.Command{ID: "mail-admin-refresh", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "Refresh Mail Admin", Description: "Reload remote domains and inboxes", Keywords: []string{"refresh", "reload", "domains", "inboxes"}, Available: onMailAdmin, Run: mailAdminAction("refresh", true)})
	m.commands.Register(commands.Command{ID: "domains-new", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "New domain", Description: "Create a managed mail domain", Keywords: []string{"domain", "new", "create"}, Available: onMailAdmin, Run: mailAdminAction("new-domain", true)})
	m.commands.Register(commands.Command{ID: "domains-validate", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "Validate selected domain", Description: "Check outbound settings", Keywords: []string{"domain", "validate", "smtp", "outbound"}, Available: onMailAdmin, Run: mailAdminAction("validate-domain", true)})
	m.commands.Register(commands.Command{ID: "inboxes-new", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "New inbox", Description: "Create an inbox on the selected domain", Keywords: []string{"inbox", "new", "create"}, Available: onMailAdmin, Run: mailAdminAction("new-inbox", true)})
	m.commands.Register(commands.Command{ID: "inboxes-pipeline", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "Show inbox pipeline", Description: "Pipeline metadata for selected inbox", Keywords: []string{"inbox", "pipeline"}, Available: onMailAdmin, Run: mailAdminAction("pipeline", true)})

	// Calendar — module-level
	m.commands.Register(commands.Command{ID: "calendar-sync", Module: commands.ModuleCalendar, Title: "Sync Calendar", Description: "Pull latest calendars, events, and occurrences", Shortcut: "S", Keywords: []string{"sync", "refresh", "agenda"}, Run: calendarAction("sync", true)})
	m.commands.Register(commands.Command{ID: "calendar-view-agenda", Module: commands.ModuleCalendar, Title: "View agenda", Description: "Show cached calendar occurrences", Keywords: []string{"agenda", "occurrences", "events"}, Available: onCalendarManagement, Run: calendarAction("view-agenda", false)})
	m.commands.Register(commands.Command{ID: "calendar-view-calendars", Module: commands.ModuleCalendar, Title: "View calendars", Description: "Show cached calendars", Shortcut: "v", Keywords: []string{"calendars", "list", "manage"}, Available: onCalendarAgenda, Run: calendarAction("view-calendars", false)})
	m.commands.Register(commands.Command{ID: "calendar-new", Module: commands.ModuleCalendar, Title: "New event", Description: "Create a calendar event", Shortcut: "n", Keywords: []string{"new", "create", "event"}, Available: onCalendarAgenda, Run: calendarAction("new", false)})
	m.commands.Register(commands.Command{ID: "calendar-edit", Module: commands.ModuleCalendar, Title: "Edit selected event", Description: "Edit the highlighted calendar event", Shortcut: "e", Keywords: []string{"edit", "update", "event"}, Available: onCalendarItem, Describe: subjectDescribe("Edit the highlighted calendar event"), Run: calendarAction("edit", false)})
	m.commands.Register(commands.Command{ID: "calendar-today", Module: commands.ModuleCalendar, Title: "Jump to today", Description: "Move selection to the next occurrence today or later", Shortcut: "t", Keywords: []string{"today", "agenda"}, Available: onCalendarAgenda, Run: calendarAction("today", false)})
	m.commands.Register(commands.Command{ID: "calendar-previous-range", Module: commands.ModuleCalendar, Title: "Previous agenda range", Description: "Move the agenda to the previous cached date range", Shortcut: "[", Keywords: []string{"previous", "prev", "range", "agenda", "calendar"}, Available: onCalendarAgenda, Run: calendarAction("previous-range", false)})
	m.commands.Register(commands.Command{ID: "calendar-next-range", Module: commands.ModuleCalendar, Title: "Next agenda range", Description: "Move the agenda to the next cached date range", Shortcut: "]", Keywords: []string{"next", "range", "agenda", "calendar"}, Available: onCalendarAgenda, Run: calendarAction("next-range", false)})
	m.commands.Register(commands.Command{ID: "calendar-filter-agenda", Module: commands.ModuleCalendar, Title: "Filter agenda", Description: "Filter agenda by calendar, status, source, title, or location", Shortcut: "/", Keywords: []string{"filter", "search", "agenda", "calendar", "status", "source"}, Available: onCalendarAgenda, Run: calendarAction("filter", false)})
	m.commands.Register(commands.Command{ID: "calendar-clear-agenda-filters", Module: commands.ModuleCalendar, Title: "Clear agenda filters", Description: "Show all cached agenda occurrences", Shortcut: "ctrl+l", Keywords: []string{"clear", "filter", "search", "agenda"}, Available: onCalendarAgenda, Run: calendarAction("clear-filter", false)})
	m.commands.Register(commands.Command{ID: "calendar-delete", Module: commands.ModuleCalendar, Title: "Delete selected event", Description: "Delete the highlighted calendar event after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "event"}, Available: onCalendarItem, Describe: subjectDescribe("Delete the highlighted calendar event"), Run: calendarAction("delete", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-show", Module: commands.ModuleCalendar, Title: "Show invitation details", Description: "Load invitation details for the linked message", Keywords: []string{"invitation", "invite", "details", "message"}, Available: onCalendarInvitation, Describe: subjectDescribe("Load invitation details for"), Run: calendarAction("invitation-show", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-sync", Module: commands.ModuleCalendar, Title: "Sync selected invitation", Description: "Sync the linked invitation message into Calendar", Keywords: []string{"invitation", "invite", "sync", "message"}, Available: onCalendarInvitation, Describe: subjectDescribe("Sync invitation for"), Run: calendarAction("invitation-sync", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-accept", Module: commands.ModuleCalendar, Title: "Accept invitation", Description: "Respond accepted to the linked invitation", Keywords: []string{"invitation", "invite", "accept", "accepted", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Accept invitation for"), Run: calendarAction("invitation-accepted", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-tentative", Module: commands.ModuleCalendar, Title: "Tentatively accept invitation", Description: "Respond tentative to the linked invitation", Keywords: []string{"invitation", "invite", "tentative", "maybe", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Tentatively accept invitation for"), Run: calendarAction("invitation-tentative", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-decline", Module: commands.ModuleCalendar, Title: "Decline invitation", Description: "Respond declined to the linked invitation", Keywords: []string{"invitation", "invite", "decline", "declined", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Decline invitation for"), Run: calendarAction("invitation-declined", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-needs-action", Module: commands.ModuleCalendar, Title: "Mark invitation needs action", Description: "Respond needs_action to the linked invitation", Keywords: []string{"invitation", "invite", "needs_action", "needs action", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Mark invitation needs action for"), Run: calendarAction("invitation-needs-action", false)})
	m.commands.Register(commands.Command{ID: "calendars-new", Module: commands.ModuleCalendar, Title: "New calendar", Description: "Create a calendar", Shortcut: "n", Keywords: []string{"new", "create", "calendar"}, Available: onCalendarManagement, Run: calendarAction("new-calendar", false)})
	m.commands.Register(commands.Command{ID: "calendars-edit", Module: commands.ModuleCalendar, Title: "Edit selected calendar", Description: "Edit the highlighted calendar", Shortcut: "e", Keywords: []string{"edit", "update", "calendar"}, Available: onCalendarCalendar, Describe: subjectDescribe("Edit the highlighted calendar"), Run: calendarAction("edit-calendar", false)})
	m.commands.Register(commands.Command{ID: "calendars-import-ics", Module: commands.ModuleCalendar, Title: "Import ICS into selected calendar", Description: "Pick an .ics file and import it into the highlighted calendar", Shortcut: "i", Keywords: []string{"import", "ics", "calendar"}, Available: onCalendarCalendar, Describe: subjectDescribe("Import ICS into calendar"), Run: calendarAction("import-ics", false)})
	m.commands.Register(commands.Command{ID: "calendars-delete", Module: commands.ModuleCalendar, Title: "Delete selected calendar", Description: "Delete the highlighted calendar after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "calendar"}, Available: onCalendarCalendar, Describe: subjectDescribe("Delete the highlighted calendar"), Run: calendarAction("delete-calendar", false)})

	// Drive — module-level
	m.commands.Register(commands.Command{ID: "drive-sync", Module: commands.ModuleDrive, Title: "Sync Drive", Description: "Pull latest Drive metadata and files", Shortcut: "S", Keywords: []string{"sync", "refresh"}, Run: driveAction("sync", true)})
	m.commands.Register(commands.Command{ID: "drive-upload", Module: commands.ModuleDrive, Title: "Upload file", Description: "Upload a local file into the current Drive folder", Shortcut: "u", Keywords: []string{"upload", "file"}, Available: onDrive, Run: driveAction("upload", false)})
	m.commands.Register(commands.Command{ID: "drive-new-folder", Module: commands.ModuleDrive, Title: "New folder", Description: "Create a folder in the current Drive folder", Shortcut: "n", Keywords: []string{"new", "folder", "create"}, Available: onDrive, Run: driveAction("new-folder", false)})
	m.commands.Register(commands.Command{ID: "drive-rename", Module: commands.ModuleDrive, Title: "Rename selected", Description: "Rename the highlighted Drive item", Shortcut: "R", Keywords: []string{"rename"}, Available: onDrive, Run: driveAction("rename", false)})
	m.commands.Register(commands.Command{ID: "drive-delete", Module: commands.ModuleDrive, Title: "Delete selected", Description: "Delete the highlighted Drive item after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove"}, Available: onDrive, Run: driveAction("delete", false)})
	m.commands.Register(commands.Command{ID: "drive-details", Module: commands.ModuleDrive, Title: "Show details", Description: "Toggle details for the highlighted Drive item", Shortcut: "i", Keywords: []string{"details", "info"}, Available: onDrive, Run: driveAction("details", false)})

	// Notes — module-level
	m.commands.Register(commands.Command{ID: "notes-sync", Module: commands.ModuleNotes, Title: "Sync Notes", Description: "Pull latest Notes folders and note bodies", Shortcut: "S", Keywords: []string{"sync", "refresh"}, Run: notesAction("sync", true)})
	m.commands.Register(commands.Command{ID: "notes-new", Module: commands.ModuleNotes, Title: "New note", Description: "Create a note in the current Notes folder", Shortcut: "n", Keywords: []string{"new", "create", "write"}, Available: onNotes, Run: notesAction("new", false)})
	m.commands.Register(commands.Command{ID: "notes-edit", Module: commands.ModuleNotes, Title: "Edit selected note", Description: "Open the highlighted note in TELEX_NOTES_EDITOR, VISUAL, or EDITOR", Shortcut: "e", Keywords: []string{"edit", "write"}, Available: onNotesItem, Run: notesAction("edit", false)})
	m.commands.Register(commands.Command{ID: "notes-delete", Module: commands.ModuleNotes, Title: "Delete selected note", Description: "Delete the highlighted note after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove"}, Available: onNotesItem, Run: notesAction("delete", false)})
	m.commands.Register(commands.Command{ID: "notes-search", Module: commands.ModuleNotes, Title: "Search current Notes folder", Description: "Filter notes and folders in the current Notes folder", Shortcut: "/", Keywords: []string{"search", "filter"}, Available: onNotes, Run: notesAction("search", false)})
	m.commands.Register(commands.Command{ID: "notes-toggle-sort", Module: commands.ModuleNotes, Title: "Toggle Notes sort order", Description: "Cycle Notes sort between A-Z and most recently updated", Shortcut: "o", Keywords: []string{"sort", "order", "recent"}, Available: onNotes, Run: notesAction("toggle-sort", false)})
	m.commands.Register(commands.Command{ID: "notes-toggle-flat", Module: commands.ModuleNotes, Title: "Toggle Notes flat view", Description: "Show all notes flat across folders, or revert to folder navigation", Shortcut: "f", Keywords: []string{"flat", "all", "view"}, Available: onNotes, Run: notesAction("toggle-flat", false)})

	// Contacts — module-level
	m.commands.Register(commands.Command{ID: "contacts-sync", Module: commands.ModuleContacts, Title: "Sync Contacts", Description: "Pull latest Contacts and notes", Shortcut: "S", Keywords: []string{"sync", "refresh", "crm"}, Run: contactsAction("sync", true)})
	m.commands.Register(commands.Command{ID: "contacts-search", Module: commands.ModuleContacts, Title: "Search Contacts", Description: "Filter cached contacts", Shortcut: "/", Keywords: []string{"search", "filter", "contacts"}, Available: onContacts, Run: contactsAction("search", false)})
	m.commands.Register(commands.Command{ID: "contacts-delete", Module: commands.ModuleContacts, Title: "Delete selected contact", Description: "Delete the highlighted contact after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "contact"}, Available: onContactItem, Describe: subjectDescribe("Delete contact"), Run: contactsAction("delete", false)})
	m.commands.Register(commands.Command{ID: "contacts-edit-note", Module: commands.ModuleContacts, Title: "Edit selected contact note", Description: "Open the highlighted contact note in an editor", Shortcut: "e", Keywords: []string{"edit", "note", "contact"}, Available: onContactItem, Describe: subjectDescribe("Edit note for"), Run: contactsAction("edit-note", false)})
	m.commands.Register(commands.Command{ID: "contacts-refresh-note", Module: commands.ModuleContacts, Title: "Refresh selected contact note", Description: "Fetch the latest note for the highlighted contact", Shortcut: "N", Keywords: []string{"note", "refresh", "contact"}, Available: onContactItem, Describe: subjectDescribe("Refresh note for"), Run: contactsAction("refresh-note", false)})
	m.commands.Register(commands.Command{ID: "contacts-communications", Module: commands.ModuleContacts, Title: "Load selected contact communications", Description: "Fetch communication history for the highlighted contact", Shortcut: "c", Keywords: []string{"communications", "history", "contact"}, Available: onContactItem, Describe: subjectDescribe("Load communications for"), Run: contactsAction("communications", false)})

	// Mail / Drafts
	m.commands.Register(commands.Command{ID: "drafts-compose", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Compose draft", Description: "Start a new draft", Shortcut: "c", Keywords: []string{"compose", "new", "write", "draft"}, Available: onMail, Run: mailAction("compose", true)})
	m.commands.Register(commands.Command{ID: "drafts-send", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Send draft", Shortcut: "S", Keywords: []string{"send", "deliver", "draft"}, Available: onMailDrafts, Describe: subjectDescribe("Send draft"), Run: mailAction("send-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-edit", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Edit draft", Description: "Open in $EDITOR", Shortcut: "e", Keywords: []string{"edit", "write", "draft"}, Available: onMailDrafts, Run: mailAction("edit-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-discard", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Discard draft", Shortcut: "x", Keywords: []string{"delete", "discard", "remove", "draft"}, Available: onMailDrafts, Run: mailAction("delete-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-attach", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Attach file to draft", Shortcut: "a", Keywords: []string{"attach", "file", "upload", "draft"}, Available: onMailDrafts, Run: mailAction("attach", false)})

	// Mail / Messages
	m.commands.Register(commands.Command{ID: "messages-reply", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Reply", Shortcut: "r", Keywords: []string{"reply", "respond"}, Available: onMailMessages, Describe: subjectDescribe("Reply"), Run: mailAction("reply", false)})
	m.commands.Register(commands.Command{ID: "messages-forward", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Forward", Shortcut: "f", Keywords: []string{"forward"}, Available: onMailMessages, Describe: subjectDescribe("Forward"), Run: mailAction("forward", false)})
	m.commands.Register(commands.Command{ID: "messages-archive", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Archive", Shortcut: "a", Keywords: []string{"archive"}, Available: onMailMessages, Describe: subjectDescribe("Archive"), Run: mailAction("archive", false)})
	m.commands.Register(commands.Command{ID: "messages-junk", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Mark as junk", Shortcut: "J", Keywords: []string{"junk", "spam"}, Available: func(ctx commands.Context) bool { return onMailMessages(ctx) && ctx.Selection.Mailbox == "inbox" }, Describe: subjectDescribe("Mark as junk"), Run: mailAction("junk", false)})
	m.commands.Register(commands.Command{ID: "messages-not-junk", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Mark as not junk", Shortcut: "U", Keywords: []string{"not junk", "spam", "inbox"}, Available: func(ctx commands.Context) bool { return onMailMessages(ctx) && ctx.Selection.Mailbox == "junk" }, Describe: subjectDescribe("Mark as not junk"), Run: mailAction("not-junk", false)})
	m.commands.Register(commands.Command{ID: "messages-trash", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Move to trash", Shortcut: "d", Keywords: []string{"trash", "delete"}, Available: onMailMessages, Describe: subjectDescribe("Trash"), Run: mailAction("trash", false)})
	m.commands.Register(commands.Command{ID: "messages-star", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Toggle star", Shortcut: "s", Keywords: []string{"star", "favorite"}, Available: onMailMessages, Run: mailAction("toggle-star", false)})
	m.commands.Register(commands.Command{ID: "messages-read", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Toggle read", Shortcut: "u", Keywords: []string{"read", "unread"}, Available: onMailMessages, Run: mailAction("toggle-read", false)})
	m.commands.Register(commands.Command{ID: "messages-restore", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Restore", Description: "Move back to inbox", Shortcut: "R", Keywords: []string{"restore"}, Available: func(ctx commands.Context) bool {
		return onMail(ctx) && ctx.Selection != nil && (ctx.Selection.Mailbox == "archive" || ctx.Selection.Mailbox == "trash")
	}, Run: mailAction("restore", false)})
	m.commands.Register(commands.Command{ID: "messages-block-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Block sender", Description: "Block future mail from this sender", Keywords: []string{"block", "sender", "spam"}, Available: onMailMessages, Run: mailAction("block-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-unblock-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Unblock sender", Description: "Remove sender block", Keywords: []string{"unblock", "sender"}, Available: onMailMessages, Run: mailAction("unblock-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-trust-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Trust sender", Description: "Trust future mail from this sender", Keywords: []string{"trust", "sender"}, Available: onMailMessages, Run: mailAction("trust-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-untrust-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Untrust sender", Description: "Remove trusted sender policy", Keywords: []string{"untrust", "sender"}, Available: onMailMessages, Run: mailAction("untrust-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-block-domain", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Block sender domain", Description: "Block future mail from this domain", Keywords: []string{"block", "domain", "spam"}, Available: onMailMessages, Run: mailAction("block-domain", false)})
	m.commands.Register(commands.Command{ID: "messages-unblock-domain", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Unblock sender domain", Description: "Remove domain block", Keywords: []string{"unblock", "domain"}, Available: onMailMessages, Run: mailAction("unblock-domain", false)})

	// Global
	m.commands.Register(commands.Command{ID: "toggle-sidebar", Module: commands.ModuleGlobal, Title: "Toggle sidebar", Keywords: []string{"sidebar", "layout"}, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "themes", Module: commands.ModuleGlobal, Title: "Themes…", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "miami"}, OpensPage: "themes"})
	m.commands.Register(commands.Command{ID: "quit", Module: commands.ModuleGlobal, Title: "Quit", Keywords: []string{"exit", "close"}, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
}

func (m Model) paletteContext() commands.Context {
	ctx := commands.Context{ActiveScreen: m.activeScreen, ActiveModule: m.activeModule()}
	if isMailScreen(m.activeScreen) {
		if mail, ok := m.screens[m.activeScreen].(screens.Mail); ok {
			sel := mail.Selection()
			ctx.Selection = &commands.Selection{
				Kind:     sel.BoxLikes,
				Subject:  sel.Subject,
				Mailbox:  sel.Box,
				IsDraft:  sel.IsDraft,
				HasItems: sel.HasItem,
			}
		}
	}
	if m.activeScreen == "notes" {
		if notesScreen, ok := m.screens["notes"].(screens.Notes); ok {
			sel := notesScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem}
		}
	}
	if m.activeScreen == "contacts" {
		if contactsScreen, ok := m.screens["contacts"].(screens.Contacts); ok {
			sel := contactsScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem}
		}
	}
	if m.activeScreen == "calendar" {
		if calendarScreen, ok := m.screens["calendar"].(screens.Calendar); ok {
			sel := calendarScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem, HasInvitation: sel.HasInvitation}
		}
	}
	return ctx
}

func (m Model) activeModule() string {
	switch {
	case isMailScreen(m.activeScreen) || m.activeScreen == "mail-admin":
		return commands.ModuleMail
	case m.activeScreen == "calendar":
		return commands.ModuleCalendar
	case m.activeScreen == "contacts":
		return commands.ModuleContacts
	case m.activeScreen == "drive":
		return commands.ModuleDrive
	case m.activeScreen == "notes":
		return commands.ModuleNotes
	case isHackerNewsScreen(m.activeScreen) || m.activeScreen == "news":
		return commands.ModuleHackerNews
	case m.activeScreen == "settings":
		return commands.ModuleSettings
	}
	return ""
}

func (m *Model) switchScreen(id string) {
	if _, ok := m.screens[id]; !ok {
		m.logs.Warn(fmt.Sprintf("Unknown screen requested: %s", id))
		return
	}
	if m.activeScreen != id {
		m.activeScreen = id
		m.logs.Info(fmt.Sprintf("Screen changed to %s", id))
	}
}

func loadInstance(configPath string) string {
	configFile, _ := config.Paths(configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil || cfg == nil || cfg.BaseURL == "" {
		return ""
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Host == "" {
		return cfg.BaseURL
	}
	return parsed.Host
}

func (m Model) CurrentScreenID() string { return m.activeScreen }

func (m Model) SwitchScreenForTest(id string) Model {
	m.switchScreen(id)
	return m
}
