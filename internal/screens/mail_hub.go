package screens

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
	"github.com/elpdev/tuimod"
)

type MailHubTab struct {
	ID    string
	Label string
}

type MailHubKeyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
}

func DefaultMailHubKeyMap() MailHubKeyMap {
	return MailHubKeyMap{
		NextTab: key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next tab")),
		PrevTab: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev tab")),
	}
}

type MailHub struct {
	tabs      []MailHubTab
	activeIdx int
	theme     theme.Theme
	screens   map[string]Screen
	initFn    func(id string) tea.Cmd
	keys      MailHubKeyMap
}

func NewMailHub(tabs []MailHubTab, t theme.Theme, screens map[string]Screen, initFn func(string) tea.Cmd) MailHub {
	return MailHub{
		tabs:    tabs,
		theme:   t,
		screens: screens,
		initFn:  initFn,
		keys:    DefaultMailHubKeyMap(),
	}
}

func (h MailHub) Reconfigure(t theme.Theme) MailHub {
	h.theme = t
	return h
}

func (h MailHub) ActiveID() string {
	if len(h.tabs) == 0 {
		return ""
	}
	return h.tabs[h.activeIdx].ID
}

func (h MailHub) ActiveIndex() int { return h.activeIdx }

func (h MailHub) SetActiveIndex(idx int) MailHub {
	if idx < 0 || idx >= len(h.tabs) {
		return h
	}
	h.activeIdx = idx
	return h
}

func (h MailHub) SetActiveID(id string) (MailHub, tea.Cmd) {
	for i, tab := range h.tabs {
		if tab.ID == id {
			if i == h.activeIdx {
				return h, nil
			}
			h.activeIdx = i
			return h, h.initFn(id)
		}
	}
	return h, nil
}

func (h MailHub) Title() string { return "Mail" }

func (h MailHub) Init() tea.Cmd {
	if h.initFn == nil {
		return nil
	}
	return h.initFn(h.ActiveID())
}

func (h MailHub) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, h.keys.NextTab):
			h.activeIdx = (h.activeIdx + 1) % len(h.tabs)
			return h, h.initFn(h.ActiveID())
		case key.Matches(keyMsg, h.keys.PrevTab):
			h.activeIdx = (h.activeIdx - 1 + len(h.tabs)) % len(h.tabs)
			return h, h.initFn(h.ActiveID())
		}
	}
	id := h.ActiveID()
	child, ok := h.screens[id]
	if !ok {
		return h, nil
	}
	updated, cmd := child.Update(msg)
	h.screens[id] = updated
	return h, cmd
}

func (h MailHub) View(width, height int) string {
	bar := h.renderTabBar(width)
	barH := lipgloss.Height(bar)
	bodyH := height - barH
	if bodyH < 0 {
		bodyH = 0
	}
	var body string
	if child, ok := h.screens[h.ActiveID()]; ok {
		body = child.View(width, bodyH)
	}
	return lipgloss.JoinVertical(lipgloss.Left, bar, body)
}

func (h MailHub) renderTabBar(width int) string {
	parts := make([]string, 0, len(h.tabs))
	for i, tab := range h.tabs {
		if i == h.activeIdx {
			parts = append(parts, h.theme.HeaderAccent.Render("["+tab.Label+"]"))
		} else {
			parts = append(parts, h.theme.Muted.Render(" "+tab.Label+" "))
		}
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(parts, " "))
}

func (h MailHub) KeyBindings() []key.Binding {
	bindings := []key.Binding{h.keys.PrevTab, h.keys.NextTab}
	if child, ok := h.screens[h.ActiveID()]; ok {
		bindings = append(bindings, child.KeyBindings()...)
	}
	return bindings
}

func (h MailHub) CapturesKey(msg tea.KeyPressMsg) bool {
	if key.Matches(msg, h.keys.NextTab) || key.Matches(msg, h.keys.PrevTab) {
		return true
	}
	if child, ok := h.screens[h.ActiveID()]; ok {
		if cap, ok := child.(tuimod.KeyCapturer); ok {
			return cap.CapturesKey(msg)
		}
	}
	return false
}

func (h MailHub) CapturesFocusKey(msg tea.KeyPressMsg) bool {
	if child, ok := h.screens[h.ActiveID()]; ok {
		if cap, ok := child.(interface {
			CapturesFocusKey(tea.KeyPressMsg) bool
		}); ok {
			return cap.CapturesFocusKey(msg)
		}
	}
	return false
}
