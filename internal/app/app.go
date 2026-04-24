package app

import (
	"fmt"
	"sort"

	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/debug"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/telex-cli/internal/theme"
	tea "charm.land/bubbletea/v2"
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
}

func New(meta BuildInfo) Model {
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

func (m *Model) registerScreens() {
	m.screens["home"] = screens.NewHome()
	m.screens["settings"] = screens.NewSettings(screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	})
	m.screens["help"] = screens.NewHelp(m.keys.FullHelp())
	m.screens["logs"] = screens.NewLogs(m.logs)
	m.refreshScreenOrder()
}

func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"home", "settings", "help", "logs"}
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
	m.commands.Register(commands.Command{ID: "go-home", Title: "Go to Home", Description: "Open the home screen", Keywords: []string{"home", "start"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"home"} } }})
	m.commands.Register(commands.Command{ID: "go-settings", Title: "Go to Settings", Description: "Open application settings", Keywords: []string{"settings", "config"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"settings"} } }})
	m.commands.Register(commands.Command{ID: "go-help", Title: "Go to Help", Description: "Open keyboard and command documentation", Keywords: []string{"help", "keys", "docs"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"help"} } }})
	m.commands.Register(commands.Command{ID: "go-logs", Title: "Go to Logs", Description: "Open debug event log", Keywords: []string{"logs", "debug", "events"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"logs"} } }})
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

func (m Model) CurrentScreenID() string { return m.activeScreen }

func (m Model) SwitchScreenForTest(id string) Model {
	m.switchScreen(id)
	return m
}
