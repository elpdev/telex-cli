package screens

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

func (d Drive) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if d.loading {
		return style.Render("Loading local drive mirror...")
	}
	if d.err != nil {
		return style.Render(fmt.Sprintf("Drive cache error: %v\n\nRun `telex drive sync` to populate the local drive mirror.", d.err))
	}
	if d.pickerOpen {
		return style.Render(d.picker.View(width, height))
	}
	var b strings.Builder
	b.WriteString("Drive / " + strings.Join(d.breadcrumbs, " / ") + "\n")
	if d.status != "" {
		b.WriteString(d.status + "\n")
	}
	if d.filtering {
		b.WriteString("Filter: " + d.filter + "\n")
	}
	if d.prompt != drivePromptNone {
		b.WriteString(d.promptLabel() + d.promptInput + "\n")
	}
	if d.confirm != "" {
		b.WriteString(d.confirm + " [y/N]\n")
	}
	if d.syncing {
		b.WriteString("Syncing remote Drive...\n")
	}
	b.WriteString("\n")
	entries := d.visibleEntries()
	if len(entries) == 0 {
		b.WriteString("No mirrored Drive items found. Press S to sync.\n")
		return style.Render(b.String())
	}
	headerLines := strings.Count(b.String(), "\n")
	b.WriteString(d.renderEntryList(entries, width, max(1, height-headerLines)))
	if d.details {
		b.WriteString("\n" + d.detailsView())
	}
	return style.Render(b.String())
}

func (d Drive) promptLabel() string {
	if d.prompt == drivePromptNewFolder {
		return "New folder: "
	}
	return "Rename: "
}

func (d Drive) detailsView() string {
	entry, ok := d.selectedEntry()
	if !ok {
		return "Details: no selection\n"
	}
	var b strings.Builder
	b.WriteString("Details\n")
	b.WriteString(fmt.Sprintf("Kind: %s\nName: %s\nLocal path: %s\n", entry.Kind, entry.Name, entry.Path))
	if entry.Folder != nil {
		b.WriteString(fmt.Sprintf("Remote ID: %d\nSynced at: %s\n", entry.Folder.RemoteID, entry.Folder.SyncedAt.Format(time.RFC3339)))
	}
	if entry.File != nil {
		cached := "remote-only"
		if entry.Cached {
			cached = "cached"
		}
		b.WriteString(fmt.Sprintf("Remote ID: %d\nMIME type: %s\nByte size: %d\nCached state: %s\nSynced at: %s\nDownload URL: %t\n", entry.File.RemoteID, entry.File.MIMEType, entry.File.ByteSize, cached, entry.File.SyncedAt.Format(time.RFC3339), entry.File.DownloadURL != ""))
	}
	return b.String()
}

func (d Drive) pathParts() []string {
	rel, err := filepath.Rel(d.store.DriveRoot(), d.path)
	if err != nil || rel == "." {
		return nil
	}
	return strings.Split(rel, string(filepath.Separator))
}

func maxDriveIndex(length int) int {
	if length <= 0 {
		return 0
	}
	return length - 1
}
