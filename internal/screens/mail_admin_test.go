package screens

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mail"
)

func TestMailAdminLoadsDomainsAndInboxes(t *testing.T) {
	screen := NewMailAdmin(func(context.Context) ([]mail.Domain, []mail.Inbox, error) {
		return []mail.Domain{{ID: 1, Name: "agent.test", Active: true}}, []mail.Inbox{{ID: 2, DomainID: 1, Address: "support@agent.test", LocalPart: "support", Active: true}}, nil
	})
	updated, _ := screen.Update(mailAdminLoadedMsg{domains: []mail.Domain{{ID: 1, Name: "agent.test", Active: true}}, inboxes: []mail.Inbox{{ID: 2, DomainID: 1, Address: "support@agent.test", LocalPart: "support", Active: true}}})
	screen = updated.(MailAdmin)

	view := screen.View(100, 24)
	if !strings.Contains(view, "agent.test") || !strings.Contains(view, "support@agent.test") {
		t.Fatalf("view = %q", view)
	}
}

func TestMailAdminSwitchesFocus(t *testing.T) {
	screen := NewMailAdmin(nil)
	updated, _ := screen.Update(mailAdminLoadedMsg{domains: []mail.Domain{{ID: 1, Name: "agent.test"}}, inboxes: []mail.Inbox{{ID: 2, DomainID: 1, Address: "support@agent.test"}}})
	screen = updated.(MailAdmin)

	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	screen = updated.(MailAdmin)
	if screen.focus != mailAdminFocusInboxes {
		t.Fatalf("focus = %v, want inboxes", screen.focus)
	}
}

func TestMailAdminUsesListNavigation(t *testing.T) {
	screen := NewMailAdmin(nil)
	updated, _ := screen.Update(mailAdminLoadedMsg{
		domains: []mail.Domain{{ID: 1, Name: "agent.test"}, {ID: 2, Name: "example.test", Active: true}},
		inboxes: []mail.Inbox{
			{ID: 3, DomainID: 2, Address: "support@example.test", Active: true},
			{ID: 4, DomainID: 2, Address: "sales@example.test", Active: true},
		},
	})
	screen = updated.(MailAdmin)

	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
	if cmd != nil {
		t.Fatal("expected no command")
	}
	screen = updated.(MailAdmin)
	if screen.domainIndex != 1 || screen.inboxIndex != 0 {
		t.Fatalf("domainIndex = %d inboxIndex = %d", screen.domainIndex, screen.inboxIndex)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	screen = updated.(MailAdmin)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
	screen = updated.(MailAdmin)
	if screen.inboxIndex != 1 {
		t.Fatalf("inboxIndex = %d, want 1", screen.inboxIndex)
	}
	inbox, ok := screen.selectedInbox()
	if !ok || inbox.Address != "sales@example.test" {
		t.Fatalf("inbox = %#v ok = %v", inbox, ok)
	}
	view := screen.View(100, 24)
	if !strings.Contains(view, "> 2  example.test") || !strings.Contains(view, "> 4  sales@example.test") {
		t.Fatalf("view missing selected domain/inbox:\n%s", view)
	}
}
