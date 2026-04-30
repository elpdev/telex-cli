package screens

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/telex-cli/internal/contactstore"
)

func (c Contacts) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if c.loading {
		return style.Render("Loading local contacts cache...")
	}
	if c.err != nil {
		return style.Render(fmt.Sprintf("Contacts cache error: %v\n\nRun `telex contacts sync` or press S to populate Contacts.", c.err))
	}
	var b strings.Builder
	b.WriteString("Contacts")
	b.WriteString(fmt.Sprintf(" · %d cached", len(c.contacts)))
	if c.filter != "" {
		b.WriteString(" · filter: " + c.filter)
	}
	b.WriteString("\n")
	if c.status != "" {
		b.WriteString(c.status + "\n")
	}
	if c.syncing {
		b.WriteString("Syncing remote Contacts...\n")
	}
	if c.editing {
		b.WriteString("Filter: " + c.filter + "\n")
	}
	if c.confirm != "" {
		b.WriteString(c.confirm + " [y/N]\n")
	}
	b.WriteString("\n")
	if c.detail != nil {
		headerLines := strings.Count(b.String(), "\n")
		b.WriteString(c.detailView(width, max(1, height-headerLines)))
		return style.Render(b.String())
	}
	visible := c.visibleContacts()
	if len(visible) == 0 {
		b.WriteString("No cached contacts found. Press S to sync.\n")
		return style.Render(b.String())
	}
	headerLines := strings.Count(b.String(), "\n")
	b.WriteString(c.renderContactsList(visible, width, max(1, height-headerLines)))
	return style.Render(b.String())
}

func (c Contacts) detailView(width, height int) string {
	contact := c.detail
	var b strings.Builder
	b.WriteString(contact.Meta.DisplayName + "\n")
	b.WriteString("Type: " + contact.Meta.ContactType + "\n")
	if contact.Meta.PrimaryEmailAddress != "" {
		b.WriteString("Email: " + contact.Meta.PrimaryEmailAddress + "\n")
	}
	if contact.Meta.CompanyName != "" {
		b.WriteString("Company: " + contact.Meta.CompanyName + "\n")
	}
	if contact.Meta.Title != "" {
		b.WriteString("Title: " + contact.Meta.Title + "\n")
	}
	if contact.Meta.Phone != "" {
		b.WriteString("Phone: " + contact.Meta.Phone + "\n")
	}
	if contact.Meta.Website != "" {
		b.WriteString("Website: " + contact.Meta.Website + "\n")
	}
	if len(contact.Meta.EmailAddresses) > 1 {
		b.WriteString("\nEmail addresses\n")
		for _, email := range contact.Meta.EmailAddresses {
			label := email.Label
			if label == "" {
				label = "email"
			}
			b.WriteString("  " + label + ": " + email.EmailAddress + "\n")
		}
	}
	if contact.Note != nil && strings.TrimSpace(contact.Note.Body) != "" {
		b.WriteString("\nNote: " + contact.Note.Meta.Title + "\n")
		b.WriteString(contact.Note.Body + "\n")
	}
	if len(contact.Communications) > 0 {
		items := append([]contactstore.CommunicationMeta(nil), contact.Communications...)
		sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
		b.WriteString("\nCommunications\n")
		for _, item := range items {
			summary := item.Subject
			if summary == "" {
				summary = item.PreviewText
			}
			b.WriteString(fmt.Sprintf("  %s · %s · %s\n", item.OccurredAt.Format("2006-01-02"), item.Direction, summary))
		}
	} else {
		b.WriteString("\nPress e to edit contact, c to load communications, or N to refresh note.\n")
	}
	body := b.String()
	c.detailViewport.SetWidth(width)
	c.detailViewport.SetHeight(height)
	c.detailViewport.SetContent(strings.TrimRight(body, "\n"))
	return c.detailViewport.View()
}
