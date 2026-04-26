package calendar

import (
	"context"
	"net/url"
	"testing"
)

func TestCalendarCRUDUsesExpectedEndpoints(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"id":12,"name":"Work"}],"meta":{"page":1,"per_page":25,"total_count":1}}`)}
	service := NewService(fake)
	_, _, err := service.ListCalendars(context.Background(), ListParams{Page: 1, PerPage: 25})
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/calendars" {
		t.Fatalf("get path = %q", fake.getPath)
	}
	assertQuery(t, fake.query, "page", "1")
	assertQuery(t, fake.query, "per_page", "25")

	fake.body = []byte(`{"data":{"id":12,"name":"Work"}}`)
	pos := 2
	if _, err := service.CreateCalendar(context.Background(), CalendarInput{Name: "Work", Color: "cyan", TimeZone: "UTC", Position: &pos}); err != nil {
		t.Fatal(err)
	}
	if fake.postPath != "/api/v1/calendars" {
		t.Fatalf("post path = %q", fake.postPath)
	}
	if _, err := service.UpdateCalendar(context.Background(), 12, CalendarInput{Name: "Work"}); err != nil {
		t.Fatal(err)
	}
	if fake.patchPath != "/api/v1/calendars/12" {
		t.Fatalf("patch path = %q", fake.patchPath)
	}
	if err := service.DeleteCalendar(context.Background(), 12); err != nil {
		t.Fatal(err)
	}
	if fake.deletePath != "/api/v1/calendars/12" {
		t.Fatalf("delete path = %q", fake.deletePath)
	}
}

func TestEventAndOccurrenceEndpoints(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"id":9,"calendar_id":12,"title":"Standup"}],"meta":{"page":1,"per_page":50,"total_count":1}}`)}
	service := NewService(fake)
	_, _, err := service.ListEvents(context.Background(), EventListParams{ListParams: ListParams{Page: 1, PerPage: 50}, CalendarID: 12, Status: "confirmed", StartsFrom: "2026-04-01", EndsTo: "2026-04-30", Sort: "starts_at"})
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/calendar_events" {
		t.Fatalf("get path = %q", fake.getPath)
	}
	assertQuery(t, fake.query, "calendar_id", "12")
	assertQuery(t, fake.query, "status", "confirmed")
	assertQuery(t, fake.query, "starts_from", "2026-04-01")
	assertQuery(t, fake.query, "ends_to", "2026-04-30")
	assertQuery(t, fake.query, "sort", "starts_at")

	fake.body = []byte(`{"data":{"id":9,"calendar_id":12,"title":"Standup"}}`)
	allDay := false
	_, err = service.CreateEvent(context.Background(), CalendarEventInput{CalendarID: 12, Title: "Standup", AllDay: &allDay, StartDate: "2026-04-25", EndDate: "2026-04-25", StartTime: "09:00", EndTime: "09:30"})
	if err != nil {
		t.Fatal(err)
	}
	if fake.postPath != "/api/v1/calendar_events" {
		t.Fatalf("post path = %q", fake.postPath)
	}
	if _, err := service.UpdateEvent(context.Background(), 9, CalendarEventInput{CalendarID: 12, Title: "Standup"}); err != nil {
		t.Fatal(err)
	}
	if fake.patchPath != "/api/v1/calendar_events/9" {
		t.Fatalf("patch path = %q", fake.patchPath)
	}
	if err := service.DeleteEvent(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if fake.deletePath != "/api/v1/calendar_events/9" {
		t.Fatalf("delete path = %q", fake.deletePath)
	}

	fake.body = []byte(`{"data":[]}`)
	if _, err := service.ListOccurrences(context.Background(), OccurrenceListParams{CalendarID: 12, StartsFrom: "2026-04-01", EndsTo: "2026-04-30"}); err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/calendar_occurrences" {
		t.Fatalf("occurrence path = %q", fake.getPath)
	}

	fake.body = []byte(`{"data":[{"id":42,"subject":"Invite","sender_display":"Alex"}]}`)
	messages, err := service.EventMessages(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/calendar_events/9/messages" {
		t.Fatalf("messages path = %q", fake.getPath)
	}
	if len(messages) != 1 || messages[0].ID != 42 || messages[0].Subject != "Invite" {
		t.Fatalf("messages = %#v", messages)
	}
}

func TestCalendarHelpersUseExpectedEndpoints(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"created":1,"updated":2,"success":true}}`)}
	service := NewService(fake)
	if _, err := service.ImportICS(context.Background(), 12, "/tmp/calendar.ics"); err != nil {
		t.Fatal(err)
	}
	if fake.multipartPath != "/api/v1/calendars/12/import_ics" || fake.multipartFile != "/tmp/calendar.ics" {
		t.Fatalf("multipart = %q %q", fake.multipartPath, fake.multipartFile)
	}

	fake.body = []byte(`{"data":{"message_id":99,"available":true}}`)
	if _, err := service.ShowInvitation(context.Background(), 99); err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/messages/99/invitation" {
		t.Fatalf("invitation path = %q", fake.getPath)
	}
	if _, err := service.SyncInvitation(context.Background(), 99); err != nil {
		t.Fatal(err)
	}
	if fake.postPath != "/api/v1/messages/99/invitation/sync" {
		t.Fatalf("sync path = %q", fake.postPath)
	}
	if _, err := service.UpdateInvitation(context.Background(), 99, InvitationInput{ParticipationStatus: "accepted"}); err != nil {
		t.Fatal(err)
	}
	if fake.patchPath != "/api/v1/messages/99/invitation" {
		t.Fatalf("patch path = %q", fake.patchPath)
	}
}

type fakeClient struct {
	body          []byte
	getPath       string
	postPath      string
	patchPath     string
	deletePath    string
	multipartPath string
	multipartFile string
	query         url.Values
}

func (f *fakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, int, error) {
	f.getPath = path
	f.query = query
	return f.body, 200, nil
}

func (f *fakeClient) Post(_ context.Context, path string, _ any) ([]byte, int, error) {
	f.postPath = path
	return f.body, 200, nil
}

func (f *fakeClient) PostMultipartFile(_ context.Context, path, _ string, filePath string) ([]byte, int, error) {
	f.multipartPath = path
	f.multipartFile = filePath
	return f.body, 200, nil
}

func (f *fakeClient) Patch(_ context.Context, path string, _ any) ([]byte, int, error) {
	f.patchPath = path
	return f.body, 200, nil
}

func (f *fakeClient) Delete(_ context.Context, path string) (int, error) {
	f.deletePath = path
	return 204, nil
}

func assertQuery(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Fatalf("query[%s] = %q, want %q", key, got, want)
	}
}
