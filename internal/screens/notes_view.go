package screens

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/emailtext"
)

func (n Notes) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if n.loading {
		return style.Render("Loading local notes cache...")
	}
	if n.err != nil {
		return style.Render(fmt.Sprintf("Notes cache error: %v\n\nRun `telex notes sync` or press S to populate Notes.", n.err))
	}
	var b strings.Builder
	b.WriteString("Notes")
	if n.flat {
		b.WriteString(" (all)")
	} else if crumb := n.breadcrumb(); crumb != "" {
		b.WriteString(" / " + crumb)
	}
	if n.flat {
		b.WriteString(fmt.Sprintf(" · %s", pluralNotes(len(n.rows))))
	} else if n.folder != nil && n.tree != nil && n.folder.ID != n.tree.ID && n.folder.NoteCount > 0 {
		b.WriteString(fmt.Sprintf(" · %s", pluralNotes(n.folder.NoteCount)))
	}
	b.WriteString(" · " + sortModeLabel(n.sortMode))
	b.WriteString("\n")
	if n.status != "" {
		b.WriteString(n.status + "\n")
	}
	if n.syncing {
		b.WriteString("Syncing remote Notes...\n")
	}
	if n.editing {
		b.WriteString("Filter: " + n.filter + "\n")
	}
	if n.confirm != "" {
		b.WriteString(n.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if n.detail != nil {
		header := b.String()
		headerLines := strings.Count(header, "\n")
		bodyHeight := max(1, height-headerLines)
		body := n.detailView(width, bodyHeight)
		b.WriteString(body)
		return style.Render(b.String())
	}
	rows := n.visibleRows()
	if len(rows) == 0 {
		b.WriteString("No cached notes found. Press S to sync or n to create a note.\n")
		return style.Render(b.String())
	}
	headerLines := strings.Count(b.String(), "\n")
	bodyHeight := max(1, height-headerLines)
	listWidth, previewWidth := notesPaneWidths(width)
	listCol := n.renderNotesList(rows, listWidth, bodyHeight)
	if previewWidth <= 0 {
		b.WriteString(listCol)
		return style.Render(b.String())
	}
	previewCol := n.renderNotesPreview(rows, previewWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(listWidth).Render(listCol),
		"  ",
		lipgloss.NewStyle().Width(previewWidth).Render(previewCol),
	)
	b.WriteString(body)
	return style.Render(b.String())
}

func notesPaneWidths(width int) (listWidth, previewWidth int) {
	if width < 64 {
		return width, 0
	}
	listWidth = width * 4 / 10
	if listWidth < 28 {
		listWidth = 28
	}
	if listWidth > 50 {
		listWidth = 50
	}
	const gap = 2
	previewWidth = width - listWidth - gap
	if previewWidth < 24 {
		return width, 0
	}
	return listWidth, previewWidth
}

func (n Notes) renderNotesList(rows []noteRow, width, height int) string {
	n.ensureNoteList(rows)
	n.rowList.SetSize(width, height)
	return n.rowList.View()
}

func (n Notes) renderNotesPreview(rows []noteRow, width int) string {
	if n.index < 0 || n.index >= len(rows) {
		return ""
	}
	row := rows[n.index]
	var b strings.Builder
	if row.Folder != nil {
		b.WriteString(row.Folder.Name + "\n")
		b.WriteString(pluralNotes(row.Folder.NoteCount))
		if row.Folder.ChildFolderCount > 0 {
			b.WriteString(fmt.Sprintf(" · %d subfolders", row.Folder.ChildFolderCount))
		}
		b.WriteString("\n")
		return b.String()
	}
	if row.Note == nil {
		return ""
	}
	b.WriteString(row.Note.Meta.Title + "\n")
	if updated := formatNotesRelative(row.Note.Meta.RemoteUpdatedAt); updated != "" {
		b.WriteString("Updated " + updated + "\n")
	}
	b.WriteString(strings.Repeat("─", width) + "\n")
	rendered, err := emailtext.RenderMarkdown(row.Note.Body, width)
	if err != nil {
		b.WriteString(row.Note.Body)
	} else {
		b.WriteString(rendered)
	}
	return b.String()
}

func formatNotesRow(row noteRow, selected bool, width int) string {
	cursor := listCursor(selected)
	glyph := "  "
	trailing := ""
	if row.Kind == "folder" {
		glyph = "▸ "
		if row.Folder != nil {
			trailing = strconv.Itoa(row.Folder.NoteCount)
		}
	} else if row.Note != nil {
		trailing = formatNotesRelative(row.Note.Meta.RemoteUpdatedAt)
	}
	const trailingCol = 12
	const prefixCols = 4
	titleSpace := width - prefixCols - 1 - trailingCol
	if titleSpace < 8 {
		return cursor + glyph + truncate(row.Name, max(0, width-prefixCols))
	}
	title := truncate(row.Name, titleSpace)
	return cursor + glyph + fmt.Sprintf("%-*s %*s", titleSpace, title, trailingCol, trailing)
}

func pluralNotes(n int) string {
	if n == 1 {
		return "1 note"
	}
	return fmt.Sprintf("%d notes", n)
}

func (n Notes) detailView(width, height int) string {
	if n.detail == nil {
		return ""
	}
	bodyWidth := notesBodyWidth(width)
	var head strings.Builder
	head.WriteString(n.detail.Meta.Title + "\n")
	meta := n.detailContext()
	if updated := formatNotesRelative(n.detail.Meta.RemoteUpdatedAt); updated != "" {
		if meta != "" {
			meta += " · Updated " + updated
		} else {
			meta = "Updated " + updated
		}
	}
	if meta != "" {
		head.WriteString(meta + "\n")
	}
	head.WriteString(strings.Repeat("─", bodyWidth) + "\n")

	rendered, err := emailtext.RenderMarkdown(n.detail.Body, bodyWidth)
	body := rendered
	if err != nil {
		body = fmt.Sprintf("Markdown render error: %v", err)
	}
	bodyLines := strings.Split(strings.TrimRight(body, "\n"), "\n")

	const reservedFooter = 2
	headLines := strings.Count(head.String(), "\n")
	visibleBodyHeight := max(1, height-headLines-reservedFooter)
	scroll := n.detailScroll
	if scroll < 0 {
		scroll = 0
	}
	maxScroll := max(0, len(bodyLines)-visibleBodyHeight)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := min(len(bodyLines), scroll+visibleBodyHeight)
	visible := bodyLines[scroll:end]

	var b strings.Builder
	b.WriteString(head.String())
	b.WriteString(strings.Join(visible, "\n"))
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("─", bodyWidth) + "\n")
	b.WriteString(detailFooterHint(scroll, len(bodyLines), visibleBodyHeight) + "\n")
	return b.String()
}

func detailFooterHint(scroll, total, visible int) string {
	hint := "[esc] back  [e] edit"
	if total > visible {
		hint += "  [j/k] scroll"
		hint += fmt.Sprintf("  %d-%d/%d", scroll+1, min(scroll+visible, total), total)
	}
	return hint
}

func (n Notes) detailContext() string {
	if n.detail == nil || n.tree == nil {
		return ""
	}
	folderID := n.detail.Meta.FolderID
	if folderID == 0 || folderID == n.tree.ID {
		return "Notes"
	}
	paths := notesFolderPath(n.tree, folderID, nil)
	if len(paths) == 0 {
		return "Notes"
	}
	if len(paths) > 1 {
		paths = paths[1:]
	}
	return "Notes / " + strings.Join(paths, " / ")
}

func notesBodyWidth(width int) int {
	if width < 24 {
		return 20
	}
	return width - 4
}

func formatNotesID(id int64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}

func formatNotesRelative(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	delta := now.Sub(t)
	if delta < 0 {
		delta = 0
	}
	switch {
	case delta < time.Minute:
		return "just now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta/time.Minute))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta/time.Hour))
	case delta < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(delta/(24*time.Hour)))
	}
	if t.Year() == now.Year() {
		return t.Format("Jan 02")
	}
	return t.Format("Jan 02 2006")
}
