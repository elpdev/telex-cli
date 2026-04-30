package screens

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c Calendar) modeAllowsFilter() bool {
	return c.mode == calendarViewAgenda || c.mode == calendarViewWeek || c.mode == calendarViewMonth
}

func (c Calendar) modeAllowsRangeShift() bool {
	return c.mode == calendarViewAgenda || c.mode == calendarViewWeek || c.mode == calendarViewMonth
}

func startOfWeek(t time.Time, weekStartsOn time.Weekday) time.Time {
	day := calendarRangeDate(t)
	offset := (int(day.Weekday()) - int(weekStartsOn) + 7) % 7
	return day.AddDate(0, 0, -offset)
}

func monthGridDays(anchor time.Time, weekStartsOn time.Weekday) [6][7]time.Time {
	if anchor.IsZero() {
		anchor = calendarRangeDate(time.Now())
	}
	first := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, anchor.Location())
	offset := (int(first.Weekday()) - int(weekStartsOn) + 7) % 7
	start := first.AddDate(0, 0, -offset)
	var grid [6][7]time.Time
	for r := 0; r < 6; r++ {
		for d := 0; d < 7; d++ {
			grid[r][d] = start.AddDate(0, 0, r*7+d)
		}
	}
	return grid
}

func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}

func bucketByDay(items []calendarstore.OccurrenceMeta) map[string][]calendarstore.OccurrenceMeta {
	out := make(map[string][]calendarstore.OccurrenceMeta, len(items))
	for _, item := range items {
		key := dayKey(item.StartsAt)
		out[key] = append(out[key], item)
	}
	return out
}

func firstAgendaIndexForDate(items []calendarstore.OccurrenceMeta, day time.Time) int {
	target := dayKey(day)
	for i, item := range items {
		if dayKey(item.StartsAt) == target {
			return i
		}
	}
	return -1
}

func firstAgendaIndexForSlot(items []calendarstore.OccurrenceMeta, day time.Time, hour int) int {
	target := dayKey(day)
	best := -1
	for i, item := range items {
		if dayKey(item.StartsAt) != target {
			continue
		}
		if item.AllDay {
			if best < 0 {
				best = i
			}
			continue
		}
		startHour := item.StartsAt.Hour()
		endHour := item.EndsAt.Hour()
		if !item.EndsAt.After(item.StartsAt) {
			endHour = startHour
		}
		if hour >= startHour && hour <= endHour {
			return i
		}
		if best < 0 && startHour >= hour {
			best = i
		}
	}
	return best
}

type weekSlot struct {
	EventID       int64
	CalendarID    int64
	Title         string
	Status        string
	Continuation  bool
	OverflowCount int
}

func placeWeekEvents(occs []calendarstore.OccurrenceMeta, weekStart time.Time, slotsPerHour, fromH, toH int) (timed [7][]weekSlot, allDay [7][]calendarstore.OccurrenceMeta) {
	if slotsPerHour <= 0 {
		slotsPerHour = 1
	}
	if toH <= fromH {
		return
	}
	slotsTotal := (toH - fromH) * slotsPerHour
	for col := 0; col < 7; col++ {
		timed[col] = make([]weekSlot, slotsTotal)
	}
	minutesPerSlot := 60 / slotsPerHour
	weekStart = calendarRangeDate(weekStart)
	for _, o := range occs {
		startLocal := o.StartsAt.In(weekStart.Location())
		col := dayDiff(weekStart, calendarRangeDate(startLocal))
		if col < 0 || col >= 7 {
			continue
		}
		if o.AllDay {
			allDay[col] = append(allDay[col], o)
			continue
		}
		endLocal := o.EndsAt.In(weekStart.Location())
		if !endLocal.After(startLocal) {
			endLocal = startLocal.Add(time.Duration(minutesPerSlot) * time.Minute)
		}
		startMin := startLocal.Hour()*60 + startLocal.Minute() - fromH*60
		endMin := endLocal.Hour()*60 + endLocal.Minute() - fromH*60
		dayMins := (toH - fromH) * 60
		if endMin <= 0 || startMin >= dayMins {
			continue
		}
		if startMin < 0 {
			startMin = 0
		}
		if endMin > dayMins {
			endMin = dayMins
		}
		s0 := startMin / minutesPerSlot
		s1 := (endMin + minutesPerSlot - 1) / minutesPerSlot
		if s1 <= s0 {
			s1 = s0 + 1
		}
		for s := s0; s < s1 && s < slotsTotal; s++ {
			if existing := timed[col][s]; existing.EventID != 0 {
				existing.OverflowCount++
				timed[col][s] = existing
				continue
			}
			slot := weekSlot{EventID: o.EventID, CalendarID: o.CalendarID, Status: o.Status, Continuation: s != s0}
			if !slot.Continuation {
				slot.Title = o.Title
			}
			timed[col][s] = slot
		}
	}
	return
}

func dayDiff(from, to time.Time) int {
	a := calendarRangeDate(from)
	b := calendarRangeDate(to)
	return int(b.Sub(a).Hours() / 24)
}

func (c Calendar) coloredCalendarBullet(calendarID int64) string {
	cal, ok := c.calendarByID(calendarID)
	color := ""
	if ok {
		color = strings.TrimSpace(cal.Color)
	}
	if color == "" {
		return "▌"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("▌")
}

func (c Calendar) calendarColorString(calendarID int64) string {
	cal, ok := c.calendarByID(calendarID)
	if !ok {
		return "63"
	}
	color := strings.TrimSpace(cal.Color)
	if color == "" {
		return "63"
	}
	return color
}

func weekColW(width, axisW int) (int, int) {
	usable := width - axisW - 7
	if usable < 7 {
		return 1, 0
	}
	dayW := usable / 7
	extra := usable - dayW*7
	return dayW, extra
}

func monthCellDims(width, bodyHeight int) (cellW, cellH, extraW, extraH int) {
	usableW := width - 6
	if usableW < 7 {
		cellW = 1
	} else {
		cellW = usableW / 7
		extraW = usableW - cellW*7
	}
	gridH := bodyHeight - 1 - 5
	if gridH < 6 {
		cellH = 1
	} else {
		cellH = gridH / 6
		extraH = gridH - cellH*6
	}
	if cellH < 2 {
		cellH = 2
	}
	return
}
