package screens

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/theme"
)

func TestHomeViewRendersAllFourModuleCards(t *testing.T) {
	root := t.TempDir()
	home := NewHome(
		mailstore.New(root),
		calendarstore.New(root),
		notestore.New(root),
		drivestore.New(root),
		theme.Phosphor(),
		nil,
	)

	loaded, _ := home.Update(homeLoadedMsg{summary: collectHomeSummary(home.mail, home.calendar, home.notes, home.drive)})
	out := loaded.View(120, 40)

	for _, label := range []string{"MAIL", "CALENDAR", "NOTES", "DRIVE"} {
		if !strings.Contains(out, label) {
			t.Errorf("dashboard view missing module label %q", label)
		}
	}
	if !strings.Contains(out, "No data cached yet") {
		t.Error("expected empty-state hint in header for fresh stores")
	}
}

func TestHomeViewHandlesVariousWidths(t *testing.T) {
	root := t.TempDir()
	home := NewHome(
		mailstore.New(root),
		calendarstore.New(root),
		notestore.New(root),
		drivestore.New(root),
		theme.Phosphor(),
		nil,
	)
	loaded, _ := home.Update(homeLoadedMsg{summary: collectHomeSummary(home.mail, home.calendar, home.notes, home.drive)})

	for _, w := range []int{40, 80, 99, 100, 120, 200} {
		out := loaded.View(w, 40)
		if out == "" {
			t.Errorf("View(%d, 40) returned empty string", w)
		}
	}
}

// TestLoadedHomeFitsInsideThemeMainWrapper guards against the dashboard
// wrap bug: app/view.go renders the active screen at the inner content
// width and then wraps it in theme.Main. If theme.Main is given the inner
// width as its total width, lipgloss soft-wraps every line by the frame
// size, splitting key hints like "(c / 2)" and right-aligned timestamps
// onto a second line.
func TestLoadedHomeFitsInsideThemeMainWrapper(t *testing.T) {
	root := t.TempDir()
	home := NewHome(
		mailstore.New(root),
		calendarstore.New(root),
		notestore.New(root),
		drivestore.New(root),
		theme.Phosphor(),
		nil,
	)
	loaded, _ := home.Update(homeLoadedMsg{summary: collectHomeSummary(home.mail, home.calendar, home.notes, home.drive)})

	th := theme.Phosphor()
	frameW, frameH := th.Main.GetFrameSize()

	for _, totalW := range []int{120, 160, 200} {
		innerW := totalW - frameW
		innerH := 30

		body := loaded.View(innerW, innerH)
		wrapped := th.Main.Width(totalW).Height(innerH + frameH).Render(body)

		lines := strings.Split(wrapped, "\n")
		for i, line := range lines {
			if lw := lipgloss.Width(line); lw != totalW {
				t.Errorf("totalW=%d: line %d width=%d (want %d): %q", totalW, i, lw, totalW, line)
			}
		}
		if len(lines) != innerH+frameH {
			t.Errorf("totalW=%d: rendered %d lines, want %d (soft-wrap inside theme.Main)", totalW, len(lines), innerH+frameH)
		}
	}
}

func TestHomeViewWhileLoading(t *testing.T) {
	home := NewHome(
		mailstore.New(t.TempDir()),
		calendarstore.New(t.TempDir()),
		notestore.New(t.TempDir()),
		drivestore.New(t.TempDir()),
		theme.Phosphor(),
		nil,
	)
	out := home.View(120, 40)
	if !strings.Contains(out, "Loading") {
		t.Errorf("expected loading message before homeLoadedMsg, got: %q", out)
	}
}
