package screens

import (
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/calendarstore"
)

func (c *Calendar) applyAgendaFilter() {
	source := c.agendaSourceItems()
	if !c.filter.active() {
		c.items = append([]calendarstore.OccurrenceMeta(nil), source...)
		c.clampIndex()
		return
	}
	items := make([]calendarstore.OccurrenceMeta, 0, len(source))
	for _, item := range source {
		if c.occurrenceMatchesFilter(item) {
			items = append(items, item)
		}
	}
	c.items = items
	c.index = 0
	c.clampIndex()
}

func (c Calendar) agendaSourceItems() []calendarstore.OccurrenceMeta {
	if c.allItems != nil {
		return c.allItems
	}
	return c.items
}

func (c Calendar) occurrenceMatchesFilter(item calendarstore.OccurrenceMeta) bool {
	if !calendarFilterMatch(c.filter.Status, item.Status) {
		return false
	}
	if !calendarFilterMatch(c.filter.Calendar, c.calendarFilterValue(item.CalendarID)) {
		return false
	}
	if !calendarFilterMatch(c.filter.Source, c.occurrenceSource(item)) {
		return false
	}
	if c.filter.Text != "" && !calendarTextMatch(item, c.filter.Text) {
		return false
	}
	return true
}

func (c Calendar) calendarFilterValue(calendarID int64) string {
	values := []string{strconv.FormatInt(calendarID, 10)}
	for _, cal := range c.calendars {
		if cal.RemoteID == calendarID {
			values = append(values, cal.Name)
			break
		}
	}
	return strings.Join(values, " ")
}

func (c Calendar) occurrenceSource(item calendarstore.OccurrenceMeta) string {
	if strings.TrimSpace(item.Source) != "" {
		return item.Source
	}
	event, err := c.store.ReadEvent(item.EventID)
	if err != nil || event == nil {
		return ""
	}
	return event.Meta.Source
}

func calendarFilterMatch(needle, haystack string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needle)
}

func calendarTextMatch(item calendarstore.OccurrenceMeta, text string) bool {
	haystack := strings.ToLower(item.Title + " " + item.Location)
	for _, term := range strings.Fields(strings.ToLower(strings.TrimSpace(text))) {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func parseCalendarAgendaFilter(input string) calendarAgendaFilter {
	filter := calendarAgendaFilter{}
	text := []string{}
	for _, token := range strings.Fields(input) {
		key, value, ok := strings.Cut(token, ":")
		if !ok || strings.TrimSpace(value) == "" {
			text = append(text, token)
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "calendar", "cal":
			filter.Calendar = strings.TrimSpace(value)
		case "status":
			filter.Status = strings.TrimSpace(value)
		case "source", "src":
			filter.Source = strings.TrimSpace(value)
		default:
			text = append(text, token)
		}
	}
	filter.Text = strings.Join(text, " ")
	return filter
}

func (f calendarAgendaFilter) active() bool {
	return strings.TrimSpace(f.Calendar) != "" || strings.TrimSpace(f.Status) != "" || strings.TrimSpace(f.Source) != "" || strings.TrimSpace(f.Text) != ""
}

func (f calendarAgendaFilter) summary() string {
	parts := []string{}
	if strings.TrimSpace(f.Calendar) != "" {
		parts = append(parts, "calendar="+strings.TrimSpace(f.Calendar))
	}
	if strings.TrimSpace(f.Status) != "" {
		parts = append(parts, "status="+strings.TrimSpace(f.Status))
	}
	if strings.TrimSpace(f.Source) != "" {
		parts = append(parts, "source="+strings.TrimSpace(f.Source))
	}
	if strings.TrimSpace(f.Text) != "" {
		parts = append(parts, "text=\""+strings.TrimSpace(f.Text)+"\"")
	}
	return strings.Join(parts, " ")
}

func (f calendarAgendaFilter) inputString() string {
	parts := []string{}
	if strings.TrimSpace(f.Calendar) != "" {
		parts = append(parts, "calendar:"+strings.TrimSpace(f.Calendar))
	}
	if strings.TrimSpace(f.Status) != "" {
		parts = append(parts, "status:"+strings.TrimSpace(f.Status))
	}
	if strings.TrimSpace(f.Source) != "" {
		parts = append(parts, "source:"+strings.TrimSpace(f.Source))
	}
	if strings.TrimSpace(f.Text) != "" {
		parts = append(parts, strings.TrimSpace(f.Text))
	}
	return strings.Join(parts, " ")
}
