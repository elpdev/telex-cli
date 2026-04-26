package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/debug"
)

type Logs struct {
	log      *debug.Log
	viewport viewport.Model
}

func NewLogs(log *debug.Log) *Logs {
	return &Logs{log: log, viewport: viewport.New()}
}

func (l *Logs) Init() tea.Cmd { return nil }

func (l *Logs) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "up", "k":
			l.viewport.ScrollUp(1)
		case "down", "j":
			l.viewport.ScrollDown(1)
		}
	}
	return l, nil
}

func (l *Logs) View(width, height int) string {
	l.viewport.SetWidth(width)
	l.viewport.SetHeight(height)
	l.viewport.SetContent(strings.TrimRight(l.content(), "\n"))
	return l.viewport.View()
}

func (l *Logs) Title() string { return "Logs" }

func (l *Logs) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "scroll up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "scroll down")),
	}
}

func (l *Logs) content() string {
	entries := l.log.Entries()
	if len(entries) == 0 {
		return "No log entries yet."
	}
	var b strings.Builder
	for _, entry := range entries {
		b.WriteString(fmt.Sprintf("%s  %-5s  %s\n", entry.Time.Format("15:04:05"), entry.Level, entry.Message))
	}
	return b.String()
}
