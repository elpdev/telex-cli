package screens

import (
	"context"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

func (h Home) loadCmd() tea.Cmd {
	mail := h.mail
	cal := h.calendar
	notes := h.notes
	drive := h.drive
	tasks := h.tasks
	contacts := h.contacts
	summaryCmd := func() tea.Msg {
		return homeLoadedMsg{summary: collectHomeSummary(mail, cal, notes, drive, tasks, contacts)}
	}
	return tea.Batch(summaryCmd, h.newsLoadCmd())
}

func (h Home) newsLoadCmd() tea.Cmd {
	fetcher := h.news
	return func() tea.Msg {
		return homeNewsLoadedMsg{news: collectNewsCard(fetcher)}
	}
}

func collectHomeSummary(
	mail mailstore.Store,
	cal calendarstore.Store,
	notes notestore.Store,
	drive drivestore.Store,
	tasks taskstore.Store,
	contacts contactstore.Store,
) homeSummary {
	s := homeSummary{
		mail:     collectMailCard(mail),
		calendar: collectCalendarCard(cal),
		notes:    collectNotesCard(notes),
		drive:    collectDriveCard(drive),
		tasks:    collectTasksCard(tasks),
		contacts: collectContactsCard(contacts),
	}
	for _, t := range []time.Time{
		s.mail.syncedAt,
		s.calendar.syncedAt,
		s.notes.syncedAt,
		s.drive.syncedAt,
		s.tasks.syncedAt,
		s.contacts.syncedAt,
	} {
		if t.After(s.lastSync) {
			s.lastSync = t
		}
	}
	return s
}

func collectMailCard(store mailstore.Store) mailCardData {
	boxes, err := store.ListMailboxes()
	if err != nil {
		return mailCardData{err: err}
	}
	data := mailCardData{hasMailboxes: len(boxes) > 0}
	var inbox []mailstore.CachedMessage
	for _, box := range boxes {
		if box.SyncedAt.After(data.syncedAt) {
			data.syncedAt = box.SyncedAt
		}
		path, perr := store.MailboxPath(box.DomainName, box.LocalPart)
		if perr != nil {
			continue
		}
		messages, _ := mailstore.ListInbox(path)
		for _, m := range messages {
			if !m.Meta.Read {
				data.unread++
			}
		}
		inbox = append(inbox, messages...)
		drafts, _ := mailstore.ListMessages(path, "drafts")
		data.drafts += len(drafts)
		outbox, _ := mailstore.ListMessages(path, "outbox")
		data.outbox += len(outbox)
	}
	sort.Slice(inbox, func(i, j int) bool {
		return inbox[i].Meta.ReceivedAt.After(inbox[j].Meta.ReceivedAt)
	})
	limit := 3
	if len(inbox) < limit {
		limit = len(inbox)
	}
	for _, m := range inbox[:limit] {
		from := m.Meta.FromName
		if from == "" {
			from = m.Meta.FromAddress
		}
		data.recent = append(data.recent, mailRecent{
			subject:  m.Meta.Subject,
			from:     from,
			received: m.Meta.ReceivedAt,
			unread:   !m.Meta.Read,
		})
	}
	return data
}

func collectCalendarCard(store calendarstore.Store) calendarCardData {
	occ, err := store.ListOccurrences()
	if err != nil {
		return calendarCardData{err: err}
	}
	data := calendarCardData{}
	now := time.Now()
	weekEnd := now.AddDate(0, 0, 7)
	for _, o := range occ {
		if o.SyncedAt.After(data.syncedAt) {
			data.syncedAt = o.SyncedAt
		}
		if sameDay(o.StartsAt, now) {
			data.today++
		}
		if !o.StartsAt.Before(now) && o.StartsAt.Before(weekEnd) {
			data.thisWeek++
		}
		if !o.StartsAt.Before(now) && len(data.upcoming) < 3 {
			data.upcoming = append(data.upcoming, o)
		}
	}
	return data
}

func collectNotesCard(store notestore.Store) notesCardData {
	data := notesCardData{}
	if total, folders, err := store.Counts(); err == nil {
		data.total = total
		data.folders = folders
	} else {
		data.err = err
	}
	all, err := store.AllNotes()
	if err != nil {
		if data.err == nil {
			data.err = err
		}
		return data
	}
	for _, n := range all {
		if n.Meta.SyncedAt.After(data.syncedAt) {
			data.syncedAt = n.Meta.SyncedAt
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Meta.RemoteUpdatedAt.After(all[j].Meta.RemoteUpdatedAt)
	})
	limit := 3
	if len(all) < limit {
		limit = len(all)
	}
	for _, n := range all[:limit] {
		data.recent = append(data.recent, notesRecent{title: n.Meta.Title, updated: n.Meta.RemoteUpdatedAt})
	}
	return data
}

func collectDriveCard(store drivestore.Store) driveCardData {
	files, err := store.AllFiles()
	if err != nil {
		return driveCardData{err: err}
	}
	data := driveCardData{files: len(files)}
	for _, f := range files {
		data.bytes += f.ByteSize
		if f.SyncedAt.After(data.syncedAt) {
			data.syncedAt = f.SyncedAt
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].RemoteUpdatedAt.After(files[j].RemoteUpdatedAt)
	})
	limit := 3
	if len(files) < limit {
		limit = len(files)
	}
	for _, f := range files[:limit] {
		data.recent = append(data.recent, driveRecent{name: f.Filename, updated: f.RemoteUpdatedAt})
	}
	return data
}

func collectTasksCard(store taskstore.Store) tasksCardData {
	projects, err := store.ListProjects()
	if err != nil {
		return tasksCardData{err: err}
	}
	data := tasksCardData{projects: len(projects)}
	type recent struct {
		title   string
		project string
		updated time.Time
	}
	var all []recent
	for _, p := range projects {
		if p.Meta.SyncedAt.After(data.syncedAt) {
			data.syncedAt = p.Meta.SyncedAt
		}
		cards, _ := store.ListCards(p.Meta.RemoteID)
		data.cards += len(cards)
		for _, c := range cards {
			all = append(all, recent{title: c.Meta.Title, project: p.Meta.Name, updated: c.Meta.RemoteUpdatedAt})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].updated.After(all[j].updated) })
	limit := 3
	if len(all) < limit {
		limit = len(all)
	}
	for _, r := range all[:limit] {
		data.recent = append(data.recent, tasksRecent{title: r.title, project: r.project, updated: r.updated})
	}
	return data
}

func collectContactsCard(store contactstore.Store) contactsCardData {
	contacts, err := store.ListContacts()
	if err != nil {
		return contactsCardData{err: err}
	}
	data := contactsCardData{total: len(contacts)}
	type recent struct {
		who     string
		subject string
		when    time.Time
	}
	var all []recent
	for _, ct := range contacts {
		if ct.Meta.SyncedAt.After(data.syncedAt) {
			data.syncedAt = ct.Meta.SyncedAt
		}
		data.comms += len(ct.Communications)
		for _, comm := range ct.Communications {
			all = append(all, recent{who: ct.Meta.DisplayName, subject: comm.Subject, when: comm.OccurredAt})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].when.After(all[j].when) })
	if len(all) == 0 && len(contacts) > 0 {
		sorted := append([]contactstore.CachedContact(nil), contacts...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Meta.RemoteUpdatedAt.After(sorted[j].Meta.RemoteUpdatedAt)
		})
		for i, ct := range sorted {
			if i >= 3 {
				break
			}
			all = append(all, recent{who: ct.Meta.DisplayName, subject: "added", when: ct.Meta.RemoteUpdatedAt})
		}
	}
	limit := 3
	if len(all) < limit {
		limit = len(all)
	}
	for _, r := range all[:limit] {
		data.recent = append(data.recent, contactsRecent{who: r.who, subject: r.subject, when: r.when})
	}
	return data
}

func collectNewsCard(fetcher NewsFetcher) newsCardData {
	if fetcher == nil {
		return newsCardData{loaded: true, fetched: time.Now()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), homeNewsTimeout)
	defer cancel()
	items, err := fetcher(ctx, homeNewsLimit)
	if err != nil {
		return newsCardData{loaded: true, fetched: time.Now(), err: err}
	}
	data := newsCardData{loaded: true, fetched: time.Now()}
	for _, item := range items {
		data.recent = append(data.recent, newsRecent{
			title:    item.Title,
			score:    item.Score,
			comments: item.Descendants,
			posted:   item.CreatedAt(),
		})
	}
	return data
}
