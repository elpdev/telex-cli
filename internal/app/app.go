package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/debug"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/drivesync"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailsend"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/mailsync"
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

	showSidebar        bool
	showHelp           bool
	showCommandPalette bool

	focus FocusArea
	keys  KeyMap

	commands       *commands.Registry
	commandPalette commands.PaletteModel

	theme theme.Theme
	logs  *debug.Log
	meta  BuildInfo

	configPath string
	dataPath   string
	instance   string
	client     *api.Client
}

func New(meta BuildInfo) Model {
	return NewWithDataPath(meta, "")
}

func NewWithDataPath(meta BuildInfo, dataPath string) Model {
	return NewWithPaths(meta, "", dataPath)
}

func NewWithPaths(meta BuildInfo, configPath, dataPath string) Model {
	log := debug.NewLog()
	log.Info("App started")

	m := Model{
		activeScreen: defaultScreen,
		screens:      make(map[string]screens.Screen),
		showSidebar:  true,
		focus:        FocusMain,
		keys:         DefaultKeyMap(),
		commands:     commands.NewRegistry(),
		theme:        theme.Phosphor(),
		logs:         log,
		meta:         meta,
		configPath:   configPath,
		dataPath:     dataPath,
		instance:     loadInstance(configPath),
	}

	m.registerScreens()
	m.registerCommands()
	m.commandPalette = commands.NewPaletteModel(m.commands, theme.BuiltIns())
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{func() tea.Msg { return tea.RequestWindowSize() }}
	for _, screen := range m.screens {
		cmds = append(cmds, screen.Init())
	}
	return tea.Batch(cmds...)
}

func (m Model) devBuild() bool { return m.meta.Version == "dev" }

func (m *Model) registerScreens() {
	m.screens["home"] = screens.NewHome()
	m.screens["mail"] = screens.NewMailWithActions(mailstore.New(m.dataPath), m.toggleMessageRead, m.toggleMessageStar, m.archiveMessage, m.trashMessage, m.restoreMessage, m.syncMail, m.sendDraft, m.updateDraft, m.deleteDraft, m.forwardMessage, m.downloadAttachment, m.searchMail).WithConversationActions(m.conversationTimeline, m.conversationBody)
	m.screens["drive"] = screens.NewDrive(drivestore.New(m.dataPath), m.syncDrive).WithActions(m.downloadDriveFile, m.openDriveFile, m.uploadDriveFile, m.createDriveFolder, m.renameDriveFile, m.renameDriveFolder, m.deleteDriveFile, m.deleteDriveFolder)
	m.screens["settings"] = screens.NewSettings(screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	})
	if m.devBuild() {
		m.screens["logs"] = screens.NewLogs(m.logs)
	}
	m.refreshScreenOrder()
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

func (m *Model) restoreMessage(ctx context.Context, id int64) error {
	service, err := m.mailService()
	if err != nil {
		return err
	}
	_, err = service.RestoreMessage(ctx, id)
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
	if opener == "" {
		opener = "xdg-open"
	}
	return exec.Command(opener, path).Run()
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

func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"home", "mail", "drive", "settings", "logs"}
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
	onMail := func(ctx commands.Context) bool { return ctx.ActiveScreen == "mail" }
	onDrive := func(ctx commands.Context) bool { return ctx.ActiveScreen == "drive" }
	onMailDrafts := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "mail" && ctx.Selection != nil && ctx.Selection.IsDraft && ctx.Selection.HasItems
	}
	onMailMessages := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "mail" && ctx.Selection != nil && ctx.Selection.Kind == "message" && ctx.Selection.HasItems
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
	m.commands.Register(commands.Command{ID: "go-home", Module: commands.ModuleGlobal, Title: "Go to Home", Description: "Open the home screen", Keywords: []string{"home", "start"}, Run: route("home")})
	m.commands.Register(commands.Command{ID: "go-mail", Module: commands.ModuleMail, Title: "Open Mail", Description: "Switch to cached mail", Keywords: []string{"mail", "email", "inbox"}, Run: route("mail")})
	m.commands.Register(commands.Command{ID: "go-drive", Module: commands.ModuleDrive, Title: "Open Drive", Description: "Switch to local Drive mirror", Keywords: []string{"drive", "files", "documents"}, Run: route("drive")})
	m.commands.Register(commands.Command{ID: "go-settings", Module: commands.ModuleSettings, Title: "Open Settings", Description: "Open application settings", Keywords: []string{"settings", "config"}, Run: route("settings")})
	if m.devBuild() {
		m.commands.Register(commands.Command{ID: "go-logs", Module: commands.ModuleGlobal, Title: "Open Logs", Description: "Open debug event log", Keywords: []string{"logs", "debug", "events"}, Run: route("logs")})
	}

	// Mail — module-level
	m.commands.Register(commands.Command{ID: "mail-sync", Module: commands.ModuleMail, Title: "Sync mailbox", Description: "Pull latest messages, drafts, outbox", Keywords: []string{"sync", "refresh"}, Run: mailAction("sync", true)})

	// Drive — module-level
	m.commands.Register(commands.Command{ID: "drive-sync", Module: commands.ModuleDrive, Title: "Sync Drive", Description: "Pull latest Drive metadata and files", Shortcut: "S", Keywords: []string{"sync", "refresh"}, Run: driveAction("sync", true)})
	m.commands.Register(commands.Command{ID: "drive-upload", Module: commands.ModuleDrive, Title: "Upload file", Description: "Upload a local file into the current Drive folder", Shortcut: "u", Keywords: []string{"upload", "file"}, Available: onDrive, Run: driveAction("upload", false)})
	m.commands.Register(commands.Command{ID: "drive-new-folder", Module: commands.ModuleDrive, Title: "New folder", Description: "Create a folder in the current Drive folder", Shortcut: "n", Keywords: []string{"new", "folder", "create"}, Available: onDrive, Run: driveAction("new-folder", false)})
	m.commands.Register(commands.Command{ID: "drive-rename", Module: commands.ModuleDrive, Title: "Rename selected", Description: "Rename the highlighted Drive item", Shortcut: "R", Keywords: []string{"rename"}, Available: onDrive, Run: driveAction("rename", false)})
	m.commands.Register(commands.Command{ID: "drive-delete", Module: commands.ModuleDrive, Title: "Delete selected", Description: "Delete the highlighted Drive item after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove"}, Available: onDrive, Run: driveAction("delete", false)})
	m.commands.Register(commands.Command{ID: "drive-details", Module: commands.ModuleDrive, Title: "Show details", Description: "Toggle details for the highlighted Drive item", Shortcut: "i", Keywords: []string{"details", "info"}, Available: onDrive, Run: driveAction("details", false)})

	// Mail / Drafts
	m.commands.Register(commands.Command{ID: "drafts-compose", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Compose new", Description: "Start a new draft", Shortcut: "c", Keywords: []string{"compose", "new", "write"}, Run: mailAction("compose", true)})
	m.commands.Register(commands.Command{ID: "drafts-actions", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Actions on selected draft…", Description: "Open focused list of draft actions", Available: onMailDrafts, OpensPage: "draft-actions"})
	m.commands.Register(commands.Command{ID: "drafts-send", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Send", Description: "Send the highlighted draft", Shortcut: "S", Keywords: []string{"send", "deliver"}, Available: onMailDrafts, Describe: subjectDescribe("Send the highlighted draft"), Run: mailAction("send-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-edit", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Edit", Description: "Open the draft in $EDITOR", Shortcut: "e", Keywords: []string{"edit", "write"}, Available: onMailDrafts, Run: mailAction("edit-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-discard", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Discard", Description: "Delete the highlighted draft", Shortcut: "x", Keywords: []string{"delete", "discard", "remove"}, Available: onMailDrafts, Run: mailAction("delete-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-attach", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Attach file…", Description: "Attach a file to the draft", Shortcut: "a", Keywords: []string{"attach", "file", "upload"}, Available: onMailDrafts, Run: mailAction("attach", false)})

	// Mail / Messages
	m.commands.Register(commands.Command{ID: "messages-actions", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Actions on selected message…", Description: "Open focused list of message actions", Available: onMailMessages, OpensPage: "message-actions"})
	m.commands.Register(commands.Command{ID: "messages-reply", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Reply", Description: "Reply to the highlighted message", Shortcut: "r", Keywords: []string{"reply", "respond"}, Available: onMailMessages, Describe: subjectDescribe("Reply"), Run: mailAction("reply", false)})
	m.commands.Register(commands.Command{ID: "messages-forward", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Forward", Description: "Forward the highlighted message", Shortcut: "f", Keywords: []string{"forward"}, Available: onMailMessages, Describe: subjectDescribe("Forward"), Run: mailAction("forward", false)})
	m.commands.Register(commands.Command{ID: "messages-archive", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Archive", Description: "Archive the highlighted message", Shortcut: "a", Keywords: []string{"archive"}, Available: onMailMessages, Describe: subjectDescribe("Archive"), Run: mailAction("archive", false)})
	m.commands.Register(commands.Command{ID: "messages-trash", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Move to trash", Description: "Trash the highlighted message", Shortcut: "d", Keywords: []string{"trash", "delete"}, Available: onMailMessages, Describe: subjectDescribe("Trash"), Run: mailAction("trash", false)})
	m.commands.Register(commands.Command{ID: "messages-star", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Toggle star", Description: "Star/unstar the highlighted message", Shortcut: "s", Keywords: []string{"star", "favorite"}, Available: onMailMessages, Run: mailAction("toggle-star", false)})
	m.commands.Register(commands.Command{ID: "messages-read", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Toggle read", Description: "Mark read/unread", Shortcut: "u", Keywords: []string{"read", "unread"}, Available: onMailMessages, Run: mailAction("toggle-read", false)})
	m.commands.Register(commands.Command{ID: "messages-restore", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Restore", Description: "Move back to inbox from archive/trash", Shortcut: "R", Keywords: []string{"restore"}, Available: func(ctx commands.Context) bool {
		return onMail(ctx) && ctx.Selection != nil && (ctx.Selection.Mailbox == "archive" || ctx.Selection.Mailbox == "trash")
	}, Run: mailAction("restore", false)})

	// Global
	m.commands.Register(commands.Command{ID: "toggle-sidebar", Module: commands.ModuleGlobal, Title: "Toggle sidebar", Description: "Show or hide sidebar navigation", Keywords: []string{"sidebar", "layout"}, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "themes", Module: commands.ModuleGlobal, Title: "Themes…", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "miami"}, OpensPage: "themes"})
	m.commands.Register(commands.Command{ID: "quit", Module: commands.ModuleGlobal, Title: "Quit", Description: "Exit Telex", Keywords: []string{"exit", "close"}, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
}

func (m Model) paletteContext() commands.Context {
	ctx := commands.Context{ActiveScreen: m.activeScreen}
	if m.activeScreen == "mail" {
		if mail, ok := m.screens["mail"].(screens.Mail); ok {
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
	return ctx
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
