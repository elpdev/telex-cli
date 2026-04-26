package card

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

type Row struct {
	Left   string
	Right  string
	Accent bool
}

type Model struct {
	theme   theme.Theme
	title   string
	keyHint string
	counts  []string
	rows    []Row
	width   int
	focused bool
	errText string
	empty   string
}

func New(t theme.Theme) Model { return Model{theme: t} }

func (m Model) WithTheme(t theme.Theme) Model      { m.theme = t; return m }
func (m Model) WithTitle(s string) Model           { m.title = s; return m }
func (m Model) WithKeyHint(s string) Model         { m.keyHint = s; return m }
func (m Model) WithCounts(counts ...string) Model  { m.counts = counts; return m }
func (m Model) WithRows(rows []Row) Model          { m.rows = rows; return m }
func (m Model) WithWidth(w int) Model              { m.width = w; return m }
func (m Model) WithError(s string) Model           { m.errText = s; return m }
func (m Model) WithEmpty(s string) Model           { m.empty = s; return m }

func (m Model) Focused() bool { return m.focused }
func (m Model) Focus() Model  { m.focused = true; return m }
func (m Model) Blur() Model   { m.focused = false; return m }

func (m Model) Init() tea.Cmd                       { return nil }
func (m Model) Update(_ tea.Msg) (Model, tea.Cmd)   { return m, nil }

func (m Model) View() string {
	frame := m.frameStyle()
	fw, _ := frame.GetFrameSize()
	inner := m.width - fw
	if inner < 4 {
		inner = 4
	}

	var b strings.Builder
	b.WriteString(m.titleRow(inner))
	b.WriteString("\n")
	b.WriteString(m.countRow(inner))
	b.WriteString("\n\n")

	if m.errText == "" {
		if len(m.rows) == 0 && m.empty != "" {
			b.WriteString(m.theme.Muted.Render(TruncateLine(m.empty, inner)))
			b.WriteString("\n")
		}
		for _, r := range m.rows {
			b.WriteString(m.renderRow(r, inner))
			b.WriteString("\n")
		}
	}

	return frame.Width(m.width).Render(strings.TrimRight(b.String(), "\n"))
}

func (m Model) frameStyle() lipgloss.Style {
	border := m.theme.Border.GetForeground()
	if m.focused {
		border = m.theme.Title.GetForeground()
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}

func (m Model) titleRow(width int) string {
	title := m.theme.Title.Render(m.title)
	hint := ""
	hintW := 0
	if m.keyHint != "" {
		hint = m.theme.HeaderAccent.Render("(" + m.keyHint + ")")
		hintW = lipgloss.Width(hint)
	}
	titleW := lipgloss.Width(title)
	if titleW+hintW >= width {
		return TruncateLine(m.title, width)
	}
	pad := width - titleW - hintW
	if pad < 1 {
		pad = 1
	}
	return title + strings.Repeat(" ", pad) + hint
}

func (m Model) countRow(width int) string {
	if m.errText != "" {
		return m.theme.Warn.Render(TruncateLine("⚠ "+m.errText, width))
	}
	if len(m.counts) == 0 {
		return ""
	}
	line := strings.Join(m.counts, "  ")
	return m.theme.HeaderAccent.Render(TruncateLine(line, width))
}

func (m Model) renderRow(r Row, width int) string {
	bullet := m.theme.Muted.Render("• ")
	if r.Accent {
		bullet = m.theme.HeaderAccent.Render("• ")
	}
	right := m.theme.Muted.Render(r.Right)
	bulletW := lipgloss.Width(bullet)
	rightW := lipgloss.Width(right)
	avail := width - bulletW - rightW - 1
	if avail < 4 {
		avail = 4
	}
	leftRaw := TruncateLine(r.Left, avail)
	leftW := lipgloss.Width(leftRaw)
	pad := width - bulletW - leftW - rightW
	if pad < 1 {
		pad = 1
	}
	leftStyled := m.theme.Text.Render(leftRaw)
	if r.Accent {
		leftStyled = m.theme.Title.Render(leftRaw)
	}
	return bullet + leftStyled + strings.Repeat(" ", pad) + right
}

func TruncateLine(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > max {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
