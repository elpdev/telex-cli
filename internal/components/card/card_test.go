package card

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

func TestCardViewFitsWidthWithoutWrap(t *testing.T) {
	rows := []Row{
		{Left: "Nunya — Testing 123", Right: "21h ago", Accent: true},
		{Left: "Uber Receipts — Your Friday evening order with Uber Eats", Right: "22h ago"},
	}
	for _, w := range []int{40, 60, 80, 99, 100, 120, 160, 200} {
		c := New(theme.Phosphor()).
			WithTitle("MAIL").
			WithKeyHint("m / 1").
			WithCounts("160 unread", "1 drafts").
			WithRows(rows).
			WithWidth(w)
		out := c.View()
		lines := strings.Split(out, "\n")

		// Total rendered width per line must equal exactly w (lipgloss pads to Width).
		for i, line := range lines {
			lw := lipgloss.Width(line)
			if lw != w {
				t.Errorf("width=%d: line %d width=%d != %d: %q", w, i, lw, w, line)
			}
		}

		// Expected content lines: top border + title + counts + blank + N rows + bottom border.
		// The exact count is brittle, so just verify "no soft-wrap": the title+counts+rows
		// should each be a single line, not split. We check by counting rendered line count
		// against the count we'd get with no wrap.
		expectedContentLines := 1 + 1 + 1 + len(rows) // title, counts, blank, rows
		expectedTotal := expectedContentLines + 2     // top + bottom border
		if len(lines) != expectedTotal {
			t.Errorf("width=%d: got %d rendered lines, expected %d (wrap detected)", w, len(lines), expectedTotal)
		}
	}
}

func TestCardFocusedTogglesBorder(t *testing.T) {
	base := New(theme.Phosphor()).WithTitle("X").WithWidth(40)
	unfocused := base.View()
	focused := base.Focus().View()
	if unfocused == focused {
		t.Error("expected focused view to differ from unfocused (border color should change)")
	}
	if base.Focused() {
		t.Error("base should not be focused")
	}
	if !base.Focus().Focused() {
		t.Error("Focus() should mark model as focused")
	}
	if base.Focus().Blur().Focused() {
		t.Error("Blur() should clear focus")
	}
}

func TestCardErrorReplacesCount(t *testing.T) {
	c := New(theme.Phosphor()).WithTitle("X").WithError("boom").WithWidth(40)
	out := c.View()
	if !strings.Contains(out, "boom") {
		t.Errorf("expected error text in view, got: %q", out)
	}
}

func TestCardEmptyStateRenders(t *testing.T) {
	c := New(theme.Phosphor()).WithTitle("X").WithEmpty("nothing here").WithWidth(40)
	out := c.View()
	if !strings.Contains(out, "nothing here") {
		t.Errorf("expected empty hint in view, got: %q", out)
	}
}

func TestTruncateLineKeepsShort(t *testing.T) {
	if got := TruncateLine("hi", 10); got != "hi" {
		t.Errorf("TruncateLine kept short input wrong: %q", got)
	}
	if got := TruncateLine("abcdef", 4); got != "abc…" {
		t.Errorf("TruncateLine output unexpected: %q", got)
	}
	if got := TruncateLine("abcdef", 0); got != "" {
		t.Errorf("TruncateLine with 0 should be empty: %q", got)
	}
}
