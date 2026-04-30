package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func (m *Mail) syncViewport(v *viewport.Model, width, height int, content string) {
	v.SetWidth(width)
	v.SetHeight(height)
	v.SetContent(strings.TrimRight(content, "\n"))
}

func viewportRangeHint(v viewport.Model) string {
	total := v.TotalLineCount()
	visible := v.VisibleLineCount()
	if total <= visible {
		return ""
	}
	offset := v.YOffset()
	end := min(total, offset+visible)
	return fmt.Sprintf("%d-%d/%d", offset+1, end, total)
}

func paginatedRange(total, selected, pageSize int) (start, end, page, pages int) {
	if total <= 0 {
		return 0, 0, 0, 0
	}
	pageSize = max(1, pageSize)
	selected = clampIndex(selected, total)
	pages = (total + pageSize - 1) / pageSize
	page = selected/pageSize + 1
	start = (page - 1) * pageSize
	end = min(total, start+pageSize)
	return start, end, page, pages
}

func truncate(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width < 3 {
		return value[:width]
	}
	return value[:width-1] + "…"
}

func formatBytes(size int64) string {
	switch {
	case size >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	case size > 0:
		return fmt.Sprintf("%d B", size)
	default:
		return ""
	}
}

func mailHeader(title string, segments ...string) string {
	out := title
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		out += " · " + seg
	}
	return out
}

func mailFooterHint(parts ...string) string {
	cleaned := parts[:0]
	for _, p := range parts {
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, "  ")
}

func mailSeparator(width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat("─", width)
}

type mailColumnLayout struct {
	sender, subject, date, label int
}

func mailColumns(width int) mailColumnLayout {
	const (
		cursorW = 2
		glyphsW = 2
		dateW   = 12
		gaps    = 4
	)
	overhead := cursorW + glyphsW + dateW + gaps
	switch {
	case width <= 80:
		sender := 14
		subject := max(8, width-overhead-sender)
		return mailColumnLayout{sender: sender, subject: subject, date: dateW, label: 0}
	case width <= 120:
		sender := 18
		label := 14
		subject := max(8, width-overhead-sender-1-label)
		return mailColumnLayout{sender: sender, subject: subject, date: dateW, label: label}
	default:
		sender := 24
		label := 30
		subject := max(8, width-overhead-sender-1-label)
		return mailColumnLayout{sender: sender, subject: subject, date: dateW, label: label}
	}
}

func formatMailRow(message mailstore.CachedMessage, selected bool, layout mailColumnLayout) string {
	cursor := listCursor(selected)
	unread := " "
	if !message.Meta.Read {
		unread = "●"
	}
	star := " "
	if message.Meta.Starred {
		star = "★"
	}
	sender := truncate(message.Meta.FromAddress, layout.sender)
	subject := truncate(message.Meta.Subject, layout.subject)
	date := message.Meta.ReceivedAt.Format("Jan 02 15:04")
	row := fmt.Sprintf("%s%s%s %-*s %-*s %*s",
		cursor, unread, star,
		layout.sender, sender,
		layout.subject, subject,
		layout.date, date,
	)
	if layout.label > 0 {
		labels := strings.Join(cachedLabelNames(message.Meta.Labels), ",")
		row += " " + fmt.Sprintf("%-*s", layout.label, truncate(labels, layout.label))
	}
	return row
}

func formatConversationRow(entry ConversationEntry, selected bool, layout mailColumnLayout) string {
	cursor := listCursor(selected)
	kind := conversationKindLabel(entry.Kind)
	sender := truncate(entry.Sender, layout.sender)
	subject := truncate(entry.Subject, layout.subject)
	date := entry.OccurredAt.Format("Jan 02 15:04")
	return fmt.Sprintf("%s%s %-*s %-*s %*s",
		cursor, kind,
		layout.sender, sender,
		layout.subject, subject,
		layout.date, date,
	)
}
