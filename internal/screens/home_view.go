package screens

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/elpdev/telex-cli/internal/components/card"
	"strings"
	"time"
)

func (h Home) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if !h.loaded {
		return style.Render(h.theme.Muted.Render("Loading dashboard…"))
	}

	header := h.renderHeader()

	stacked := width < homeStackedBelow
	cardWidth := width / homeGridCols
	if stacked {
		cardWidth = width
	}
	if cardWidth < 32 {
		cardWidth = 32
	}

	sized := make([]card.Model, len(h.cards))
	orphan := !stacked && len(h.cards)%homeGridCols == 1
	for i, c := range h.cards {
		w := cardWidth
		if orphan && i == len(h.cards)-1 {
			w = width
		}
		sized[i] = c.WithWidth(w)
	}

	var grid string
	if stacked {
		parts := make([]string, len(sized))
		for i, c := range sized {
			parts[i] = c.View()
		}
		grid = lipgloss.JoinVertical(lipgloss.Left, parts...)
	} else {
		rows := []string{}
		for i := 0; i < len(sized); i += homeGridCols {
			end := i + homeGridCols
			if end > len(sized) {
				end = len(sized)
			}
			rowViews := make([]string, end-i)
			for j := i; j < end; j++ {
				rowViews[j-i] = sized[j].View()
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowViews...))
		}
		grid = lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", grid, "", h.renderFooter())
	return style.Render(body)
}

func (h Home) renderHeader() string {
	title := h.theme.Title.Render("Telex")
	var sub string
	if h.summary.lastSync.IsZero() {
		sub = h.theme.Muted.Render("No data cached yet — open a module and press S to sync.")
	} else {
		sub = h.theme.Muted.Render(fmt.Sprintf("Last sync %s ago", humanAgo(time.Since(h.summary.lastSync))))
	}
	return title + "  " + sub
}

func (h Home) renderFooter() string {
	parts := []string{
		h.theme.HeaderAccent.Render("ctrl+k") + h.theme.Muted.Render(" palette"),
		h.theme.HeaderAccent.Render("?") + h.theme.Muted.Render(" help"),
		h.theme.HeaderAccent.Render("tab") + h.theme.Muted.Render(" focus"),
		h.theme.HeaderAccent.Render("enter") + h.theme.Muted.Render(" open"),
		h.theme.HeaderAccent.Render("m c o n t d w") + h.theme.Muted.Render(" jump"),
		h.theme.HeaderAccent.Render("r") + h.theme.Muted.Render(" refresh"),
	}
	return strings.Join(parts, h.theme.Muted.Render("  •  "))
}

func (h Home) buildCards() ([]card.Model, []string) {
	cards := []card.Model{
		h.makeMailCard(),
		h.makeCalendarCard(),
		h.makeContactsCard(),
		h.makeNotesCard(),
		h.makeTasksCard(),
		h.makeDriveCard(),
		h.makeNewsCard(),
	}
	ids := []string{"mail", "calendar", "contacts", "notes", "tasks", "drive", "news"}
	for i := range cards {
		if i == h.focusedIdx {
			cards[i] = cards[i].Focus()
		}
	}
	return cards, ids
}

func humanAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
}

func humanBytes(n int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func formatWhen(t time.Time, allDay bool, now time.Time) string {
	if allDay {
		if sameDay(t, now) {
			return "today"
		}
		if sameDay(t, now.AddDate(0, 0, 1)) {
			return "tomorrow"
		}
		return t.Format("Mon Jan 2")
	}
	if sameDay(t, now) {
		return t.Format("15:04")
	}
	if sameDay(t, now.AddDate(0, 0, 1)) {
		return "tmrw " + t.Format("15:04")
	}
	return t.Format("Mon 15:04")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
