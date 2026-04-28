package screens

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

func (d Drive) selectedEntry() (drivestore.Entry, bool) {
	entries := d.visibleEntries()
	if len(entries) == 0 {
		return drivestore.Entry{}, false
	}
	return entries[d.clampedEntryIndex(entries)], true
}

func (d Drive) visibleEntries() []drivestore.Entry {
	filter := strings.ToLower(strings.TrimSpace(d.filter))
	if filter == "" {
		return d.entries
	}
	out := make([]drivestore.Entry, 0, len(d.entries))
	for _, entry := range d.entries {
		if strings.Contains(strings.ToLower(entry.Name), filter) {
			out = append(out, entry)
		}
	}
	return out
}

func (d Drive) renderEntryList(entries []drivestore.Entry, width, height int) string {
	d.ensureEntryList(entries)
	d.entryList.SetSize(width, height)
	return d.entryList.View()
}

func (d *Drive) ensureEntryList(entries []drivestore.Entry) {
	if len(d.entryList.Items()) == len(entries) {
		d.entryList.Select(d.clampedEntryIndex(entries))
		return
	}
	d.syncEntryList()
}

func (d *Drive) syncEntryList() {
	entries := d.visibleEntries()
	d.index = d.clampedEntryIndex(entries)
	d.entryList = newDriveList(entries, d.index, d.entryList.Width(), d.entryList.Height())
}

func (d *Drive) clampIndex() {
	d.index = d.clampedEntryIndex(d.visibleEntries())
}

func (d Drive) clampedEntryIndex(entries []drivestore.Entry) int {
	if d.index < 0 || len(entries) == 0 {
		return 0
	}
	if d.index >= len(entries) {
		return len(entries) - 1
	}
	return d.index
}

func newDriveList(entries []drivestore.Entry, selected, width, height int) list.Model {
	items := make([]list.Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, driveListItem{entry: entry})
	}
	return newSimpleList(items, driveListDelegate{}, selected, width, height)
}

type driveListDelegate struct{ simpleDelegate }

func (d driveListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	driveItem, ok := item.(driveListItem)
	if !ok {
		return
	}
	entry := driveItem.entry
	cursor := listCursor(index == m.Index())
	kind := "file"
	status := ""
	if entry.Kind == "folder" {
		kind = "dir "
	} else if !entry.Cached {
		status = " remote-only"
	}
	_, _ = io.WriteString(w, padRight(fmt.Sprintf("%s%s  %s%s", cursor, kind, entry.Name, status), m.Width()))
}
