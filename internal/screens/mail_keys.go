package screens

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func DefaultMailKeyMap() MailKeyMap {
	return MailKeyMap{
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "message up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "message down")),
		Previous:     key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("left/h", "mailbox prev")),
		Next:         key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("right/l", "mailbox next")),
		BoxPrev:      key.NewBinding(key.WithKeys("{"), key.WithHelp("{", "box prev")),
		BoxNext:      key.NewBinding(key.WithKeys("}"), key.WithHelp("}", "box next")),
		Open:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		OpenHTML:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open html")),
		Links:        key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "links")),
		Extract:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "extract")),
		Compose:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compose")),
		Reply:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
		Forward:      key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "forward")),
		Send:         key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "send draft")),
		Delete:       key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete draft")),
		Attachments:  key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "attachments")),
		ToggleRead:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "read/unread")),
		ToggleStar:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "star/unstar")),
		Archive:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive")),
		Junk:         key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "junk")),
		NotJunk:      key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "not junk")),
		Trash:        key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "trash")),
		Restore:      key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "restore")),
		Copy:         key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy link")),
		Back:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Refresh:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		RemoteSearch: key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "remote search")),
		Thread:       key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "thread")),
	}
}

func (m Mail) KeyBindings() []key.Binding {
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Previous, m.keys.Next, m.keys.BoxPrev, m.keys.BoxNext, m.keys.Open, m.keys.OpenHTML, m.keys.Links, m.keys.Attachments, m.keys.Extract, m.keys.Compose, m.keys.Reply, m.keys.Forward, m.keys.Send, m.keys.Delete, m.keys.ToggleRead, m.keys.ToggleStar, m.keys.Archive, m.keys.Junk, m.keys.NotJunk, m.keys.Trash, m.keys.Restore, m.keys.Copy, m.keys.Back, m.keys.Refresh, m.keys.RemoteSearch, m.keys.Thread}
}

func (m Mail) CapturesFocusKey(msg tea.KeyPressMsg) bool {
	return m.mode == mailModeConversation && msg.String() == "tab"
}
