package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	legacykey "github.com/charmbracelet/bubbles/key"
	hackernews "github.com/elpdev/hackernews"
	hnconfig "github.com/elpdev/hackernews/pkg/config"
	"github.com/elpdev/hackernews/pkg/history"
	"github.com/elpdev/hackernews/pkg/saved"
	hnscreens "github.com/elpdev/hackernews/pkg/screens"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/mailstore"
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
		if id == "hn-saved" {
			return screen.Init()
		}
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
		m.screens[hackerNewsScreenID(spec.ID)] = wrapHackerNewsScreen(spec.Screen)
	}
}

func (m Model) newHackerNewsModule() hackernews.Module {
	settings := m.loadHackerNewsSettings()

	savedStore := saved.NewJSONStore(m.hackerNewsSavedPath())
	historyStore := history.NewJSONStore(m.hackerNewsHistoryPath())

	return hackernews.New(hackernews.Options{SavedStore: savedStore, HistoryStore: historyStore, Settings: settings})
}

func (m Model) loadHackerNewsSettings() hnconfig.Settings {
	settings := m.defaultHackerNewsSettings()
	path := m.hackerNewsConfigPath()
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			m.logs.Warn(fmt.Sprintf("Hacker News config unavailable: %v", err))
		}
		return settings
	}
	loaded, err := hnconfig.NewStore(path).Load()
	if err != nil {
		m.logs.Warn(fmt.Sprintf("Could not load Hacker News config: %v", err))
		return settings
	}
	settings = loaded
	if strings.TrimSpace(settings.SyncDir) == "" || settings.SyncDir == hnconfig.Defaults().SyncDir {
		settings.SyncDir = m.hackerNewsSyncDir()
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
		Pinned:      true,
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
	searchScreen, ok := unwrapHackerNewsScreen(m.screens["hn-search"])
	if !ok {
		return m
	}
	search, ok := searchScreen.(hnscreens.Search)
	if !ok {
		return m
	}
	screens := make(map[string]tuimod.Screen)
	for id, screen := range m.screens {
		if isHackerNewsScreen(id) {
			if unwrapped, ok := unwrapHackerNewsScreen(screen); ok {
				screens[id[len(hackerNewsPrefix):]] = unwrapped
			}
		}
	}
	m.screens["hn-search"] = wrapHackerNewsScreen(hackernews.RefreshSearchScreen(screens, search))
	return m
}

func (m Model) openHackerNewsDoctor() (Model, tea.Cmd) {
	doctorScreen, ok := unwrapHackerNewsScreen(m.screens["hn-doctor"])
	if !ok {
		m.logs.Warn("Hacker News doctor screen unavailable")
		return m, nil
	}
	doctor, ok := doctorScreen.(hnscreens.Doctor)
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
	m.screens["hn-doctor"] = wrapHackerNewsScreen(updated)
	m.switchScreen("hn-doctor")
	return m, cmd
}

func (m Model) applyHackerNewsSettings(settings hnconfig.Settings) Model {
	m.saveHackerNewsSettings(settings)
	for id, screen := range m.screens {
		unwrapped, ok := unwrapHackerNewsScreen(screen)
		if !ok {
			continue
		}
		if stories, ok := unwrapped.(hnscreens.Top); ok {
			stories.SetHideRead(settings.HideRead)
			stories.SetSortMode(settings.SortMode)
			m.screens[id] = wrapHackerNewsScreen(stories)
		}
		if settingsScreen, ok := unwrapped.(hnscreens.Settings); ok {
			m.screens[id] = wrapHackerNewsScreen(settingsScreen.WithSettings(settings))
		}
		if doctorScreen, ok := unwrapped.(hnscreens.Doctor); ok {
			m.screens[id] = wrapHackerNewsScreen(doctorScreen.WithSettings(settings))
		}
	}
	return m
}

type hackerNewsScreen struct{ inner tuimod.Screen }

func wrapHackerNewsScreen(screen tuimod.Screen) screens.Screen {
	return hackerNewsScreen{inner: screen}
}

func unwrapHackerNewsScreen(screen screens.Screen) (tuimod.Screen, bool) {
	wrapped, ok := screen.(hackerNewsScreen)
	if !ok {
		return nil, false
	}
	return wrapped.inner, true
}

func (s hackerNewsScreen) Init() tea.Cmd { return s.inner.Init() }

func (s hackerNewsScreen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	updated, cmd := s.inner.Update(msg)
	return wrapHackerNewsScreen(updated), cmd
}

func (s hackerNewsScreen) View(width, height int) string { return s.inner.View(width, height) }

func (s hackerNewsScreen) Title() string { return s.inner.Title() }

func (s hackerNewsScreen) KeyBindings() []key.Binding {
	return convertKeyBindings(s.inner.KeyBindings())
}

func (s hackerNewsScreen) CapturesKey(msg tea.KeyPressMsg) bool {
	capturer, ok := s.inner.(tuimod.KeyCapturer)
	return ok && capturer.CapturesKey(msg)
}

func convertKeyBindings(bindings []legacykey.Binding) []key.Binding {
	converted := make([]key.Binding, 0, len(bindings))
	for _, binding := range bindings {
		help := binding.Help()
		options := []key.BindingOpt{key.WithKeys(binding.Keys()...), key.WithHelp(help.Key, help.Desc)}
		if !binding.Enabled() {
			options = append(options, key.WithDisabled())
		}
		converted = append(converted, key.NewBinding(options...))
	}
	return converted
}

func (m Model) saveHackerNewsSettings(settings hnconfig.Settings) {
	path := m.hackerNewsConfigPath()
	if strings.TrimSpace(settings.SyncDir) == "" || settings.SyncDir == hnconfig.Defaults().SyncDir {
		settings.SyncDir = m.hackerNewsSyncDir()
	}
	if err := hnconfig.NewStore(path).Save(settings); err != nil {
		m.logs.Warn(fmt.Sprintf("Could not save Hacker News config: %v", err))
	}
}

func (m Model) defaultHackerNewsSettings() hnconfig.Settings {
	settings := hnconfig.Defaults()
	settings.SyncDir = m.hackerNewsSyncDir()
	return settings
}

func (m Model) hackerNewsDataRoot() string {
	return filepath.Join(mailstore.RootOrDefault(m.dataPath), "hackernews")
}

func (m Model) hackerNewsSavedPath() string {
	return filepath.Join(m.hackerNewsDataRoot(), "saved.json")
}

func (m Model) hackerNewsHistoryPath() string {
	return filepath.Join(m.hackerNewsDataRoot(), "history.json")
}

func (m Model) hackerNewsDeletedSavedPath() string {
	return filepath.Join(m.hackerNewsDataRoot(), "deleted_saved.json")
}

func (m Model) hackerNewsSyncDir() string {
	return filepath.Join(m.hackerNewsDataRoot(), "sync")
}

func (m Model) hackerNewsConfigPath() string {
	return filepath.Join(filepath.Dir(config.PrefsPathFor(m.configPath)), "hackernews.json")
}
