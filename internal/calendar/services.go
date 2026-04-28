package calendar

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/api"
)

type Client interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
	Post(context.Context, string, any) ([]byte, int, error)
	PostMultipartFile(context.Context, string, string, string) ([]byte, int, error)
	Patch(context.Context, string, any) ([]byte, int, error)
	Delete(context.Context, string) (int, error)
}

type Service struct {
	client Client
}

func NewService(client Client) *Service { return &Service{client: client} }

func (s *Service) ListCalendars(ctx context.Context, params ListParams) ([]Calendar, *api.Pagination, error) {
	return api.List[Calendar](s.client, ctx, "/api/v1/calendars", listQuery(params))
}

func (s *Service) ShowCalendar(ctx context.Context, id int64) (*Calendar, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/calendars/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Calendar](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateCalendar(ctx context.Context, input CalendarInput) (*Calendar, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/calendars", map[string]any{"calendar": calendarInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Calendar](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateCalendar(ctx context.Context, id int64, input CalendarInput) (*Calendar, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/calendars/%d", id), map[string]any{"calendar": calendarInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Calendar](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteCalendar(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/calendars/%d", id))
	return err
}

func (s *Service) ImportICS(ctx context.Context, calendarID int64, filePath string) (*ImportResult, error) {
	body, _, err := s.client.PostMultipartFile(ctx, fmt.Sprintf("/api/v1/calendars/%d/import_ics", calendarID), "file", filePath)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[ImportResult](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ListEvents(ctx context.Context, params EventListParams) ([]CalendarEvent, *api.Pagination, error) {
	return api.List[CalendarEvent](s.client, ctx, "/api/v1/calendar_events", eventQuery(params))
}

func (s *Service) ShowEvent(ctx context.Context, id int64) (*CalendarEvent, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/calendar_events/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[CalendarEvent](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateEvent(ctx context.Context, input CalendarEventInput) (*CalendarEvent, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/calendar_events", map[string]any{"calendar_event": eventInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[CalendarEvent](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateEvent(ctx context.Context, id int64, input CalendarEventInput) (*CalendarEvent, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/calendar_events/%d", id), map[string]any{"calendar_event": eventInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[CalendarEvent](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteEvent(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/calendar_events/%d", id))
	return err
}

func (s *Service) EventMessages(ctx context.Context, id int64) ([]MessageSummary, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/calendar_events/%d/messages", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]MessageSummary](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ListOccurrences(ctx context.Context, params OccurrenceListParams) ([]CalendarOccurrence, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/calendar_occurrences", occurrenceQuery(params))
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]CalendarOccurrence](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ShowInvitation(ctx context.Context, messageID int64) (*Invitation, error) {
	return s.invitation(ctx, messageID, "show", InvitationInput{})
}

func (s *Service) SyncInvitation(ctx context.Context, messageID int64) (*Invitation, error) {
	return s.invitation(ctx, messageID, "sync", InvitationInput{})
}

func (s *Service) UpdateInvitation(ctx context.Context, messageID int64, input InvitationInput) (*Invitation, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/messages/%d/invitation", messageID), map[string]any{"invitation": map[string]any{"participation_status": input.ParticipationStatus}})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Invitation](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) invitation(ctx context.Context, messageID int64, action string, _ InvitationInput) (*Invitation, error) {
	path := fmt.Sprintf("/api/v1/messages/%d/invitation", messageID)
	var body []byte
	var err error
	if action == "sync" {
		body, _, err = s.client.Post(ctx, path+"/sync", nil)
	} else {
		body, _, err = s.client.Get(ctx, path, nil)
	}
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Invitation](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func listQuery(params ListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	api.SetString(query, "updated_since", params.UpdatedSince)
	return query
}

func eventQuery(params EventListParams) url.Values {
	query := listQuery(params.ListParams)
	api.SetInt64(query, "calendar_id", params.CalendarID)
	api.SetString(query, "status", params.Status)
	api.SetString(query, "source", params.Source)
	api.SetString(query, "uid", params.UID)
	api.SetString(query, "starts_from", params.StartsFrom)
	api.SetString(query, "ends_to", params.EndsTo)
	api.SetString(query, "updated_since", params.UpdatedSince)
	api.SetString(query, "sort", params.Sort)
	return query
}

func occurrenceQuery(params OccurrenceListParams) url.Values {
	query := url.Values{}
	api.SetInt64(query, "calendar_id", params.CalendarID)
	if len(params.CalendarIDs) > 0 {
		ids := make([]string, 0, len(params.CalendarIDs))
		for _, id := range params.CalendarIDs {
			if id > 0 {
				ids = append(ids, strconv.FormatInt(id, 10))
			}
		}
		if len(ids) > 0 {
			query.Set("calendar_ids", strings.Join(ids, ","))
		}
	}
	api.SetString(query, "starts_from", params.StartsFrom)
	api.SetString(query, "ends_to", params.EndsTo)
	return query
}

func calendarInputMap(input CalendarInput) map[string]any {
	payload := map[string]any{}
	setPayloadString(payload, "name", input.Name)
	setPayloadString(payload, "color", input.Color)
	setPayloadString(payload, "time_zone", input.TimeZone)
	if input.Position != nil {
		payload["position"] = *input.Position
	}
	return payload
}

func eventInputMap(input CalendarEventInput) map[string]any {
	payload := map[string]any{"calendar_id": input.CalendarID}
	setPayloadString(payload, "title", input.Title)
	setPayloadString(payload, "description", input.Description)
	setPayloadString(payload, "location", input.Location)
	if input.AllDay != nil {
		payload["all_day"] = *input.AllDay
	}
	setPayloadString(payload, "start_date", input.StartDate)
	setPayloadString(payload, "end_date", input.EndDate)
	setPayloadString(payload, "start_time", input.StartTime)
	setPayloadString(payload, "end_time", input.EndTime)
	setPayloadString(payload, "time_zone", input.TimeZone)
	setPayloadString(payload, "status", input.Status)
	setPayloadString(payload, "recurrence_frequency", input.RecurrenceFrequency)
	if input.RecurrenceInterval != nil {
		payload["recurrence_interval"] = *input.RecurrenceInterval
	}
	setPayloadString(payload, "recurrence_until", input.RecurrenceUntil)
	if len(input.RecurrenceWeekdays) > 0 {
		payload["recurrence_weekdays"] = input.RecurrenceWeekdays
	}
	return payload
}

func setPayloadString(payload map[string]any, key, value string) {
	if value != "" {
		payload[key] = value
	}
}
