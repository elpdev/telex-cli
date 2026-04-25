package calendar

import "time"

type ListParams struct {
	Page    int
	PerPage int
}

type Calendar struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	TimeZone  string    `json:"time_zone"`
	Position  int       `json:"position"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CalendarInput struct {
	Name     string
	Color    string
	TimeZone string
	Position *int
}

type EventListParams struct {
	ListParams
	CalendarID int64
	Status     string
	Source     string
	UID        string
	StartsFrom string
	EndsTo     string
	Sort       string
}

type OccurrenceListParams struct {
	CalendarID  int64
	CalendarIDs []int64
	StartsFrom  string
	EndsTo      string
}

type CalendarEvent struct {
	ID                   int64                   `json:"id"`
	CalendarID           int64                   `json:"calendar_id"`
	Title                string                  `json:"title"`
	Description          string                  `json:"description"`
	Location             string                  `json:"location"`
	AllDay               bool                    `json:"all_day"`
	StartsAt             time.Time               `json:"starts_at"`
	EndsAt               time.Time               `json:"ends_at"`
	TimeZone             string                  `json:"time_zone"`
	EffectiveTimeZone    string                  `json:"effective_time_zone"`
	Status               string                  `json:"status"`
	Source               string                  `json:"source"`
	UID                  string                  `json:"uid"`
	OrganizerName        string                  `json:"organizer_name"`
	OrganizerEmail       string                  `json:"organizer_email"`
	RecurrenceRule       string                  `json:"recurrence_rule"`
	RecurrenceSummary    string                  `json:"recurrence_summary"`
	RecurrenceExceptions []string                `json:"recurrence_exceptions"`
	SequenceNumber       int                     `json:"sequence_number"`
	Invitation           bool                    `json:"invitation"`
	NextOccurrences      []time.Time             `json:"next_occurrences"`
	Attendees            []CalendarEventAttendee `json:"attendees"`
	Links                []CalendarEventLink     `json:"links"`
	CurrentUserAttendee  *CalendarEventAttendee  `json:"current_user_attendee"`
	Messages             []MessageSummary        `json:"messages"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

type CalendarEventInput struct {
	CalendarID          int64
	Title               string
	Description         string
	Location            string
	AllDay              *bool
	StartDate           string
	EndDate             string
	StartTime           string
	EndTime             string
	TimeZone            string
	Status              string
	RecurrenceFrequency string
	RecurrenceInterval  *int
	RecurrenceUntil     string
	RecurrenceWeekdays  []string
}

type CalendarEventAttendee struct {
	ID                  int64     `json:"id"`
	Email               string    `json:"email"`
	Name                string    `json:"name"`
	Role                string    `json:"role"`
	ParticipationStatus string    `json:"participation_status"`
	ResponseRequested   bool      `json:"response_requested"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CalendarEventLink struct {
	ID             int64     `json:"id"`
	MessageID      int64     `json:"message_id"`
	ICalUID        string    `json:"ical_uid"`
	ICalMethod     string    `json:"ical_method"`
	SequenceNumber int       `json:"sequence_number"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CalendarOccurrence struct {
	StartsAt time.Time     `json:"starts_at"`
	EndsAt   time.Time     `json:"ends_at"`
	AllDay   bool          `json:"all_day"`
	Event    CalendarEvent `json:"event"`
}

type MessageSummary struct {
	ID             int64     `json:"id"`
	InboxID        int64     `json:"inbox_id"`
	ConversationID int64     `json:"conversation_id"`
	Subject        string    `json:"subject"`
	FromAddress    string    `json:"from_address"`
	FromName       string    `json:"from_name"`
	SenderDisplay  string    `json:"sender_display"`
	PreviewText    string    `json:"preview_text"`
	ReceivedAt     time.Time `json:"received_at"`
	SystemState    string    `json:"system_state"`
}

type Invitation struct {
	MessageID           int64                  `json:"message_id"`
	Available           bool                   `json:"available"`
	InvitationData      map[string]any         `json:"invitation_data"`
	CalendarEvent       *CalendarEvent         `json:"calendar_event"`
	CurrentUserAttendee *CalendarEventAttendee `json:"current_user_attendee"`
}

type InvitationInput struct {
	ParticipationStatus string
}

type ImportResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
	Success bool     `json:"success"`
}
