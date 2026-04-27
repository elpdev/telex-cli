package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/calendarsync"
	"github.com/spf13/cobra"
)

type calendarSyncResult = calendarsync.Result

func newCalendarCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "calendar", Short: "Calendar commands"}
	cmd.AddCommand(newCalendarSyncCommand(rt))
	cmd.AddCommand(newCalendarCalendarsCommand(rt))
	cmd.AddCommand(newCalendarOccurrencesCommand(rt))
	cmd.AddCommand(newCalendarEventsCommand(rt))
	cmd.AddCommand(newCalendarShowCommand(rt))
	cmd.AddCommand(newCalendarCreateCommand(rt))
	cmd.AddCommand(newCalendarEditCommand(rt))
	cmd.AddCommand(newCalendarDeleteCommand(rt))
	cmd.AddCommand(newCalendarImportICSCommand(rt))
	cmd.AddCommand(newCalendarInvitationCommand(rt))
	cmd.AddCommand(newCalendarInvitationSyncCommand(rt))
	cmd.AddCommand(newCalendarInvitationRespondCommand(rt))
	return cmd
}

func newCalendarSyncCommand(rt *runtime) *cobra.Command {
	var from string
	var to string
	var calendarID int64
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync remote Calendar into the local cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			result, err := runCalendarSync(rt, service, calendarstore.New(rt.dataPath), calendarSyncOptions{From: from, To: to, CalendarID: calendarID})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Synced %d calendar(s), %d event(s), %d occurrence(s).\n", result.Calendars, result.Events, result.Occurrences)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "range start time; default today")
	cmd.Flags().StringVar(&to, "to", "", "range end time; default 30 days from now")
	cmd.Flags().Int64Var(&calendarID, "calendar-id", 0, "limit sync to one calendar ID")
	return cmd
}

func newCalendarCalendarsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendars",
		Short: "List cached calendars",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := calendarstore.New(rt.dataPath).ListCalendars()
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{strconv.FormatInt(item.RemoteID, 10), item.Name, item.Color, item.TimeZone, strconv.Itoa(item.Position)})
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "name", "color", "time_zone", "position"}, rows)
			return nil
		},
	}
	cmd.AddCommand(newCalendarCreateCalendarCommand(rt))
	cmd.AddCommand(newCalendarEditCalendarCommand(rt))
	cmd.AddCommand(newCalendarDeleteCalendarCommand(rt))
	return cmd
}

func newCalendarCreateCalendarCommand(rt *runtime) *cobra.Command {
	var input calendar.CalendarInput
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a remote calendar and cache it locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(input.Name) == "" {
				return fmt.Errorf("--name is required")
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			created, err := service.CreateCalendar(rt.context(), input)
			if err != nil {
				return err
			}
			if err := calendarstore.New(rt.dataPath).StoreCalendar(*created, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, calendarFields(*created))
			return nil
		},
	}
	addCalendarInputFlags(cmd, &input)
	return cmd
}

func newCalendarEditCalendarCommand(rt *runtime) *cobra.Command {
	var input calendar.CalendarInput
	cmd := &cobra.Command{
		Use:   "edit <calendar-id>",
		Short: "Update a remote calendar and cache it locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			updated, err := service.UpdateCalendar(rt.context(), id, input)
			if err != nil {
				return err
			}
			if err := calendarstore.New(rt.dataPath).StoreCalendar(*updated, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, calendarFields(*updated))
			return nil
		},
	}
	addCalendarInputFlags(cmd, &input)
	return cmd
}

func newCalendarDeleteCalendarCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <calendar-id>",
		Short: "Delete a remote calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			if err := service.DeleteCalendar(rt.context(), id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted calendar %d.\n", id)
			return nil
		},
	}
}

func newCalendarOccurrencesCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "occurrences",
		Short: "List cached calendar occurrences",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := calendarstore.New(rt.dataPath).ListOccurrences()
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{strconv.FormatInt(item.EventID, 10), strconv.FormatInt(item.CalendarID, 10), item.Title, formatTime(item.StartsAt), formatTime(item.EndsAt), strconv.FormatBool(item.AllDay), item.Status})
			}
			writeRows(cmd.OutOrStdout(), []string{"event_id", "calendar_id", "title", "starts_at", "ends_at", "all_day", "status"}, rows)
			return nil
		},
	}
}

func newCalendarEventsCommand(rt *runtime) *cobra.Command {
	var calendarID int64
	cmd := &cobra.Command{
		Use:   "events",
		Short: "List cached calendar events",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := calendarstore.New(rt.dataPath).ListEvents(calendarID)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, cachedEventRow(item))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "calendar_id", "title", "starts_at", "ends_at", "status", "path"}, rows)
			return nil
		},
	}
	cmd.Flags().Int64Var(&calendarID, "calendar-id", 0, "cached calendar ID")
	return cmd
}

func newCalendarShowCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "show <event-id>",
		Short: "Show a cached calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			event, err := calendarstore.New(rt.dataPath).ReadEvent(id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, cachedEventFields(*event))
			if strings.TrimSpace(event.Description) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", event.Description)
			}
			return nil
		},
	}
}

func newCalendarCreateCommand(rt *runtime) *cobra.Command {
	var input calendar.CalendarEventInput
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a remote calendar event and cache it locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input.CalendarID == 0 || strings.TrimSpace(input.Title) == "" {
				return fmt.Errorf("--calendar-id and --title are required")
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			event, err := service.CreateEvent(rt.context(), input)
			if err != nil {
				return err
			}
			if err := calendarstore.New(rt.dataPath).StoreEvent(*event, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, eventFields(*event))
			return nil
		},
	}
	addEventInputFlags(cmd, &input)
	return cmd
}

func newCalendarEditCommand(rt *runtime) *cobra.Command {
	var input calendar.CalendarEventInput
	cmd := &cobra.Command{
		Use:   "edit <event-id>",
		Short: "Update a remote calendar event and cache it locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if input.CalendarID == 0 {
				cached, err := calendarstore.New(rt.dataPath).ReadEvent(id)
				if err == nil {
					input.CalendarID = cached.Meta.CalendarID
				}
			}
			if input.CalendarID == 0 {
				return fmt.Errorf("--calendar-id is required when the event is not cached")
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			event, err := service.UpdateEvent(rt.context(), id, input)
			if err != nil {
				return err
			}
			if err := calendarstore.New(rt.dataPath).StoreEvent(*event, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, eventFields(*event))
			return nil
		},
	}
	addEventInputFlags(cmd, &input)
	return cmd
}

func newCalendarDeleteCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <event-id>",
		Short: "Delete a remote calendar event and remove the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			if err := service.DeleteEvent(rt.context(), id); err != nil {
				return err
			}
			if err := calendarstore.New(rt.dataPath).DeleteEvent(id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted calendar event %d.\n", id)
			return nil
		},
	}
}

func newCalendarImportICSCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "import-ics <calendar-id> <file>",
		Short: "Import an ICS file into a calendar",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			result, err := service.ImportICS(rt.context(), id, args[1])
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"created", strconv.Itoa(result.Created)}, {"updated", strconv.Itoa(result.Updated)}, {"skipped", strconv.Itoa(result.Skipped)}, {"failed", strconv.Itoa(result.Failed)}, {"success", strconv.FormatBool(result.Success)}})
			return nil
		},
	}
}

func newCalendarInvitationCommand(rt *runtime) *cobra.Command {
	return calendarInvitationCommand("invitation", "Show a message calendar invitation", func(service *calendar.Service, id int64) (*calendar.Invitation, error) {
		return service.ShowInvitation(rt.context(), id)
	}, rt)
}

func newCalendarInvitationSyncCommand(rt *runtime) *cobra.Command {
	return calendarInvitationCommand("invitation-sync", "Sync a message calendar invitation into Calendar", func(service *calendar.Service, id int64) (*calendar.Invitation, error) {
		return service.SyncInvitation(rt.context(), id)
	}, rt)
}

func newCalendarInvitationRespondCommand(rt *runtime) *cobra.Command {
	var status string
	cmd := calendarInvitationCommand("invitation-respond", "Respond to a message calendar invitation", func(service *calendar.Service, id int64) (*calendar.Invitation, error) {
		if status == "" {
			return nil, fmt.Errorf("--status is required")
		}
		return service.UpdateInvitation(rt.context(), id, calendar.InvitationInput{ParticipationStatus: status})
	}, rt)
	cmd.Flags().StringVar(&status, "status", "", "participation status: accepted, tentative, declined, needs_action")
	return cmd
}

func calendarInvitationCommand(use, short string, run func(*calendar.Service, int64) (*calendar.Invitation, error), rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <message-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := calendarService(rt)
			if err != nil {
				return err
			}
			invite, err := run(service, id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, invitationFields(*invite))
			return nil
		},
	}
}

type calendarSyncOptions = calendarsync.Options

func runCalendarSync(rt *runtime, service *calendar.Service, store calendarstore.Store, opts calendarSyncOptions) (*calendarSyncResult, error) {
	result, err := calendarsync.Run(rt.context(), store, service, opts)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func calendarService(rt *runtime) (*calendar.Service, error) {
	client, err := rt.apiClient()
	if err != nil {
		return nil, err
	}
	return calendar.NewService(client), nil
}

func addEventInputFlags(cmd *cobra.Command, input *calendar.CalendarEventInput) {
	var allDay bool
	cmd.Flags().Int64Var(&input.CalendarID, "calendar-id", 0, "remote calendar ID")
	cmd.Flags().StringVar(&input.Title, "title", "", "event title")
	cmd.Flags().StringVar(&input.Description, "description", "", "event description")
	cmd.Flags().StringVar(&input.Location, "location", "", "event location")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "event lasts all day")
	cmd.Flags().StringVar(&input.StartDate, "start-date", "", "start date, YYYY-MM-DD")
	cmd.Flags().StringVar(&input.EndDate, "end-date", "", "end date, YYYY-MM-DD")
	cmd.Flags().StringVar(&input.StartTime, "start-time", "", "start time, HH:MM")
	cmd.Flags().StringVar(&input.EndTime, "end-time", "", "end time, HH:MM")
	cmd.Flags().StringVar(&input.TimeZone, "time-zone", "", "IANA time zone")
	cmd.Flags().StringVar(&input.Status, "status", "", "event status")
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("all-day") {
			input.AllDay = &allDay
		}
	}
}

func addCalendarInputFlags(cmd *cobra.Command, input *calendar.CalendarInput) {
	var position int
	cmd.Flags().StringVar(&input.Name, "name", "", "calendar name")
	cmd.Flags().StringVar(&input.Color, "color", "", "calendar color")
	cmd.Flags().StringVar(&input.TimeZone, "time-zone", "", "IANA time zone")
	cmd.Flags().IntVar(&position, "position", 0, "calendar sort position")
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("position") {
			input.Position = &position
		}
	}
}

func defaultCalendarRange(from, to string) (string, string) {
	return calendarsync.DefaultRange(from, to)
}

func cachedEventRow(event calendarstore.CachedEvent) []string {
	return []string{strconv.FormatInt(event.Meta.RemoteID, 10), strconv.FormatInt(event.Meta.CalendarID, 10), event.Meta.Title, formatTime(event.Meta.StartsAt), formatTime(event.Meta.EndsAt), event.Meta.Status, event.Path}
}

func cachedEventFields(event calendarstore.CachedEvent) [][]string {
	return [][]string{{"id", strconv.FormatInt(event.Meta.RemoteID, 10)}, {"calendar_id", strconv.FormatInt(event.Meta.CalendarID, 10)}, {"title", event.Meta.Title}, {"location", event.Meta.Location}, {"starts_at", formatTime(event.Meta.StartsAt)}, {"ends_at", formatTime(event.Meta.EndsAt)}, {"all_day", strconv.FormatBool(event.Meta.AllDay)}, {"status", event.Meta.Status}, {"source", event.Meta.Source}, {"uid", event.Meta.UID}, {"path", event.Path}}
}

func eventFields(event calendar.CalendarEvent) [][]string {
	return [][]string{{"id", strconv.FormatInt(event.ID, 10)}, {"calendar_id", strconv.FormatInt(event.CalendarID, 10)}, {"title", event.Title}, {"starts_at", formatTime(event.StartsAt)}, {"ends_at", formatTime(event.EndsAt)}, {"status", event.Status}}
}

func calendarFields(calendar calendar.Calendar) [][]string {
	return [][]string{{"id", strconv.FormatInt(calendar.ID, 10)}, {"name", calendar.Name}, {"color", calendar.Color}, {"time_zone", calendar.TimeZone}, {"position", strconv.Itoa(calendar.Position)}}
}

func invitationFields(invite calendar.Invitation) [][]string {
	rows := [][]string{{"message_id", strconv.FormatInt(invite.MessageID, 10)}, {"available", strconv.FormatBool(invite.Available)}}
	if invite.CalendarEvent != nil {
		rows = append(rows, []string{"event_id", strconv.FormatInt(invite.CalendarEvent.ID, 10)}, []string{"event_title", invite.CalendarEvent.Title})
	}
	if invite.CurrentUserAttendee != nil {
		rows = append(rows, []string{"participation_status", invite.CurrentUserAttendee.ParticipationStatus})
	}
	return rows
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04")
}
