package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
	"github.com/elpdev/tuimod"
)

type NewsTab struct {
	ID    string
	Label string
}

type NewsKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
}

func DefaultNewsKeyMap() NewsKeyMap {
	return NewsKeyMap{
		NextTab: key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next tab")),
		PrevTab: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev tab")),
	}
}

type News struct {
	tabs      []NewsTab
	activeIdx int
	theme     theme.Theme
	screens   map[string]Screen
	initFn    func(id string) tea.Cmd
	keys      NewsKeyMap
}

func NewNews(tabs []NewsTab, t theme.Theme, screens map[string]Screen, initFn func(string) tea.Cmd) News {
	return News{
		tabs:    tabs,
		theme:   t,
		screens: screens,
		initFn:  initFn,
		keys:    DefaultNewsKeyMap(),
	}
}

func (n News) Reconfigure(t theme.Theme) News {
	n.theme = t
	return n
}

func (n News) ActiveID() string {
	if len(n.tabs) == 0 {
		return ""
	}
	return n.tabs[n.activeIdx].ID
}

func (n News) ActiveIndex() int { return n.activeIdx }

func (n News) SetActiveIndex(idx int) News {
	if idx < 0 || idx >= len(n.tabs) {
		return n
	}
	n.activeIdx = idx
	return n
}

func (n News) SetActiveID(id string) (News, tea.Cmd) {
	for i, tab := range n.tabs {
		if tab.ID == id {
			if i == n.activeIdx {
				return n, nil
			}
			n.activeIdx = i
			return n, n.initFn(id)
		}
	}
	return n, nil
}

func (n News) Title() string { return "News" }

func (n News) Init() tea.Cmd {
	if n.initFn == nil {
		return nil
	}
	return n.initFn(n.ActiveID())
}

func (n News) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, n.keys.NextTab):
			n.activeIdx = (n.activeIdx + 1) % len(n.tabs)
			return n, n.initFn(n.ActiveID())
		case key.Matches(keyMsg, n.keys.PrevTab):
			n.activeIdx = (n.activeIdx - 1 + len(n.tabs)) % len(n.tabs)
			return n, n.initFn(n.ActiveID())
		}
	}
	id := n.ActiveID()
	child, ok := n.screens[id]
	if !ok {
		return n, nil
	}
	updated, cmd := child.Update(msg)
	n.screens[id] = updated
	return n, cmd
}

func (n News) View(width, height int) string {
	bar := n.renderTabBar(width)
	barH := lipgloss.Height(bar)
	bodyH := height - barH
	if bodyH < 0 {
		bodyH = 0
	}
	var body string
	if child, ok := n.screens[n.ActiveID()]; ok {
		body = child.View(width, bodyH)
	}
	return lipgloss.JoinVertical(lipgloss.Left, bar, body)
}

func (n News) renderTabBar(width int) string {
	parts := make([]string, 0, len(n.tabs))
	for i, tab := range n.tabs {
		if i == n.activeIdx {
			parts = append(parts, n.theme.HeaderAccent.Render("["+tab.Label+"]"))
		} else {
			parts = append(parts, n.theme.Muted.Render(" "+tab.Label+" "))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(parts, " "))
}

func (n News) KeyBindings() []key.Binding {
	bindings := []key.Binding{n.keys.PrevTab, n.keys.NextTab}
	if child, ok := n.screens[n.ActiveID()]; ok {
		bindings = append(bindings, child.KeyBindings()...)
	}
	return bindings
}

func (n News) CapturesKey(msg tea.KeyPressMsg) bool {
	if key.Matches(msg, n.keys.NextTab) || key.Matches(msg, n.keys.PrevTab) {
		return true
	}
	if child, ok := n.screens[n.ActiveID()]; ok {
		if cap, ok := child.(tuimod.KeyCapturer); ok {
			return cap.CapturesKey(msg)
		}
	}
	return false
}
