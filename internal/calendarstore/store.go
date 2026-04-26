package calendarstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

const SchemaVersion = 1

type Store struct {
	Root string
}

type CalendarMeta struct {
	SchemaVersion   int       `toml:"schema_version"`
	RemoteID        int64     `toml:"remote_id"`
	UserID          int64     `toml:"user_id"`
	Name            string    `toml:"name"`
	Color           string    `toml:"color"`
	TimeZone        string    `toml:"time_zone"`
	Position        int       `toml:"position"`
	Source          string    `toml:"source"`
	RemoteCreatedAt time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt time.Time `toml:"remote_updated_at"`
	SyncedAt        time.Time `toml:"synced_at"`
}

type EventMeta struct {
	SchemaVersion        int            `toml:"schema_version"`
	RemoteID             int64          `toml:"remote_id"`
	CalendarID           int64          `toml:"calendar_id"`
	Title                string         `toml:"title"`
	Location             string         `toml:"location"`
	AllDay               bool           `toml:"all_day"`
	StartsAt             time.Time      `toml:"starts_at"`
	EndsAt               time.Time      `toml:"ends_at"`
	TimeZone             string         `toml:"time_zone"`
	Status               string         `toml:"status"`
	Source               string         `toml:"source"`
	UID                  string         `toml:"uid"`
	OrganizerName        string         `toml:"organizer_name"`
	OrganizerEmail       string         `toml:"organizer_email"`
	RecurrenceRule       string         `toml:"recurrence_rule"`
	RecurrenceSummary    string         `toml:"recurrence_summary"`
	RecurrenceExceptions []string       `toml:"recurrence_exceptions"`
	NextOccurrences      []time.Time    `toml:"next_occurrences"`
	Invitation           bool           `toml:"invitation"`
	Attendees            []AttendeeMeta `toml:"attendees"`
	CurrentUserAttendee  *AttendeeMeta  `toml:"current_user_attendee"`
	Links                []LinkMeta     `toml:"links"`
	Messages             []MessageMeta  `toml:"messages"`
	RemoteCreatedAt      time.Time      `toml:"remote_created_at"`
	RemoteUpdatedAt      time.Time      `toml:"remote_updated_at"`
	SyncedAt             time.Time      `toml:"synced_at"`
}

type AttendeeMeta struct {
	ID                  int64  `toml:"id"`
	Email               string `toml:"email"`
	Name                string `toml:"name"`
	Role                string `toml:"role"`
	ParticipationStatus string `toml:"participation_status"`
	ResponseRequested   bool   `toml:"response_requested"`
}

type LinkMeta struct {
	ID             int64  `toml:"id"`
	MessageID      int64  `toml:"message_id"`
	ICalUID        string `toml:"ical_uid"`
	ICalMethod     string `toml:"ical_method"`
	SequenceNumber int    `toml:"sequence_number"`
}

type MessageMeta struct {
	ID             int64     `toml:"id"`
	InboxID        int64     `toml:"inbox_id"`
	ConversationID int64     `toml:"conversation_id"`
	Subject        string    `toml:"subject"`
	FromAddress    string    `toml:"from_address"`
	FromName       string    `toml:"from_name"`
	SenderDisplay  string    `toml:"sender_display"`
	PreviewText    string    `toml:"preview_text"`
	ReceivedAt     time.Time `toml:"received_at"`
	SystemState    string    `toml:"system_state"`
}

type OccurrenceMeta struct {
	SchemaVersion int       `toml:"schema_version"`
	EventID       int64     `toml:"event_id"`
	CalendarID    int64     `toml:"calendar_id"`
	Title         string    `toml:"title"`
	Location      string    `toml:"location"`
	StartsAt      time.Time `toml:"starts_at"`
	EndsAt        time.Time `toml:"ends_at"`
	AllDay        bool      `toml:"all_day"`
	Status        string    `toml:"status"`
	Source        string    `toml:"source"`
	SyncedAt      time.Time `toml:"synced_at"`
}

type CachedEvent struct {
	Meta        EventMeta
	Description string
	Path        string
}

func New(root string) Store {
	if root == "" {
		root = mailstore.DefaultRoot()
	}
	return Store{Root: root}
}

func (s Store) CalendarRoot() string { return filepath.Join(s.Root, "calendar") }

func (s Store) EnsureRoot() error {
	for _, dir := range []string{s.CalendarRoot(), s.calendarsRoot(), s.eventsRoot(), s.occurrencesRoot()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) StoreCalendar(value calendar.Calendar, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	meta := CalendarMeta{SchemaVersion: SchemaVersion, RemoteID: value.ID, UserID: value.UserID, Name: value.Name, Color: value.Color, TimeZone: value.TimeZone, Position: value.Position, Source: value.Source, RemoteCreatedAt: value.CreatedAt, RemoteUpdatedAt: value.UpdatedAt, SyncedAt: syncedAt}
	return writeTOML(s.calendarPath(value.ID), meta)
}

func (s Store) ListCalendars() ([]CalendarMeta, error) {
	entries, err := os.ReadDir(s.calendarsRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CalendarMeta{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		var meta CalendarMeta
		if _, err := toml.DecodeFile(filepath.Join(s.calendarsRoot(), entry.Name()), &meta); err != nil {
			return nil, err
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Position == out[j].Position {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}
		return out[i].Position < out[j].Position
	})
	return out, nil
}

func (s Store) DeleteCalendar(id int64) error {
	if err := os.Remove(s.calendarPath(id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	events, err := s.ListEvents(id)
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := s.DeleteEvent(event.Meta.RemoteID); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) StoreEvent(value calendar.CalendarEvent, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.eventPath(value.ID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	meta := EventMeta{SchemaVersion: SchemaVersion, RemoteID: value.ID, CalendarID: value.CalendarID, Title: value.Title, Location: value.Location, AllDay: value.AllDay, StartsAt: value.StartsAt, EndsAt: value.EndsAt, TimeZone: value.TimeZone, Status: value.Status, Source: value.Source, UID: value.UID, OrganizerName: value.OrganizerName, OrganizerEmail: value.OrganizerEmail, RecurrenceRule: value.RecurrenceRule, RecurrenceSummary: value.RecurrenceSummary, RecurrenceExceptions: value.RecurrenceExceptions, NextOccurrences: value.NextOccurrences, Invitation: value.Invitation, Attendees: attendeeMetas(value.Attendees), CurrentUserAttendee: attendeeMeta(value.CurrentUserAttendee), Links: linkMetas(value.Links), Messages: messageMetas(value.Messages), RemoteCreatedAt: value.CreatedAt, RemoteUpdatedAt: value.UpdatedAt, SyncedAt: syncedAt}
	if err := writeTOML(filepath.Join(path, "meta.toml"), meta); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "description.txt"), []byte(value.Description), 0o600)
}

func (s Store) ListEvents(calendarID int64) ([]CachedEvent, error) {
	entries, err := os.ReadDir(s.eventsRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CachedEvent{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cached, err := s.ReadEventPath(filepath.Join(s.eventsRoot(), entry.Name()))
		if err != nil {
			continue
		}
		if calendarID == 0 || cached.Meta.CalendarID == calendarID {
			out = append(out, *cached)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Meta.StartsAt.Equal(out[j].Meta.StartsAt) {
			return out[i].Meta.RemoteID < out[j].Meta.RemoteID
		}
		return out[i].Meta.StartsAt.Before(out[j].Meta.StartsAt)
	})
	return out, nil
}

func (s Store) ReadEvent(id int64) (*CachedEvent, error) { return s.ReadEventPath(s.eventPath(id)) }

func (s Store) ReadEventPath(path string) (*CachedEvent, error) {
	var meta EventMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(path, "description.txt"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return &CachedEvent{Meta: meta, Description: string(body), Path: path}, nil
}

func (s Store) DeleteEvent(id int64) error {
	if err := s.DeleteEventOccurrences(id); err != nil {
		return err
	}
	err := os.RemoveAll(s.eventPath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) DeleteEventOccurrences(id int64) error {
	entries, err := os.ReadDir(s.occurrencesRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		path := filepath.Join(s.occurrencesRoot(), entry.Name())
		var meta OccurrenceMeta
		if _, err := toml.DecodeFile(path, &meta); err != nil {
			return err
		}
		if meta.EventID == id {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func (s Store) StoreOccurrences(values []calendar.CalendarOccurrence, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	if err := os.RemoveAll(s.occurrencesRoot()); err != nil {
		return err
	}
	if err := os.MkdirAll(s.occurrencesRoot(), 0o700); err != nil {
		return err
	}
	for i, value := range values {
		meta := OccurrenceMeta{SchemaVersion: SchemaVersion, EventID: value.Event.ID, CalendarID: value.Event.CalendarID, Title: value.Event.Title, Location: value.Event.Location, StartsAt: value.StartsAt, EndsAt: value.EndsAt, AllDay: value.AllDay, Status: value.Event.Status, Source: value.Event.Source, SyncedAt: syncedAt}
		if err := writeTOML(filepath.Join(s.occurrencesRoot(), fmt.Sprintf("%06d-%d.toml", i, value.Event.ID)), meta); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ListOccurrences() ([]OccurrenceMeta, error) {
	entries, err := os.ReadDir(s.occurrencesRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []OccurrenceMeta{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		var meta OccurrenceMeta
		if _, err := toml.DecodeFile(filepath.Join(s.occurrencesRoot(), entry.Name()), &meta); err != nil {
			return nil, err
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return out, nil
}

func (s Store) ListOccurrencesRange(from, to time.Time) ([]OccurrenceMeta, error) {
	items, err := s.ListOccurrences()
	if err != nil {
		return nil, err
	}
	if from.IsZero() && to.IsZero() {
		return items, nil
	}
	out := make([]OccurrenceMeta, 0, len(items))
	for _, item := range items {
		if !from.IsZero() && item.StartsAt.Before(from) {
			continue
		}
		if !to.IsZero() && !item.StartsAt.Before(to) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func attendeeMetas(attendees []calendar.CalendarEventAttendee) []AttendeeMeta {
	out := make([]AttendeeMeta, 0, len(attendees))
	for _, attendee := range attendees {
		out = append(out, attendeeMetaValue(attendee))
	}
	return out
}

func attendeeMeta(attendee *calendar.CalendarEventAttendee) *AttendeeMeta {
	if attendee == nil {
		return nil
	}
	meta := attendeeMetaValue(*attendee)
	return &meta
}

func attendeeMetaValue(attendee calendar.CalendarEventAttendee) AttendeeMeta {
	return AttendeeMeta{ID: attendee.ID, Email: attendee.Email, Name: attendee.Name, Role: attendee.Role, ParticipationStatus: attendee.ParticipationStatus, ResponseRequested: attendee.ResponseRequested}
}

func linkMetas(links []calendar.CalendarEventLink) []LinkMeta {
	out := make([]LinkMeta, 0, len(links))
	for _, link := range links {
		out = append(out, LinkMeta{ID: link.ID, MessageID: link.MessageID, ICalUID: link.ICalUID, ICalMethod: link.ICalMethod, SequenceNumber: link.SequenceNumber})
	}
	return out
}

func messageMetas(messages []calendar.MessageSummary) []MessageMeta {
	out := make([]MessageMeta, 0, len(messages))
	for _, message := range messages {
		out = append(out, MessageMeta{ID: message.ID, InboxID: message.InboxID, ConversationID: message.ConversationID, Subject: message.Subject, FromAddress: message.FromAddress, FromName: message.FromName, SenderDisplay: message.SenderDisplay, PreviewText: message.PreviewText, ReceivedAt: message.ReceivedAt, SystemState: message.SystemState})
	}
	return out
}

func (s Store) calendarsRoot() string { return filepath.Join(s.CalendarRoot(), "calendars") }

func (s Store) eventsRoot() string { return filepath.Join(s.CalendarRoot(), "events") }

func (s Store) occurrencesRoot() string { return filepath.Join(s.CalendarRoot(), "occurrences") }

func (s Store) calendarPath(id int64) string {
	return filepath.Join(s.calendarsRoot(), fmt.Sprintf("%d.toml", id))
}

func (s Store) eventPath(id int64) string {
	return filepath.Join(s.eventsRoot(), fmt.Sprintf("%d", id))
}

func writeTOML(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(value); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
