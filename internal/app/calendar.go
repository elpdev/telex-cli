package app

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/calendarsync"
	"github.com/elpdev/telex-cli/internal/screens"
)

func (m *Model) syncCalendar(ctx context.Context, from, to string) (screens.CalendarSyncResult, error) {
	service, err := m.calendarService()
	if err != nil {
		return screens.CalendarSyncResult{}, err
	}
	result, err := runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{From: from, To: to})
	return screens.CalendarSyncResult{Calendars: result.Calendars, Events: result.Events, Occurrences: result.Occurrences}, err
}

func (m *Model) createCalendar(ctx context.Context, input calendar.CalendarInput) (*calendar.Calendar, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	created, err := service.CreateCalendar(ctx, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreCalendar(*created, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return created, err
}

func (m *Model) updateCalendar(ctx context.Context, id int64, input calendar.CalendarInput) (*calendar.Calendar, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	updated, err := service.UpdateCalendar(ctx, id, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreCalendar(*updated, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return updated, err
}

func (m *Model) deleteCalendar(ctx context.Context, id int64) error {
	service, err := m.calendarService()
	if err != nil {
		return err
	}
	if err := service.DeleteCalendar(ctx, id); err != nil {
		return err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.DeleteCalendar(id); err != nil {
		return err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return err
}

func (m *Model) importCalendarICS(ctx context.Context, calendarID int64, path string) (*calendar.ImportResult, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	result, err := service.ImportICS(ctx, calendarID, path)
	if err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{})
	return result, err
}

func (m *Model) showCalendarInvitation(ctx context.Context, messageID int64) (*calendar.Invitation, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	invite, err := service.ShowInvitation(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if invite.CalendarEvent != nil {
		if err := calendarstore.New(m.dataPath).StoreEvent(*invite.CalendarEvent, time.Now()); err != nil {
			return nil, err
		}
	}
	return invite, nil
}

func (m *Model) syncCalendarInvitation(ctx context.Context, messageID int64) (*calendar.Invitation, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	invite, err := service.SyncInvitation(ctx, messageID)
	if err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{})
	return invite, err
}

func (m *Model) respondCalendarInvitation(ctx context.Context, messageID int64, input calendar.InvitationInput) (*calendar.Invitation, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	invite, err := service.UpdateInvitation(ctx, messageID, input)
	if err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, calendarstore.New(m.dataPath), service, calendarSyncOptions{})
	return invite, err
}

func (m *Model) createCalendarEvent(ctx context.Context, input calendar.CalendarEventInput) (*calendar.CalendarEvent, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	event, err := service.CreateEvent(ctx, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreEvent(*event, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return event, err
}

func (m *Model) updateCalendarEvent(ctx context.Context, id int64, input calendar.CalendarEventInput) (*calendar.CalendarEvent, error) {
	service, err := m.calendarService()
	if err != nil {
		return nil, err
	}
	event, err := service.UpdateEvent(ctx, id, input)
	if err != nil {
		return nil, err
	}
	store := calendarstore.New(m.dataPath)
	if err := store.StoreEvent(*event, time.Now()); err != nil {
		return nil, err
	}
	_, err = runCalendarSync(ctx, store, service, calendarSyncOptions{})
	return event, err
}

func (m *Model) deleteCalendarEvent(ctx context.Context, id int64) error {
	service, err := m.calendarService()
	if err != nil {
		return err
	}
	if err := service.DeleteEvent(ctx, id); err != nil {
		return err
	}
	return calendarstore.New(m.dataPath).DeleteEvent(id)
}

type calendarSyncResult = calendarsync.Result

type calendarSyncOptions = calendarsync.Options

func runCalendarSync(ctx context.Context, store calendarstore.Store, service *calendar.Service, opts calendarSyncOptions) (calendarSyncResult, error) {
	return calendarsync.Run(ctx, store, service, opts)
}

func calendarDefaultRange(from, to string) (string, string) {
	return calendarsync.DefaultRange(from, to)
}
