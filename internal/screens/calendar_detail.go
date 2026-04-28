package screens

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) detailView() string {
	item, ok := c.selected()
	if !ok {
		if c.invitation != nil {
			return strings.Join(invitationView(*c.invitation), "\n") + "\n"
		}
		return "No event selected.\n"
	}
	cal, hasCalendar := c.calendarByID(item.CalendarID)
	event, err := c.store.ReadEvent(item.EventID)
	if err != nil {
		lines := occurrenceDetailLines(item, cal, hasCalendar)
		lines = append(lines, "", "Cached event details: unavailable")
		return strings.Join(lines, "\n") + "\n"
	}
	messageID := firstEventMessageID(event.Meta)
	if !hasCalendar && event.Meta.CalendarID != item.CalendarID {
		cal, hasCalendar = c.calendarByID(event.Meta.CalendarID)
	}
	lines := cachedEventDetailLines(*event, cal, hasCalendar)
	if c.invitation != nil && c.invitation.MessageID == messageID {
		lines = append(lines, "")
		lines = append(lines, invitationView(*c.invitation)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func occurrenceDetailLines(item calendarstore.OccurrenceMeta, cal calendarstore.CalendarMeta, hasCalendar bool) []string {
	return []string{
		item.Title,
		"",
		"Event ID: " + strconv.FormatInt(item.EventID, 10),
		"Calendar ID: " + strconv.FormatInt(item.CalendarID, 10),
		"Calendar: " + calendarDetailName(item.CalendarID, cal, hasCalendar),
		"Calendar color: " + calendarDetailColor(cal, hasCalendar),
		"Calendar time zone: " + calendarDetailTimeZone(cal, hasCalendar, ""),
		"Starts: " + item.StartsAt.Format("2006-01-02 15:04"),
		"Ends: " + item.EndsAt.Format("2006-01-02 15:04"),
		"All day: " + strconv.FormatBool(item.AllDay),
		"Location: " + item.Location,
		"Status: " + item.Status,
	}
}

func cachedEventDetailLines(event calendarstore.CachedEvent, cal calendarstore.CalendarMeta, hasCalendar bool) []string {
	meta := event.Meta
	lines := []string{
		meta.Title,
		"",
		"Event ID: " + strconv.FormatInt(meta.RemoteID, 10),
		"Calendar ID: " + strconv.FormatInt(meta.CalendarID, 10),
		"Calendar: " + calendarDetailName(meta.CalendarID, cal, hasCalendar),
		"Calendar color: " + calendarDetailColor(cal, hasCalendar),
		"Calendar time zone: " + calendarDetailTimeZone(cal, hasCalendar, meta.TimeZone),
		"Starts: " + meta.StartsAt.Format("2006-01-02 15:04"),
		"Ends: " + meta.EndsAt.Format("2006-01-02 15:04"),
		"All day: " + strconv.FormatBool(meta.AllDay),
		"Event time zone: " + emptyDash(meta.TimeZone),
		"Location: " + emptyDash(meta.Location),
		"Status: " + emptyDash(meta.Status),
	}
	lines = append(lines, descriptionView(event.Description)...)
	lines = append(lines, eventOrganizerView(meta)...)
	lines = append(lines, recurrenceView(meta)...)
	lines = append(lines, attendeeListView(meta.Attendees, meta.CurrentUserAttendee)...)
	lines = append(lines, linkListView(meta.Links)...)
	lines = append(lines, messageSummaryView(meta.Messages)...)
	if meta.Invitation {
		lines = append(lines, "", "Invitation: true")
	}
	return lines
}

func descriptionView(description string) []string {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil
	}
	lines := []string{"", "Description:"}
	for _, line := range strings.Split(description, "\n") {
		lines = append(lines, strings.TrimRight(line, " \t"))
	}
	return lines
}

func eventOrganizerView(event calendarstore.EventMeta) []string {
	if event.OrganizerName == "" && event.OrganizerEmail == "" {
		return nil
	}
	return []string{"", "Organizer: " + organizerDisplay(event)}
}

func recurrenceView(event calendarstore.EventMeta) []string {
	if event.RecurrenceSummary == "" && event.RecurrenceRule == "" && len(event.NextOccurrences) == 0 && len(event.RecurrenceExceptions) == 0 {
		return nil
	}
	lines := []string{"", "Recurrence:"}
	if event.RecurrenceSummary != "" {
		lines = append(lines, "Summary: "+event.RecurrenceSummary)
	}
	if event.RecurrenceRule != "" {
		lines = append(lines, "Rule: "+event.RecurrenceRule)
	}
	if len(event.NextOccurrences) > 0 {
		lines = append(lines, fmt.Sprintf("Next occurrences: %d", len(event.NextOccurrences)))
		for _, occurrence := range event.NextOccurrences {
			lines = append(lines, "- "+formatCalendarMessageTime(occurrence))
		}
	}
	if len(event.RecurrenceExceptions) > 0 {
		lines = append(lines, fmt.Sprintf("Exceptions: %d", len(event.RecurrenceExceptions)))
		for _, exception := range event.RecurrenceExceptions {
			lines = append(lines, "- "+emptyDash(exception))
		}
	}
	return lines
}

func attendeeListView(attendees []calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) []string {
	if len(attendees) == 0 {
		lines := []string{"", "Attendees: none"}
		if current != nil {
			lines = append(lines, "Current attendee: "+attendeeSummary(*current, false))
		}
		return lines
	}
	lines := []string{"", fmt.Sprintf("Attendees: %d", len(attendees))}
	if current != nil {
		lines = append(lines, "Current attendee: "+attendeeSummary(*current, false))
	}
	displayed := displayedAttendees(attendees, current)
	for _, attendee := range displayed {
		lines = append(lines, "- "+attendeeSummary(attendee, attendeeMatchesCurrent(attendee, current)))
	}
	if len(attendees) > len(displayed) {
		lines = append(lines, fmt.Sprintf("... %d more attendee(s) not shown", len(attendees)-len(displayed)))
	}
	return lines
}

func displayedAttendees(attendees []calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) []calendarstore.AttendeeMeta {
	limit := min(len(attendees), calendarDetailMaxAttendees)
	displayed := append([]calendarstore.AttendeeMeta(nil), attendees[:limit]...)
	if current == nil || len(attendees) <= limit || attendeeListContains(displayed, current) {
		return displayed
	}
	for _, attendee := range attendees[limit:] {
		if attendeeMatchesCurrent(attendee, current) {
			displayed[len(displayed)-1] = attendee
			return displayed
		}
	}
	return displayed
}

func attendeeListContains(attendees []calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) bool {
	for _, attendee := range attendees {
		if attendeeMatchesCurrent(attendee, current) {
			return true
		}
	}
	return false
}

func attendeeMatchesCurrent(attendee calendarstore.AttendeeMeta, current *calendarstore.AttendeeMeta) bool {
	if current == nil {
		return false
	}
	if current.ID != 0 && attendee.ID == current.ID {
		return true
	}
	return strings.TrimSpace(current.Email) != "" && strings.EqualFold(strings.TrimSpace(attendee.Email), strings.TrimSpace(current.Email))
}

func attendeeSummary(attendee calendarstore.AttendeeMeta, current bool) string {
	display := attendeeDisplay(attendee)
	if current {
		display += " [you]"
	}
	return fmt.Sprintf("%s | role:%s | status:%s | response requested:%t", display, emptyDash(attendee.Role), emptyDash(attendee.ParticipationStatus), attendee.ResponseRequested)
}

func linkListView(links []calendarstore.LinkMeta) []string {
	if len(links) == 0 {
		return []string{"", "Links: none"}
	}
	lines := []string{"", fmt.Sprintf("Links: %d", len(links))}
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("- message:%d | uid:%s | method:%s | sequence:%d", link.MessageID, emptyDash(link.ICalUID), emptyDash(link.ICalMethod), link.SequenceNumber))
	}
	return lines
}

func messageSummaryView(messages []calendarstore.MessageMeta) []string {
	if len(messages) == 0 {
		return []string{"", "Messages: none"}
	}
	lines := []string{"", fmt.Sprintf("Messages: %d", len(messages))}
	for _, message := range messages {
		summary := fmt.Sprintf("- %s | %s | %s | inbox:%d | %s", emptyDash(message.Subject), calendarMessageSender(message), formatCalendarMessageTime(message.ReceivedAt), message.InboxID, emptyDash(message.SystemState))
		if strings.TrimSpace(message.PreviewText) != "" {
			summary += " | " + strings.TrimSpace(message.PreviewText)
		}
		lines = append(lines, summary)
	}
	return lines
}

func invitationView(invite calendar.Invitation) []string {
	lines := []string{"Invitation details:", "Message ID: " + strconv.FormatInt(invite.MessageID, 10), "Available: " + strconv.FormatBool(invite.Available)}
	if invite.CalendarEvent != nil {
		lines = append(lines, "Event: "+invite.CalendarEvent.Title, "Event ID: "+strconv.FormatInt(invite.CalendarEvent.ID, 10))
	}
	if invite.CurrentUserAttendee != nil {
		lines = append(lines, "Current response: "+emptyDash(invite.CurrentUserAttendee.ParticipationStatus))
	}
	return lines
}

func attendeeDisplay(attendee calendarstore.AttendeeMeta) string {
	if strings.TrimSpace(attendee.Name) != "" && strings.TrimSpace(attendee.Email) != "" {
		return fmt.Sprintf("%s <%s>", attendee.Name, attendee.Email)
	}
	if strings.TrimSpace(attendee.Email) != "" {
		return attendee.Email
	}
	return emptyDash(attendee.Name)
}

func organizerDisplay(event calendarstore.EventMeta) string {
	name := strings.TrimSpace(event.OrganizerName)
	email := strings.TrimSpace(event.OrganizerEmail)
	if name != "" && email != "" {
		return fmt.Sprintf("%s <%s>", name, email)
	}
	if email != "" {
		return email
	}
	return emptyDash(name)
}

func linkedMessagesView(messages []calendarstore.MessageMeta) []string {
	if len(messages) == 0 {
		return []string{"Linked messages: none"}
	}
	lines := []string{"Linked messages:"}
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | inbox:%d | %s", emptyDash(message.Subject), calendarMessageSender(message), formatCalendarMessageTime(message.ReceivedAt), message.InboxID, emptyDash(message.SystemState)))
	}
	return lines
}

func calendarMessageSender(message calendarstore.MessageMeta) string {
	if strings.TrimSpace(message.SenderDisplay) != "" {
		return message.SenderDisplay
	}
	if strings.TrimSpace(message.FromName) != "" && strings.TrimSpace(message.FromAddress) != "" {
		return fmt.Sprintf("%s <%s>", message.FromName, message.FromAddress)
	}
	if strings.TrimSpace(message.FromAddress) != "" {
		return message.FromAddress
	}
	return "-"
}

func formatCalendarMessageTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("2006-01-02 15:04")
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
