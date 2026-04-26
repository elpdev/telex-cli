package commands

import (
	"strings"
	"testing"

	"github.com/elpdev/telex-cli/internal/theme"
)

func TestPaletteViewRendersMailSubgroupHeadings(t *testing.T) {
	r := NewRegistry()
	r.Register(Command{ID: "go-inbox", Module: ModuleMail, Group: GroupNav, Title: "Open Inbox", Pinned: true})
	r.Register(Command{ID: "drafts-compose", Module: ModuleMail, Group: GroupDrafts, Title: "Compose draft"})
	r.Register(Command{ID: "messages-reply", Module: ModuleMail, Group: GroupMessages, Title: "Reply"})
	r.Register(Command{ID: "messages-block", Module: ModuleMail, Group: GroupPolicy, Title: "Block sender"})
	r.Register(Command{ID: "domains-new", Module: ModuleMail, Group: GroupAdmin, Title: "New domain"})
	r.Register(Command{ID: "mail-sync", Module: ModuleMail, Title: "Sync mailbox"}) // ungrouped

	p := NewPaletteModel(r, theme.BuiltIns())
	p.SetSize(140, 40)
	p.Reset("phosphor", Context{ActiveScreen: ModuleMail, ActiveModule: ModuleMail})

	out := p.View(theme.BuiltIns()[0])

	mustContain := []string{"MAIL", "NAV", "DRAFTS", "MESSAGES", "POLICY", "ADMIN", "Sync mailbox"}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("expected palette view to contain %q, got:\n%s", s, out)
		}
	}

	// Ungrouped command (Sync mailbox) should appear before subgroup headings.
	syncIdx := strings.Index(out, "Sync mailbox")
	navIdx := strings.Index(out, "NAV")
	if syncIdx == -1 || navIdx == -1 || syncIdx > navIdx {
		t.Errorf("expected ungrouped Sync to appear before NAV heading; sync=%d nav=%d", syncIdx, navIdx)
	}
}
