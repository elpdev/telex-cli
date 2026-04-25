package app

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/debug"
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
	m.screens["mail"] = screens.NewMailWithActions(mailstore.New(m.dataPath), m.toggleMessageRead, m.toggleMessageStar, m.archiveMessage, m.trashMessage, m.restoreMessage, m.syncMail, m.sendDraft, m.forwardMessage, m.downloadAttachment)
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

func (m *Model) forwardMessage(ctx context.Context, id int64, to []string) (int64, string, error) {
	service, err := m.mailService()
	if err != nil {
		return 0, "", err
	}
	outbound, err := service.Forward(ctx, id, to)
	if err != nil {
		return 0, "", err
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

func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"home", "mail", "settings", "logs"}
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
	m.commands.Register(commands.Command{ID: "go-home", Title: "Home", Description: "Open the home screen", Keywords: []string{"home", "start"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"home"} } }})
	m.commands.Register(commands.Command{ID: "go-mail", Title: "Mail", Description: "Open cached mail", Keywords: []string{"mail", "email", "inbox"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"mail"} } }})
	m.commands.Register(commands.Command{ID: "go-settings", Title: "Settings", Description: "Open application settings", Keywords: []string{"settings", "config"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"settings"} } }})
	if m.devBuild() {
		m.commands.Register(commands.Command{ID: "go-logs", Title: "Logs", Description: "Open debug event log", Keywords: []string{"logs", "debug", "events"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"logs"} } }})
	}
	m.commands.Register(commands.Command{ID: "toggle-sidebar", Title: "Toggle Sidebar", Description: "Show or hide sidebar navigation", Keywords: []string{"sidebar", "layout"}, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "themes", Title: "Themes", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "miami"}})
	m.commands.Register(commands.Command{ID: "quit", Title: "Quit", Description: "Exit Telex", Keywords: []string{"exit", "close"}, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
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
