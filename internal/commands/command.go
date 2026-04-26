package commands

import tea "charm.land/bubbletea/v2"

const (
	ModuleMail     = "mail"
	ModuleCalendar = "calendar"
	ModuleDrive    = "drive"
	ModuleNotes    = "notes"
	ModuleSettings = "settings"
	ModuleGlobal   = "global"

	GroupDrafts   = "drafts"
	GroupMessages = "messages"
	GroupOutbox   = "outbox"
	GroupInbox    = "inbox"
)

type Command struct {
	ID          string
	Module      string
	Group       string
	Title       string
	Description string
	Shortcut    string
	Keywords    []string
	Available   func(Context) bool
	Describe    func(Context) string
	OpensPage   string
	Run         func() tea.Cmd
}

type Context struct {
	ActiveScreen string
	Selection    *Selection
}

type Selection struct {
	Kind          string
	Subject       string
	Mailbox       string
	IsDraft       bool
	HasItems      bool
	HasInvitation bool
}

func (c Command) IsAvailable(ctx Context) bool {
	if c.Available == nil {
		return true
	}
	return c.Available(ctx)
}

func (c Command) DescriptionFor(ctx Context) string {
	if c.Describe != nil {
		if d := c.Describe(ctx); d != "" {
			return d
		}
	}
	return c.Description
}
