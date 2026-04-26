package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	hackernews "github.com/elpdev/hackernews"
	hnconfig "github.com/elpdev/hackernews/pkg/config"
	"github.com/elpdev/hackernews/pkg/history"
	"github.com/elpdev/hackernews/pkg/saved"
	hnscreens "github.com/elpdev/hackernews/pkg/screens"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/tuimod"
)

const hackerNewsPrefix = "hn-"

func hackerNewsScreenID(id string) string {
	if id == "" {
		return ""
	}
	return hackerNewsPrefix + id
}

func isHackerNewsScreen(id string) bool {
	return len(id) > len(hackerNewsPrefix) && id[:len(hackerNewsPrefix)] == hackerNewsPrefix
}

func (m *Model) initScreen(id string) tea.Cmd {
	screen, ok := m.screens[id]
	if !ok {
		return nil
	}
	if isHackerNewsScreen(id) {
		if m.initialized[id] {
			return nil
		}
		m.initialized[id] = true
	}
	return screen.Init()
}

func (m *Model) registerHackerNewsModule() {
	module := m.newHackerNewsModule()
	for _, spec := range module.ScreenSpecs() {
		m.screens[hackerNewsScreenID(spec.ID)] = spec.Screen
	}
}

func (m Model) newHackerNewsModule() hackernews.Module {
	settings := m.loadHackerNewsSettings()

	var savedStore saved.Store
	if path, err := saved.DefaultPath(); err != nil {
		m.logs.Warn(fmt.Sprintf("Hacker News saved store unavailable: %v", err))
	} else {
		savedStore = saved.NewJSONStore(path)
	}

	var historyStore history.Store
	if path, err := history.DefaultPath(); err != nil {
		m.logs.Warn(fmt.Sprintf("Hacker News read history unavailable: %v", err))
	} else {
		historyStore = history.NewJSONStore(path)
	}

	return hackernews.New(hackernews.Options{SavedStore: savedStore, HistoryStore: historyStore, Settings: settings})
}

func (m Model) loadHackerNewsSettings() hnconfig.Settings {
	settings := hnconfig.Defaults()
	if path, err := hnconfig.DefaultPath(); err != nil {
		m.logs.Warn(fmt.Sprintf("Hacker News config unavailable: %v", err))
	} else if loaded, err := hnconfig.NewStore(path).Load(); err != nil {
		m.logs.Warn(fmt.Sprintf("Could not load Hacker News config: %v", err))
	} else {
		settings = loaded
	}
	return settings
}

func (m *Model) registerHackerNewsCommands() {
	m.commands.Register(commands.Command{
		ID:          "go-news",
		Module:      commands.ModuleHackerNews,
		Title:       "Open News",
		Description: "Open Hacker News feeds",
		Keywords:    []string{"news", "hacker", "hn", "stories"},
		Run:         func() tea.Cmd { return func() tea.Msg { return routeMsg{"news"} } },
	})

	module := m.newHackerNewsModule()
	for _, spec := range module.Commands() {
		if spec.ScreenID == "" && spec.Run == nil {
			continue
		}
		// Per-feed nav specs are handled as tabs inside the News screen, so
		// they'd be redundant in the palette.
		if spec.Run == nil && isNewsTabScreen(hackerNewsScreenID(spec.ScreenID)) {
			continue
		}
		command := commands.Command{
			ID:          hackerNewsScreenID(spec.ID),
			Module:      commands.ModuleHackerNews,
			Title:       spec.Title,
			Description: spec.Description,
			Shortcut:    spec.Shortcut,
			Keywords:    spec.Keywords,
		}
		if spec.Run != nil {
			command.Run = spec.Run
		} else {
			screenID := hackerNewsScreenID(spec.ScreenID)
			command.Run = func() tea.Cmd { return func() tea.Msg { return routeMsg{screenID} } }
		}
		m.commands.Register(command)
	}
}

func (m Model) refreshHackerNewsSearchScreen() Model {
	search, ok := m.screens["hn-search"].(hnscreens.Search)
	if !ok {
		return m
	}
	screens := make(map[string]tuimod.Screen)
	for id, screen := range m.screens {
		if isHackerNewsScreen(id) {
			screens[id[len(hackerNewsPrefix):]] = screen
		}
	}
	m.screens["hn-search"] = hackernews.RefreshSearchScreen(screens, search)
	return m
}

func (m Model) openHackerNewsDoctor() (Model, tea.Cmd) {
	doctor, ok := m.screens["hn-doctor"].(hnscreens.Doctor)
	if !ok {
		m.logs.Warn("Hacker News doctor screen unavailable")
		return m, nil
	}
	returnTo := "top"
	switch {
	case m.activeScreen == "news":
		if news, ok := m.screens["news"].(screens.News); ok {
			if id := news.ActiveID(); isHackerNewsScreen(id) {
				returnTo = id[len(hackerNewsPrefix):]
			}
		}
	case isHackerNewsScreen(m.activeScreen):
		returnTo = m.activeScreen[len(hackerNewsPrefix):]
	}
	updated, cmd := doctor.Open(returnTo, m.loadHackerNewsSettings())
	m.screens["hn-doctor"] = updated
	m.switchScreen("hn-doctor")
	return m, cmd
}

func (m Model) applyHackerNewsSettings(settings hnconfig.Settings) Model {
	m.saveHackerNewsSettings(settings)
	for id, screen := range m.screens {
		if stories, ok := screen.(hnscreens.Top); ok {
			stories.SetHideRead(settings.HideRead)
			stories.SetSortMode(settings.SortMode)
			m.screens[id] = stories
		}
		if settingsScreen, ok := screen.(hnscreens.Settings); ok {
			m.screens[id] = settingsScreen.WithSettings(settings)
		}
		if doctorScreen, ok := screen.(hnscreens.Doctor); ok {
			m.screens[id] = doctorScreen.WithSettings(settings)
		}
	}
	return m
}

func (m Model) saveHackerNewsSettings(settings hnconfig.Settings) {
	path, err := hnconfig.DefaultPath()
	if err != nil {
		m.logs.Warn(fmt.Sprintf("Hacker News config unavailable: %v", err))
		return
	}
	if err := hnconfig.NewStore(path).Save(settings); err != nil {
		m.logs.Warn(fmt.Sprintf("Could not save Hacker News config: %v", err))
	}
}
