package screens

import (
	"strings"
	"testing"

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
