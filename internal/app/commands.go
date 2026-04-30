package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/commands"
	"github.com/elpdev/telex-cli/internal/screens"
)

func (m *Model) registerCommands() {
	route := func(id string) func() tea.Cmd {
		return func() tea.Cmd { return func() tea.Msg { return routeMsg{id} } }
	}
	mailAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.MailActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"mail"} }, actionMsg)
		}
	}
	driveAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.DriveActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"drive"} }, actionMsg)
		}
	}
	calendarAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.CalendarActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"calendar"} }, actionMsg)
		}
	}
	mailAdminAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.MailAdminActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"mail-admin"} }, actionMsg)
		}
	}
	notesAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.NotesActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"notes"} }, actionMsg)
		}
	}
	tasksAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.TasksActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"tasks"} }, actionMsg)
		}
	}
	contactsAction := func(action string, alsoRoute bool) func() tea.Cmd {
		return func() tea.Cmd {
			actionMsg := func() tea.Msg { return screens.ContactsActionMsg{Action: action} }
			if !alsoRoute {
				return actionMsg
			}
			return tea.Sequence(func() tea.Msg { return routeMsg{"contacts"} }, actionMsg)
		}
	}
	onMail := func(ctx commands.Context) bool { return isMailScreen(ctx.ActiveScreen) }
	onMailAdmin := func(ctx commands.Context) bool { return ctx.ActiveScreen == "mail-admin" }
	onMailOrAdmin := func(ctx commands.Context) bool {
		return isMailScreen(ctx.ActiveScreen) || ctx.ActiveScreen == "mail-admin"
	}
	onCalendarAgenda := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar-event"
	}
	onCalendarItem := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar-event" && ctx.Selection.HasItems
	}
	onCalendarInvitation := func(ctx commands.Context) bool {
		return onCalendarItem(ctx) && ctx.Selection.HasInvitation
	}
	onCalendarManagement := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar"
	}
	onCalendarCalendar := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "calendar" && ctx.Selection != nil && ctx.Selection.Kind == "calendar" && ctx.Selection.HasItems
	}
	onDrive := func(ctx commands.Context) bool { return ctx.ActiveScreen == "drive" }
	onNotes := func(ctx commands.Context) bool { return ctx.ActiveScreen == "notes" }
	onTasks := func(ctx commands.Context) bool { return ctx.ActiveScreen == "tasks" }
	onContacts := func(ctx commands.Context) bool { return ctx.ActiveScreen == "contacts" }
	onContactItem := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "contacts" && ctx.Selection != nil && ctx.Selection.Kind == "contact" && ctx.Selection.HasItems
	}
	onNotesItem := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "notes" && ctx.Selection != nil && ctx.Selection.Kind == "note" && ctx.Selection.HasItems
	}
	onTaskCard := func(ctx commands.Context) bool {
		return ctx.ActiveScreen == "tasks" && ctx.Selection != nil && ctx.Selection.Kind == "task-card" && ctx.Selection.HasItems
	}
	onMailDrafts := func(ctx commands.Context) bool {
		return isMailScreen(ctx.ActiveScreen) && ctx.Selection != nil && ctx.Selection.IsDraft && ctx.Selection.HasItems
	}
	onMailMessages := func(ctx commands.Context) bool {
		return isMailScreen(ctx.ActiveScreen) && ctx.Selection != nil && ctx.Selection.Kind == "message" && ctx.Selection.HasItems
	}
	subjectDescribe := func(prefix string) func(commands.Context) string {
		return func(ctx commands.Context) string {
			if ctx.Selection != nil && ctx.Selection.Subject != "" {
				return prefix + " · " + ctx.Selection.Subject
			}
			return prefix
		}
	}

	m.commands.Register(commands.Command{ID: "go-home", Module: commands.ModuleGlobal, Title: "Go to Home", Keywords: []string{"home", "start"}, Run: route("home")})
	m.commands.Register(commands.Command{ID: "go-mail", Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open Mail", Description: "Unread across all mailboxes", Keywords: []string{"mail", "email", "unread", "inbox"}, Pinned: true, Run: route("mail-unread")})
	m.commands.Register(commands.Command{ID: "go-mailboxes", Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open Mailboxes", Description: "Browse one mailbox at a time", Keywords: []string{"mail", "email", "mailboxes", "accounts"}, Pinned: true, Run: route("mail-mailboxes")})
	for _, scope := range aggregateMailScreens() {
		m.commands.Register(commands.Command{ID: "go-" + scope.id, Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open " + scope.title, Keywords: []string{"mail", "email", strings.ToLower(scope.title)}, Run: route(scope.id)})
	}
	m.commands.Register(commands.Command{ID: "go-mail-admin", Module: commands.ModuleMail, Group: commands.GroupNav, Title: "Open Mail Admin", Description: "Manage domains and inboxes", Keywords: []string{"mail", "admin", "domains", "inboxes"}, Run: route("mail-admin")})
	m.commands.Register(commands.Command{ID: "go-calendar", Module: commands.ModuleCalendar, Title: "Open Calendar", Keywords: []string{"calendar", "events", "agenda"}, Pinned: true, Run: route("calendar")})
	m.commands.Register(commands.Command{ID: "go-contacts", Module: commands.ModuleContacts, Title: "Open Contacts", Keywords: []string{"contacts", "crm", "people"}, Pinned: true, Run: route("contacts")})
	m.commands.Register(commands.Command{ID: "go-notes", Module: commands.ModuleNotes, Title: "Open Notes", Keywords: []string{"notes", "markdown", "memo"}, Pinned: true, Run: route("notes")})
	m.commands.Register(commands.Command{ID: "go-tasks", Module: commands.ModuleTasks, Title: "Open Tasks", Keywords: []string{"tasks", "kanban", "cards", "projects"}, Pinned: true, Run: route("tasks")})
	m.commands.Register(commands.Command{ID: "go-drive", Module: commands.ModuleDrive, Title: "Open Drive", Description: "Local Drive mirror", Keywords: []string{"drive", "files", "documents"}, Pinned: true, Run: route("drive")})
	m.commands.Register(commands.Command{ID: "go-settings", Module: commands.ModuleSettings, Title: "Open Settings", Keywords: []string{"settings", "config"}, Pinned: true, Run: route("settings")})
	m.registerHackerNewsCommands()
	if m.devBuild() {
		m.commands.Register(commands.Command{ID: "go-logs", Module: commands.ModuleGlobal, Title: "Open Logs", Description: "Debug event log", Keywords: []string{"logs", "debug", "events"}, Run: route("logs")})
	}

	m.commands.Register(commands.Command{ID: "mail-sync", Module: commands.ModuleMail, Title: "Sync mailbox", Description: "Pull latest messages, drafts, outbox", Keywords: []string{"sync", "refresh"}, Available: onMailOrAdmin, Run: mailAction("sync", true)})
	m.commands.Register(commands.Command{ID: "mail-admin-refresh", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "Refresh Mail Admin", Description: "Reload remote domains and inboxes", Keywords: []string{"refresh", "reload", "domains", "inboxes"}, Available: onMailAdmin, Run: mailAdminAction("refresh", true)})
	m.commands.Register(commands.Command{ID: "domains-new", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "New domain", Description: "Create a managed mail domain", Keywords: []string{"domain", "new", "create"}, Available: onMailAdmin, Run: mailAdminAction("new-domain", true)})
	m.commands.Register(commands.Command{ID: "domains-validate", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "Validate selected domain", Description: "Check outbound settings", Keywords: []string{"domain", "validate", "smtp", "outbound"}, Available: onMailAdmin, Run: mailAdminAction("validate-domain", true)})
	m.commands.Register(commands.Command{ID: "inboxes-new", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "New inbox", Description: "Create an inbox on the selected domain", Keywords: []string{"inbox", "new", "create"}, Available: onMailAdmin, Run: mailAdminAction("new-inbox", true)})
	m.commands.Register(commands.Command{ID: "inboxes-pipeline", Module: commands.ModuleMail, Group: commands.GroupAdmin, Title: "Show inbox pipeline", Description: "Pipeline metadata for selected inbox", Keywords: []string{"inbox", "pipeline"}, Available: onMailAdmin, Run: mailAdminAction("pipeline", true)})

	m.commands.Register(commands.Command{ID: "calendar-sync", Module: commands.ModuleCalendar, Title: "Sync Calendar", Description: "Pull latest calendars, events, and occurrences", Shortcut: "S", Keywords: []string{"sync", "refresh", "agenda"}, Run: calendarAction("sync", true)})
	m.commands.Register(commands.Command{ID: "calendar-view-agenda", Module: commands.ModuleCalendar, Title: "View agenda", Description: "Show cached calendar occurrences", Keywords: []string{"agenda", "occurrences", "events"}, Available: onCalendarManagement, Run: calendarAction("view-agenda", false)})
	m.commands.Register(commands.Command{ID: "calendar-view-calendars", Module: commands.ModuleCalendar, Title: "View calendars", Description: "Show cached calendars", Shortcut: "v", Keywords: []string{"calendars", "list", "manage"}, Available: onCalendarAgenda, Run: calendarAction("view-calendars", false)})
	m.commands.Register(commands.Command{ID: "calendar-new", Module: commands.ModuleCalendar, Title: "New event", Description: "Create a calendar event", Shortcut: "n", Keywords: []string{"new", "create", "event"}, Available: onCalendarAgenda, Run: calendarAction("new", false)})
	m.commands.Register(commands.Command{ID: "calendar-edit", Module: commands.ModuleCalendar, Title: "Edit selected event", Description: "Edit the highlighted calendar event", Shortcut: "e", Keywords: []string{"edit", "update", "event"}, Available: onCalendarItem, Describe: subjectDescribe("Edit the highlighted calendar event"), Run: calendarAction("edit", false)})
	m.commands.Register(commands.Command{ID: "calendar-today", Module: commands.ModuleCalendar, Title: "Jump to today", Description: "Move selection to the next occurrence today or later", Shortcut: "t", Keywords: []string{"today", "agenda"}, Available: onCalendarAgenda, Run: calendarAction("today", false)})
	m.commands.Register(commands.Command{ID: "calendar-previous-range", Module: commands.ModuleCalendar, Title: "Previous agenda range", Description: "Move the agenda to the previous cached date range", Shortcut: "[", Keywords: []string{"previous", "prev", "range", "agenda", "calendar"}, Available: onCalendarAgenda, Run: calendarAction("previous-range", false)})
	m.commands.Register(commands.Command{ID: "calendar-next-range", Module: commands.ModuleCalendar, Title: "Next agenda range", Description: "Move the agenda to the next cached date range", Shortcut: "]", Keywords: []string{"next", "range", "agenda", "calendar"}, Available: onCalendarAgenda, Run: calendarAction("next-range", false)})
	m.commands.Register(commands.Command{ID: "calendar-filter-agenda", Module: commands.ModuleCalendar, Title: "Filter agenda", Description: "Filter agenda by calendar, status, source, title, or location", Shortcut: "/", Keywords: []string{"filter", "search", "agenda", "calendar", "status", "source"}, Available: onCalendarAgenda, Run: calendarAction("filter", false)})
	m.commands.Register(commands.Command{ID: "calendar-clear-agenda-filters", Module: commands.ModuleCalendar, Title: "Clear agenda filters", Description: "Show all cached agenda occurrences", Shortcut: "ctrl+l", Keywords: []string{"clear", "filter", "search", "agenda"}, Available: onCalendarAgenda, Run: calendarAction("clear-filter", false)})
	m.commands.Register(commands.Command{ID: "calendar-delete", Module: commands.ModuleCalendar, Title: "Delete selected event", Description: "Delete the highlighted calendar event after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "event"}, Available: onCalendarItem, Describe: subjectDescribe("Delete the highlighted calendar event"), Run: calendarAction("delete", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-show", Module: commands.ModuleCalendar, Title: "Show invitation details", Description: "Load invitation details for the linked message", Keywords: []string{"invitation", "invite", "details", "message"}, Available: onCalendarInvitation, Describe: subjectDescribe("Load invitation details for"), Run: calendarAction("invitation-show", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-sync", Module: commands.ModuleCalendar, Title: "Sync selected invitation", Description: "Sync the linked invitation message into Calendar", Keywords: []string{"invitation", "invite", "sync", "message"}, Available: onCalendarInvitation, Describe: subjectDescribe("Sync invitation for"), Run: calendarAction("invitation-sync", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-accept", Module: commands.ModuleCalendar, Title: "Accept invitation", Description: "Respond accepted to the linked invitation", Keywords: []string{"invitation", "invite", "accept", "accepted", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Accept invitation for"), Run: calendarAction("invitation-accepted", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-tentative", Module: commands.ModuleCalendar, Title: "Tentatively accept invitation", Description: "Respond tentative to the linked invitation", Keywords: []string{"invitation", "invite", "tentative", "maybe", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Tentatively accept invitation for"), Run: calendarAction("invitation-tentative", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-decline", Module: commands.ModuleCalendar, Title: "Decline invitation", Description: "Respond declined to the linked invitation", Keywords: []string{"invitation", "invite", "decline", "declined", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Decline invitation for"), Run: calendarAction("invitation-declined", false)})
	m.commands.Register(commands.Command{ID: "calendar-invitation-needs-action", Module: commands.ModuleCalendar, Title: "Mark invitation needs action", Description: "Respond needs_action to the linked invitation", Keywords: []string{"invitation", "invite", "needs_action", "needs action", "rsvp"}, Available: onCalendarInvitation, Describe: subjectDescribe("Mark invitation needs action for"), Run: calendarAction("invitation-needs-action", false)})
	m.commands.Register(commands.Command{ID: "calendars-new", Module: commands.ModuleCalendar, Title: "New calendar", Description: "Create a calendar", Shortcut: "n", Keywords: []string{"new", "create", "calendar"}, Available: onCalendarManagement, Run: calendarAction("new-calendar", false)})
	m.commands.Register(commands.Command{ID: "calendars-edit", Module: commands.ModuleCalendar, Title: "Edit selected calendar", Description: "Edit the highlighted calendar", Shortcut: "e", Keywords: []string{"edit", "update", "calendar"}, Available: onCalendarCalendar, Describe: subjectDescribe("Edit the highlighted calendar"), Run: calendarAction("edit-calendar", false)})
	m.commands.Register(commands.Command{ID: "calendars-import-ics", Module: commands.ModuleCalendar, Title: "Import ICS into selected calendar", Description: "Pick an .ics file and import it into the highlighted calendar", Shortcut: "i", Keywords: []string{"import", "ics", "calendar"}, Available: onCalendarCalendar, Describe: subjectDescribe("Import ICS into calendar"), Run: calendarAction("import-ics", false)})
	m.commands.Register(commands.Command{ID: "calendars-delete", Module: commands.ModuleCalendar, Title: "Delete selected calendar", Description: "Delete the highlighted calendar after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "calendar"}, Available: onCalendarCalendar, Describe: subjectDescribe("Delete the highlighted calendar"), Run: calendarAction("delete-calendar", false)})

	m.commands.Register(commands.Command{ID: "drive-sync", Module: commands.ModuleDrive, Title: "Sync Drive", Description: "Pull latest Drive metadata and files", Shortcut: "S", Keywords: []string{"sync", "refresh"}, Run: driveAction("sync", true)})
	m.commands.Register(commands.Command{ID: "drive-upload", Module: commands.ModuleDrive, Title: "Upload file", Description: "Upload a local file into the current Drive folder", Shortcut: "u", Keywords: []string{"upload", "file"}, Available: onDrive, Run: driveAction("upload", false)})
	m.commands.Register(commands.Command{ID: "drive-new-folder", Module: commands.ModuleDrive, Title: "New folder", Description: "Create a folder in the current Drive folder", Shortcut: "n", Keywords: []string{"new", "folder", "create"}, Available: onDrive, Run: driveAction("new-folder", false)})
	m.commands.Register(commands.Command{ID: "drive-rename", Module: commands.ModuleDrive, Title: "Rename selected", Description: "Rename the highlighted Drive item", Shortcut: "R", Keywords: []string{"rename"}, Available: onDrive, Run: driveAction("rename", false)})
	m.commands.Register(commands.Command{ID: "drive-delete", Module: commands.ModuleDrive, Title: "Delete selected", Description: "Delete the highlighted Drive item after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove"}, Available: onDrive, Run: driveAction("delete", false)})
	m.commands.Register(commands.Command{ID: "drive-details", Module: commands.ModuleDrive, Title: "Show details", Description: "Toggle details for the highlighted Drive item", Shortcut: "i", Keywords: []string{"details", "info"}, Available: onDrive, Run: driveAction("details", false)})

	m.commands.Register(commands.Command{ID: "notes-sync", Module: commands.ModuleNotes, Title: "Sync Notes", Description: "Pull latest Notes folders and note bodies", Shortcut: "S", Keywords: []string{"sync", "refresh"}, Run: notesAction("sync", true)})
	m.commands.Register(commands.Command{ID: "notes-new", Module: commands.ModuleNotes, Title: "New note", Description: "Create a note in the current Notes folder", Shortcut: "n", Keywords: []string{"new", "create", "write"}, Available: onNotes, Run: notesAction("new", false)})
	m.commands.Register(commands.Command{ID: "notes-edit", Module: commands.ModuleNotes, Title: "Edit selected note", Description: "Open the highlighted note in TELEX_NOTES_EDITOR, VISUAL, or EDITOR", Shortcut: "e", Keywords: []string{"edit", "write"}, Available: onNotesItem, Run: notesAction("edit", false)})
	m.commands.Register(commands.Command{ID: "notes-delete", Module: commands.ModuleNotes, Title: "Delete selected note", Description: "Delete the highlighted note after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove"}, Available: onNotesItem, Run: notesAction("delete", false)})
	m.commands.Register(commands.Command{ID: "notes-search", Module: commands.ModuleNotes, Title: "Search current Notes folder", Description: "Filter notes and folders in the current Notes folder", Shortcut: "/", Keywords: []string{"search", "filter"}, Available: onNotes, Run: notesAction("search", false)})
	m.commands.Register(commands.Command{ID: "notes-toggle-sort", Module: commands.ModuleNotes, Title: "Toggle Notes sort order", Description: "Cycle Notes sort between A-Z and most recently updated", Shortcut: "o", Keywords: []string{"sort", "order", "recent"}, Available: onNotes, Run: notesAction("toggle-sort", false)})
	m.commands.Register(commands.Command{ID: "notes-toggle-flat", Module: commands.ModuleNotes, Title: "Toggle Notes flat view", Description: "Show all notes flat across folders, or revert to folder navigation", Shortcut: "f", Keywords: []string{"flat", "all", "view"}, Available: onNotes, Run: notesAction("toggle-flat", false)})

	m.commands.Register(commands.Command{ID: "tasks-sync", Module: commands.ModuleTasks, Title: "Sync Tasks", Description: "Pull latest task projects, boards, and cards", Shortcut: "S", Keywords: []string{"sync", "refresh", "kanban"}, Run: tasksAction("sync", true)})
	m.commands.Register(commands.Command{ID: "tasks-projects", Module: commands.ModuleTasks, Title: "Show task projects", Description: "Return to the cached task project list", Shortcut: "p", Keywords: []string{"projects", "list"}, Available: onTasks, Run: tasksAction("projects", false)})
	m.commands.Register(commands.Command{ID: "tasks-new-project", Module: commands.ModuleTasks, Title: "New task project", Description: "Create a task project", Keywords: []string{"new", "create", "project"}, Available: onTasks, Run: tasksAction("new-project", false)})
	m.commands.Register(commands.Command{ID: "tasks-new-card", Module: commands.ModuleTasks, Title: "New task card", Description: "Create a card in the current task project", Shortcut: "n", Keywords: []string{"new", "create", "card"}, Available: onTasks, Run: tasksAction("new-card", false)})
	m.commands.Register(commands.Command{ID: "tasks-edit-card", Module: commands.ModuleTasks, Title: "Edit selected task card", Description: "Open the highlighted task card in TELEX_TASKS_EDITOR, VISUAL, or EDITOR", Shortcut: "e", Keywords: []string{"edit", "write", "card"}, Available: onTaskCard, Run: tasksAction("edit-card", false)})
	m.commands.Register(commands.Command{ID: "tasks-delete-card", Module: commands.ModuleTasks, Title: "Delete selected task card", Description: "Delete the highlighted task card after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "card"}, Available: onTaskCard, Run: tasksAction("delete-card", false)})
	m.commands.Register(commands.Command{ID: "tasks-move-card-next", Module: commands.ModuleTasks, Title: "Move card to next column", Description: "Move the selected card one column to the right", Shortcut: ">", Keywords: []string{"move", "next", "column", "kanban"}, Available: onTaskCard, Run: tasksAction("move-card-next", false)})
	m.commands.Register(commands.Command{ID: "tasks-move-card-prev", Module: commands.ModuleTasks, Title: "Move card to previous column", Description: "Move the selected card one column to the left", Shortcut: "<", Keywords: []string{"move", "previous", "column", "kanban"}, Available: onTaskCard, Run: tasksAction("move-card-prev", false)})
	m.commands.Register(commands.Command{ID: "tasks-move-card-to", Module: commands.ModuleTasks, Title: "Move card to column…", Description: "Pick a column to move the selected card to", Shortcut: "m", Keywords: []string{"move", "column", "kanban", "todo", "doing", "done"}, Available: onTaskCard, Run: tasksAction("move-card-to", false)})
	m.commands.Register(commands.Command{ID: "tasks-search", Module: commands.ModuleTasks, Title: "Search Tasks", Description: "Filter cached task projects and cards", Shortcut: "/", Keywords: []string{"search", "filter"}, Available: onTasks, Run: tasksAction("search", false)})

	m.commands.Register(commands.Command{ID: "contacts-sync", Module: commands.ModuleContacts, Title: "Sync Contacts", Description: "Pull latest Contacts and notes", Shortcut: "S", Keywords: []string{"sync", "refresh", "crm"}, Run: contactsAction("sync", true)})
	m.commands.Register(commands.Command{ID: "contacts-search", Module: commands.ModuleContacts, Title: "Search Contacts", Description: "Filter cached contacts", Shortcut: "/", Keywords: []string{"search", "filter", "contacts"}, Available: onContacts, Run: contactsAction("search", false)})
	m.commands.Register(commands.Command{ID: "contacts-delete", Module: commands.ModuleContacts, Title: "Delete selected contact", Description: "Delete the highlighted contact after confirmation", Shortcut: "x", Keywords: []string{"delete", "remove", "contact"}, Available: onContactItem, Describe: subjectDescribe("Delete contact"), Run: contactsAction("delete", false)})
	m.commands.Register(commands.Command{ID: "contacts-edit-note", Module: commands.ModuleContacts, Title: "Edit selected contact", Description: "Open the highlighted contact document in an editor", Shortcut: "e", Keywords: []string{"edit", "note", "contact"}, Available: onContactItem, Describe: subjectDescribe("Edit contact"), Run: contactsAction("edit-note", false)})
	m.commands.Register(commands.Command{ID: "contacts-refresh-note", Module: commands.ModuleContacts, Title: "Refresh selected contact note", Description: "Fetch the latest note for the highlighted contact", Shortcut: "N", Keywords: []string{"note", "refresh", "contact"}, Available: onContactItem, Describe: subjectDescribe("Refresh note for"), Run: contactsAction("refresh-note", false)})
	m.commands.Register(commands.Command{ID: "contacts-communications", Module: commands.ModuleContacts, Title: "Load selected contact communications", Description: "Fetch communication history for the highlighted contact", Shortcut: "c", Keywords: []string{"communications", "history", "contact"}, Available: onContactItem, Describe: subjectDescribe("Load communications for"), Run: contactsAction("communications", false)})

	m.commands.Register(commands.Command{ID: "drafts-compose", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Compose draft", Description: "Start a new draft", Shortcut: "c", Keywords: []string{"compose", "new", "write", "draft"}, Available: onMail, Run: mailAction("compose", true)})
	m.commands.Register(commands.Command{ID: "drafts-send", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Send draft", Shortcut: "S", Keywords: []string{"send", "deliver", "draft"}, Available: onMailDrafts, Describe: subjectDescribe("Send draft"), Run: mailAction("send-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-edit", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Edit draft", Description: "Open in $EDITOR", Shortcut: "e", Keywords: []string{"edit", "write", "draft"}, Available: onMailDrafts, Run: mailAction("edit-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-discard", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Discard draft", Shortcut: "x", Keywords: []string{"delete", "discard", "remove", "draft"}, Available: onMailDrafts, Run: mailAction("delete-draft", false)})
	m.commands.Register(commands.Command{ID: "drafts-attach", Module: commands.ModuleMail, Group: commands.GroupDrafts, Title: "Attach file to draft", Shortcut: "a", Keywords: []string{"attach", "file", "upload", "draft"}, Available: onMailDrafts, Run: mailAction("attach", false)})

	m.commands.Register(commands.Command{ID: "messages-reply", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Reply", Shortcut: "r", Keywords: []string{"reply", "respond"}, Available: onMailMessages, Describe: subjectDescribe("Reply"), Run: mailAction("reply", false)})
	m.commands.Register(commands.Command{ID: "messages-forward", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Forward", Shortcut: "f", Keywords: []string{"forward"}, Available: onMailMessages, Describe: subjectDescribe("Forward"), Run: mailAction("forward", false)})
	m.commands.Register(commands.Command{ID: "messages-archive", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Archive", Shortcut: "a", Keywords: []string{"archive"}, Available: onMailMessages, Describe: subjectDescribe("Archive"), Run: mailAction("archive", false)})
	m.commands.Register(commands.Command{ID: "messages-junk", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Mark as junk", Shortcut: "J", Keywords: []string{"junk", "spam"}, Available: func(ctx commands.Context) bool { return onMailMessages(ctx) && ctx.Selection.Mailbox == "inbox" }, Describe: subjectDescribe("Mark as junk"), Run: mailAction("junk", false)})
	m.commands.Register(commands.Command{ID: "messages-not-junk", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Mark as not junk", Shortcut: "U", Keywords: []string{"not junk", "spam", "inbox"}, Available: func(ctx commands.Context) bool { return onMailMessages(ctx) && ctx.Selection.Mailbox == "junk" }, Describe: subjectDescribe("Mark as not junk"), Run: mailAction("not-junk", false)})
	m.commands.Register(commands.Command{ID: "messages-trash", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Move to trash", Shortcut: "d", Keywords: []string{"trash", "delete"}, Available: onMailMessages, Describe: subjectDescribe("Trash"), Run: mailAction("trash", false)})
	m.commands.Register(commands.Command{ID: "messages-star", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Toggle star", Shortcut: "s", Keywords: []string{"star", "favorite"}, Available: onMailMessages, Run: mailAction("toggle-star", false)})
	m.commands.Register(commands.Command{ID: "messages-read", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Toggle read", Shortcut: "u", Keywords: []string{"read", "unread"}, Available: onMailMessages, Run: mailAction("toggle-read", false)})
	m.commands.Register(commands.Command{ID: "messages-restore", Module: commands.ModuleMail, Group: commands.GroupMessages, Title: "Restore", Description: "Move back to inbox", Shortcut: "R", Keywords: []string{"restore"}, Available: func(ctx commands.Context) bool {
		return onMail(ctx) && ctx.Selection != nil && (ctx.Selection.Mailbox == "archive" || ctx.Selection.Mailbox == "trash")
	}, Run: mailAction("restore", false)})
	m.commands.Register(commands.Command{ID: "messages-block-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Block sender", Description: "Block future mail from this sender", Keywords: []string{"block", "sender", "spam"}, Available: onMailMessages, Run: mailAction("block-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-unblock-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Unblock sender", Description: "Remove sender block", Keywords: []string{"unblock", "sender"}, Available: onMailMessages, Run: mailAction("unblock-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-trust-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Trust sender", Description: "Trust future mail from this sender", Keywords: []string{"trust", "sender"}, Available: onMailMessages, Run: mailAction("trust-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-untrust-sender", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Untrust sender", Description: "Remove trusted sender policy", Keywords: []string{"untrust", "sender"}, Available: onMailMessages, Run: mailAction("untrust-sender", false)})
	m.commands.Register(commands.Command{ID: "messages-block-domain", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Block sender domain", Description: "Block future mail from this domain", Keywords: []string{"block", "domain", "spam"}, Available: onMailMessages, Run: mailAction("block-domain", false)})
	m.commands.Register(commands.Command{ID: "messages-unblock-domain", Module: commands.ModuleMail, Group: commands.GroupPolicy, Title: "Unblock sender domain", Description: "Remove domain block", Keywords: []string{"unblock", "domain"}, Available: onMailMessages, Run: mailAction("unblock-domain", false)})

	m.commands.Register(commands.Command{ID: "toggle-sidebar", Module: commands.ModuleGlobal, Title: "Toggle sidebar", Keywords: []string{"sidebar", "layout"}, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "themes", Module: commands.ModuleGlobal, Title: "Themes…", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "miami"}, OpensPage: "themes"})
	m.commands.Register(commands.Command{ID: "quit", Module: commands.ModuleGlobal, Title: "Quit", Keywords: []string{"exit", "close"}, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
}

func (m Model) paletteContext() commands.Context {
	ctx := commands.Context{ActiveScreen: m.activeScreen, ActiveModule: m.activeModule()}
	if isMailScreen(m.activeScreen) {
		if mail, ok := m.activeMailScreen(); ok {
			sel := mail.Selection()
			ctx.Selection = &commands.Selection{
				Kind:     sel.BoxLikes,
				Subject:  sel.Subject,
				Mailbox:  sel.Box,
				IsDraft:  sel.IsDraft,
				HasItems: sel.HasItem,
			}
		}
	}
	if m.activeScreen == "notes" {
		if notesScreen, ok := m.screens["notes"].(screens.Notes); ok {
			sel := notesScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem}
		}
	}
	if m.activeScreen == "tasks" {
		if tasksScreen, ok := m.screens["tasks"].(screens.Tasks); ok {
			sel := tasksScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem}
		}
	}
	if m.activeScreen == "contacts" {
		if contactsScreen, ok := m.screens["contacts"].(screens.Contacts); ok {
			sel := contactsScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem}
		}
	}
	if m.activeScreen == "calendar" {
		if calendarScreen, ok := m.screens["calendar"].(screens.Calendar); ok {
			sel := calendarScreen.Selection()
			ctx.Selection = &commands.Selection{Kind: sel.Kind, Subject: sel.Subject, HasItems: sel.HasItem, HasInvitation: sel.HasInvitation}
		}
	}
	return ctx
}

func (m Model) activeMailScreen() (screens.Mail, bool) {
	if m.activeScreen == "mail" {
		hub, ok := m.screens["mail"].(screens.MailHub)
		if !ok {
			return screens.Mail{}, false
		}
		child, ok := m.screens[hub.ActiveID()].(screens.Mail)
		return child, ok
	}
	mail, ok := m.screens[m.activeScreen].(screens.Mail)
	return mail, ok
}

func (m Model) activeModule() string {
	switch {
	case isMailScreen(m.activeScreen) || m.activeScreen == "mail-admin":
		return commands.ModuleMail
	case m.activeScreen == "calendar":
		return commands.ModuleCalendar
	case m.activeScreen == "contacts":
		return commands.ModuleContacts
	case m.activeScreen == "drive":
		return commands.ModuleDrive
	case m.activeScreen == "notes":
		return commands.ModuleNotes
	case m.activeScreen == "tasks":
		return commands.ModuleTasks
	case isHackerNewsScreen(m.activeScreen) || m.activeScreen == "news":
		return commands.ModuleHackerNews
	case m.activeScreen == "settings":
		return commands.ModuleSettings
	}
	return ""
}
