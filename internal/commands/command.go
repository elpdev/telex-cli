package commands

import tea "charm.land/bubbletea/v2"

const (
	ModuleMail       = "mail"
	ModuleCalendar   = "calendar"
	ModuleContacts   = "contacts"
	ModuleDrive      = "drive"
	ModuleNotes      = "notes"
	ModuleHackerNews = "hackernews"
	ModuleSettings   = "settings"
	ModuleGlobal     = "global"

	GroupDrafts   = "drafts"
	GroupMessages = "messages"
	GroupOutbox   = "outbox"
	GroupInbox    = "inbox"
)

func Modules() []string {
	return []string{ModuleMail, ModuleCalendar, ModuleContacts, ModuleDrive, ModuleNotes, ModuleHackerNews, ModuleSettings, ModuleGlobal}
}

func ScopedModules() []string {
	return []string{ModuleMail, ModuleCalendar, ModuleContacts, ModuleDrive, ModuleNotes, ModuleHackerNews, ModuleSettings}
}

func Groups() []string {
	return []string{GroupDrafts, GroupMessages, GroupOutbox, GroupInbox}
}

type Command struct {
	ID          string
	Module      string
	Group       string
	Title       string
	Description string
	Shortcut    string
	Keywords    []string
	Pinned      bool
	Available   func(Context) bool
	Describe    func(Context) string
	OpensPage   string
	Run         func() tea.Cmd
}

type Context struct {
	ActiveScreen string
	ActiveModule string
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

func (c Command) HasCustomAvailability() bool { return c.Available != nil }

func (c Command) DescriptionFor(ctx Context) string {
	if c.Describe != nil {
		if d := c.Describe(ctx); d != "" {
			return d
		}
	}
	return c.Description
}
