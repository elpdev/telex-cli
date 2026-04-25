package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/debug"
)

type Logs struct {
	log    *debug.Log
	offset int
}

func NewLogs(log *debug.Log) Logs {
	return Logs{log: log}
}

func (l Logs) Init() tea.Cmd { return nil }

func (l Logs) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "up", "k":
			if l.offset > 0 {
				l.offset--
			}
		case "down", "j":
			if l.offset < max(0, len(l.log.Entries())-1) {
				l.offset++
			}
		}
	}
	return l, nil
}

func (l Logs) View(width, height int) string {
	lines := strings.Split(strings.TrimRight(l.content(), "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	if l.offset > max(0, len(lines)-1) {
		l.offset = max(0, len(lines)-1)
	}
	end := min(len(lines), l.offset+height)
	return strings.Join(lines[l.offset:end], "\n")
}

func (l Logs) Title() string { return "Logs" }

func (l Logs) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "scroll up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "scroll down")),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (l Logs) content() string {
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
