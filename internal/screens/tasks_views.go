package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/emailtext"
)

func (t Tasks) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if t.loading {
		return style.Render("Loading local tasks cache...")
	}
	if t.err != nil {
		return style.Render(fmt.Sprintf("Tasks cache error: %v\n\nRun `telex tasks sync` or press S to populate Tasks.", t.err))
	}
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true)
	hintStyle := lipgloss.NewStyle().Faint(true)
	if t.project != nil {
		b.WriteString(titleStyle.Render(t.project.Meta.Name))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("Projects › " + t.project.Meta.Name + "   esc/p: back to projects"))
		b.WriteString("\n")
	} else {
		b.WriteString(titleStyle.Render("Projects"))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("All projects   n: new project · S: sync"))
		b.WriteString("\n")
	}
	if legend := t.actionLegend(); legend != "" {
		b.WriteString(hintStyle.Render(legend))
		b.WriteString("\n")
	}
	if t.status != "" {
		b.WriteString(t.status + "\n")
	}
	if t.syncing {
		b.WriteString("Syncing remote Tasks...\n")
	}
	if t.filtering {
		b.WriteString("Filter: " + t.filter + "\n")
	}
	if t.picking {
		line := "Move to column: " + t.picker
		if cols := t.columnNamesString(); cols != "" {
			line += "   (" + cols + ")"
		}
		b.WriteString(line + "\n")
	}
	if t.confirm != "" {
		b.WriteString(t.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if t.detail != nil {
		headerLines := strings.Count(b.String(), "\n")
		b.WriteString(t.detailView(width, max(1, height-headerLines)))
		return style.Render(b.String())
	}
	rows := t.visibleRows()
	if len(rows) == 0 {
		b.WriteString("No cached task projects found. Press S to sync or n to create a project.\n")
		return style.Render(b.String())
	}
	headerLines := strings.Count(b.String(), "\n")
	bodyHeight := max(1, height-headerLines)
	listWidth, previewWidth := tasksPaneWidths(width)
	listCol := t.renderList(rows, listWidth, bodyHeight)
	if previewWidth <= 0 {
		b.WriteString(listCol)
		return style.Render(b.String())
	}
	previewCol := t.renderPreview(rows, previewWidth)
	separator := tasksPaneSeparator(bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(listWidth).Render(listCol), separator, lipgloss.NewStyle().Width(previewWidth).Render(previewCol))
	b.WriteString(body)
	return style.Render(b.String())
}

func tasksPaneSeparator(height int) string {
	if height < 1 {
		height = 1
	}
	bar := lipgloss.NewStyle().Faint(true).Render("│")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = " " + bar + " "
	}
	return strings.Join(lines, "\n")
}

func tasksPaneWidths(width int) (int, int) {
	if width < 64 {
		return width, 0
	}
	listWidth := width * 4 / 10
	if listWidth < 28 {
		listWidth = 28
	}
	if listWidth > 50 {
		listWidth = 50
	}
	previewWidth := width - listWidth - 2
	if previewWidth < 24 {
		return width, 0
	}
	return listWidth, previewWidth
}

func (t Tasks) renderList(rows []taskRow, width, height int) string {
	t.ensureList(rows)
	t.rowList.SetSize(width, height)
	return t.rowList.View()
}

func (t Tasks) renderPreview(rows []taskRow, width int) string {
	if t.index < 0 || t.index >= len(rows) {
		return ""
	}
	row := rows[t.index]
	var b strings.Builder
	switch {
	case row.Project != nil:
		b.WriteString(row.Project.Meta.Name + "\n")
		b.WriteString(fmt.Sprintf("Project %d\n", row.Project.Meta.RemoteID))
	case row.Card != nil:
		b.WriteString(row.Card.Meta.Title + "\n")
		meta := ""
		if updated := formatNotesRelative(row.Card.Meta.RemoteUpdatedAt); updated != "" {
			meta = "Updated " + updated
		}
		columnName := "Unlinked"
		if row.Column != nil {
			columnName = row.Column.Name
		}
		if meta != "" {
			meta += " · in " + columnName
		} else {
			meta = "in " + columnName
		}
		b.WriteString(meta + "\n")
		b.WriteString(strings.Repeat("─", width) + "\n")
		rendered, err := emailtext.RenderMarkdown(row.Card.Body, width)
		if err != nil {
			b.WriteString(row.Card.Body)
		} else {
			b.WriteString(rendered)
		}
	case row.Column != nil:
		b.WriteString(row.Column.Name + "\n")
		b.WriteString(fmt.Sprintf("%d linked card(s)\n", len(row.Column.Cards)))
	case row.Missing:
		b.WriteString(row.Name + "\nMissing linked card\n")
	}
	return b.String()
}

func (t Tasks) Title() string { return "Projects" }

func (t Tasks) actionLegend() string {
	row, ok := t.selectedRow()
	if !ok {
		if t.project == nil {
			return "enter: open · n: new project · S: sync"
		}
		return "n: new card · S: sync · /: filter"
	}
	switch {
	case row.Project != nil:
		return "enter: open · n: new project · S: sync"
	case row.Card != nil:
		return "enter: open · e: edit · x: delete · </>: move column · m: move to…"
	case row.Column != nil:
		return "n: new card · S: sync · /: filter"
	}
	return ""
}

func (t Tasks) columnNamesString() string {
	if t.board == nil {
		return ""
	}
	names := make([]string, 0, len(t.board.Columns))
	for _, col := range t.board.Columns {
		names = append(names, col.Name)
	}
	return strings.Join(names, " · ")
}

func (t Tasks) detailView(width, height int) string {
	if t.detail == nil {
		return ""
	}
	bodyWidth := notesBodyWidth(width)
	var head strings.Builder
	head.WriteString(t.detail.Meta.Title + "\n")
	if updated := formatNotesRelative(t.detail.Meta.RemoteUpdatedAt); updated != "" {
		head.WriteString("Updated " + updated + "\n")
	}
	head.WriteString(strings.Repeat("─", bodyWidth) + "\n")
	rendered, err := emailtext.RenderMarkdown(t.detail.Body, bodyWidth)
	body := rendered
	if err != nil {
		body = fmt.Sprintf("Markdown render error: %v", err)
	}
	bodyLines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	visibleBodyHeight := max(1, height-strings.Count(head.String(), "\n")-2)
	scroll := t.detailScroll
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
