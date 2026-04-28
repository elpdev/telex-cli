package screens

import (
	"fmt"
	"time"

	"github.com/elpdev/telex-cli/internal/components/card"
)

func (h Home) makeMailCard() card.Model {
	d := h.summary.mail
	c := card.New(h.theme).WithTitle("MAIL").WithKeyHint("m / 1")
	switch {
	case d.err != nil:
		c = c.WithError("mail cache error — see Logs")
	case !d.hasMailboxes:
		c = c.WithEmpty("No mailbox yet — sync from Mail.")
	default:
		counts := []string{fmt.Sprintf("%d unread", d.unread)}
		if d.drafts > 0 {
			counts = append(counts, fmt.Sprintf("%d drafts", d.drafts))
		}
		if d.outbox > 0 {
			counts = append(counts, fmt.Sprintf("%d outbox", d.outbox))
		}
		c = c.WithCounts(counts...)
		if len(d.recent) == 0 {
			c = c.WithEmpty("Inbox empty.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, m := range d.recent {
				subject := m.subject
				if subject == "" {
					subject = "(no subject)"
				}
				left := subject
				if m.from != "" {
					left = m.from + " — " + subject
				}
				rows = append(rows, card.Row{
					Left:   left,
					Right:  humanAgo(time.Since(m.received)) + " ago",
					Accent: m.unread,
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeCalendarCard() card.Model {
	d := h.summary.calendar
	c := card.New(h.theme).WithTitle("CALENDAR").WithKeyHint("c / 2")
	switch {
	case d.err != nil:
		c = c.WithError("calendar cache error — see Logs")
	case d.syncedAt.IsZero():
		c = c.WithEmpty("No calendar yet — sync from Calendar.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d today", d.today), fmt.Sprintf("%d this week", d.thisWeek))
		if len(d.upcoming) == 0 {
			c = c.WithEmpty("No upcoming events.")
		} else {
			now := time.Now()
			rows := make([]card.Row, 0, len(d.upcoming))
			for _, occ := range d.upcoming {
				rows = append(rows, card.Row{
					Left:  occ.Title,
					Right: formatWhen(occ.StartsAt, occ.AllDay, now),
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeNotesCard() card.Model {
	d := h.summary.notes
	c := card.New(h.theme).WithTitle("NOTES").WithKeyHint("n / 4")
	switch {
	case d.err != nil:
		c = c.WithError("notes cache error — see Logs")
	case d.syncedAt.IsZero() && d.total == 0:
		c = c.WithEmpty("No notes yet — sync from Notes.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d notes", d.total), fmt.Sprintf("%d folders", d.folders))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent edits.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, n := range d.recent {
				title := n.title
				if title == "" {
					title = "(untitled)"
				}
				rows = append(rows, card.Row{
					Left:  title,
					Right: humanAgo(time.Since(n.updated)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeDriveCard() card.Model {
	d := h.summary.drive
	c := card.New(h.theme).WithTitle("DRIVE").WithKeyHint("d / 6")
	switch {
	case d.err != nil:
		c = c.WithError("drive cache error — see Logs")
	case d.syncedAt.IsZero() && d.files == 0:
		c = c.WithEmpty("No files yet — sync from Drive.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d files", d.files), humanBytes(d.bytes))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent uploads.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, f := range d.recent {
				rows = append(rows, card.Row{
					Left:  f.name,
					Right: humanAgo(time.Since(f.updated)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeContactsCard() card.Model {
	d := h.summary.contacts
	c := card.New(h.theme).WithTitle("CONTACTS").WithKeyHint("o / 3")
	switch {
	case d.err != nil:
		c = c.WithError("contacts cache error — see Logs")
	case d.syncedAt.IsZero() && d.total == 0:
		c = c.WithEmpty("No contacts yet — sync from Contacts.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d contacts", d.total), fmt.Sprintf("%d comms", d.comms))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent activity.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, r := range d.recent {
				who := r.who
				if who == "" {
					who = "(unknown)"
				}
				left := who
				if r.subject != "" {
					left = who + " — " + r.subject
				}
				rows = append(rows, card.Row{
					Left:  left,
					Right: humanAgo(time.Since(r.when)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeTasksCard() card.Model {
	d := h.summary.tasks
	c := card.New(h.theme).WithTitle("TASKS").WithKeyHint("t / 5")
	switch {
	case d.err != nil:
		c = c.WithError("tasks cache error — see Logs")
	case d.syncedAt.IsZero() && d.projects == 0:
		c = c.WithEmpty("No tasks yet — sync from Tasks.")
	default:
		c = c.WithCounts(fmt.Sprintf("%d projects", d.projects), fmt.Sprintf("%d cards", d.cards))
		if len(d.recent) == 0 {
			c = c.WithEmpty("No recent updates.")
		} else {
			rows := make([]card.Row, 0, len(d.recent))
			for _, r := range d.recent {
				title := r.title
				if title == "" {
					title = "(untitled)"
				}
				left := title
				if r.project != "" {
					left = r.project + " — " + title
				}
				rows = append(rows, card.Row{
					Left:  left,
					Right: humanAgo(time.Since(r.updated)) + " ago",
				})
			}
			c = c.WithRows(rows)
		}
	}
	return c
}

func (h Home) makeNewsCard() card.Model {
	d := h.summary.news
	c := card.New(h.theme).WithTitle("NEWS").WithKeyHint("w / 7")
	switch {
	case d.err != nil:
		c = c.WithError("news fetch failed — check connection")
	case !d.loaded:
		c = c.WithEmpty("Loading top stories…")
	case len(d.recent) == 0:
		c = c.WithEmpty("No top stories.")
	default:
		c = c.WithCounts("Top stories", "fetched "+humanAgo(time.Since(d.fetched))+" ago")
		rows := make([]card.Row, 0, len(d.recent))
		for _, r := range d.recent {
			title := r.title
			if title == "" {
				title = "(untitled)"
			}
			rows = append(rows, card.Row{
				Left:  title,
				Right: fmt.Sprintf("↑%d", r.score),
			})
		}
		c = c.WithRows(rows)
	}
	return c
}
