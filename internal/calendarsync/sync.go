package calendarsync

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

type Result struct {
	Calendars   int
	Events      int
	Occurrences int
}

type Options struct {
	From       string
	To         string
	CalendarID int64
}

func Run(ctx context.Context, store calendarstore.Store, service *calendar.Service, opts Options) (Result, error) {
	syncedAt := time.Now()
	calendars, _, err := service.ListCalendars(ctx, calendar.ListParams{Page: 1, PerPage: 100})
	if err != nil {
		return Result{}, err
	}
	var result Result
	keepCalendars := map[int64]bool{}
	keepEvents := map[int64]bool{}
	for _, item := range calendars {
		if opts.CalendarID > 0 && item.ID != opts.CalendarID {
			continue
		}
		if err := store.StoreCalendar(item, syncedAt); err != nil {
			return Result{}, err
		}
		keepCalendars[item.ID] = true
		result.Calendars++
		page := 1
		for {
			events, pagination, err := service.ListEvents(ctx, calendar.EventListParams{ListParams: calendar.ListParams{Page: page, PerPage: 100}, CalendarID: item.ID, Sort: "starts_at"})
			if err != nil {
				return Result{}, err
			}
			for _, event := range events {
				messages, err := service.EventMessages(ctx, event.ID)
				if err != nil {
					return Result{}, err
				}
				event.Messages = messages
				if err := store.StoreEvent(event, syncedAt); err != nil {
					return Result{}, err
				}
				keepEvents[event.ID] = true
				result.Events++
			}
			if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
				break
			}
			page++
		}
	}
	if err := store.PruneMissingCalendars(keepCalendars); err != nil {
		return Result{}, err
	}
	if err := store.PruneMissingEvents(keepEvents); err != nil {
		return Result{}, err
	}
	from, to := DefaultRange(opts.From, opts.To)
	occurrences, err := service.ListOccurrences(ctx, calendar.OccurrenceListParams{CalendarID: opts.CalendarID, StartsFrom: from, EndsTo: to})
	if err != nil {
		return Result{}, err
	}
	if err := store.StoreOccurrences(occurrences, syncedAt); err != nil {
		return Result{}, err
	}
	result.Occurrences = len(occurrences)
	return result, nil
}

func latestEventUpdatedSince(store calendarstore.Store) string {
	events, err := store.ListEvents(0)
	if err != nil {
		return ""
	}
	var latest time.Time
	for _, event := range events {
		if event.Meta.RemoteUpdatedAt.After(latest) {
			latest = event.Meta.RemoteUpdatedAt
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339Nano)
}

func DefaultRange(from, to string) (string, string) {
	now := time.Now()
	if from == "" {
		from = now.Format("2006-01-02")
	}
	if to == "" {
		to = now.AddDate(0, 0, 30).Format("2006-01-02")
	}
	return from, to
}
