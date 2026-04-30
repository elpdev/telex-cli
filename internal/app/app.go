package app

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	helpbubble "charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/pkg/hn"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/debug"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/telex-cli/internal/taskstore"
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
	syncState  *backgroundSyncState
}

type backgroundSyncState struct {
	mu          sync.Mutex
	mailSyncing bool
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
		syncState:    &backgroundSyncState{},
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
	cmds = append(cmds, m.startupSyncCmd(), m.backgroundMailSyncCmd("boot"), mailAutoSyncTickCmd())
	return tea.Batch(cmds...)
}

func (m Model) devBuild() bool { return m.meta.Version == "dev" }

func (m *Model) registerScreens() {
	m.screens["home"] = m.buildHome()
	m.screens["mail-mailboxes"] = m.buildMailScreen()
	for _, scope := range aggregateMailScreens() {
		m.screens[scope.id] = m.buildAggregateMailScreen(scope)
	}
	m.screens["mail"] = m.buildMailHub()
	m.screens["mail-admin"] = screens.NewMailAdmin(m.loadMailAdmin).WithActions(m.saveDomain, m.deleteDomain, m.validateDomainOutbound, m.saveInbox, m.deleteInbox, m.inboxPipeline)
	m.screens["calendar"] = screens.NewCalendar(calendarstore.New(m.dataPath), m.syncCalendar).WithActions(m.createCalendarEvent, m.updateCalendarEvent, m.deleteCalendarEvent).WithCalendarActions(m.createCalendar, m.updateCalendar, m.deleteCalendar).WithImportICS(m.importCalendarICS).WithInvitationActions(m.showCalendarInvitation, m.syncCalendarInvitation, m.respondCalendarInvitation)
	m.screens["contacts"] = screens.NewContacts(contactstore.New(m.dataPath), m.syncContacts).WithActions(m.updateContact, m.deleteContact, m.loadContactNote, m.updateContactNote, m.loadContactCommunications)
	m.screens["drive"] = screens.NewDrive(drivestore.New(m.dataPath), m.syncDrive).WithActions(m.downloadDriveFile, m.openDriveFile, m.uploadDriveFile, m.createDriveFolder, m.renameDriveFile, m.renameDriveFolder, m.deleteDriveFile, m.deleteDriveFolder)
	m.screens["notes"] = screens.NewNotes(notestore.New(m.dataPath), m.syncNotes).WithActions(m.createNote, m.updateNote, m.deleteNote)
	m.screens["tasks"] = screens.NewTasks(taskstore.New(m.dataPath), m.syncTasks).WithActions(m.createTaskProject, m.createTaskCard, m.updateTaskCard, m.deleteTaskCard, m.moveTaskCard)
	m.screens["settings"] = m.buildSettings()
	if m.devBuild() {
		m.screens["logs"] = screens.NewLogs(m.logs)
	}
	m.registerHackerNewsModule()
	m.screens["news"] = m.buildNews()
	m.refreshScreenOrder()
}

func (m *Model) buildHome() screens.Home {
	if existing, ok := m.screens["home"].(screens.Home); ok {
		return existing.Reconfigure(m.theme)
	}
	navigate := func(id string) tea.Cmd {
		return func() tea.Msg { return routeMsg{ScreenID: id} }
	}
	hnClient := hn.NewClient(nil)
	newsFetcher := func(ctx context.Context, limit int) ([]hn.Item, error) {
		return hnClient.TopStories(ctx, limit)
	}
	return screens.NewHome(
		mailstore.New(m.dataPath),
		calendarstore.New(m.dataPath),
		notestore.New(m.dataPath),
		drivestore.New(m.dataPath),
		taskstore.New(m.dataPath),
		contactstore.New(m.dataPath),
		newsFetcher,
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

func (m *Model) buildMailHub() screens.MailHub {
	tabs := []screens.MailHubTab{
		{ID: "mail-unread", Label: "Unread"},
		{ID: "mail-inbox", Label: "Inbox"},
		{ID: "mail-starred", Label: "Starred"},
		{ID: "mail-drafts", Label: "Drafts"},
		{ID: "mail-sent", Label: "Sent"},
		{ID: "mail-outbox", Label: "Outbox"},
		{ID: "mail-archive", Label: "Archive"},
		{ID: "mail-junk", Label: "Junk"},
		{ID: "mail-trash", Label: "Trash"},
		{ID: "mail-mailboxes", Label: "Mailboxes"},
	}
	initFn := func(id string) tea.Cmd { return m.initScreen(id) }
	hub := screens.NewMailHub(tabs, m.theme, m.screens, initFn)
	if existing, ok := m.screens["mail"].(screens.MailHub); ok {
		hub = hub.SetActiveIndex(existing.ActiveIndex())
	}
	return hub
}

func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		if id == "mail-admin" || id == "mail-mailboxes" || isAggregateMailScreen(id) || isHackerNewsScreen(id) {
			continue
		}
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"home", "mail", "calendar", "contacts", "notes", "tasks", "drive", "news", "settings", "logs"}
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
