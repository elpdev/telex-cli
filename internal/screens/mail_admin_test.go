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
