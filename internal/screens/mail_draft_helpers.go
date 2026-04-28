package screens

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/emailtext"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func editorCommand(path string) (*exec.Cmd, error) {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return nil, fmt.Errorf("editor is not configured")
	}
	args := append(parts[1:], path)
	return exec.Command(parts[0], args...), nil
}

type draftFields struct {
	From            string
	To              []string
	CC              []string
	BCC             []string
	Subject         string
	Body            string
	SourceMessageID int64
	ConversationID  int64
	DraftKind       string
}

func draftTemplate(fields draftFields) string {
	extra := ""
	if fields.SourceMessageID > 0 {
		extra += fmt.Sprintf("X-Telex-Source-Message-ID: %d\n", fields.SourceMessageID)
	}
	if fields.ConversationID > 0 {
		extra += fmt.Sprintf("X-Telex-Conversation-ID: %d\n", fields.ConversationID)
	}
	if fields.DraftKind != "" {
		extra += fmt.Sprintf("X-Telex-Draft-Kind: %s\n", fields.DraftKind)
	}
	return fmt.Sprintf("From: %s\nTo: %s\nCc: %s\nBcc: %s\nSubject: %s\n%s\n%s", fields.From, strings.Join(fields.To, ", "), strings.Join(fields.CC, ", "), strings.Join(fields.BCC, ", "), fields.Subject, extra, fields.Body)
}

func saveEditedDraft(store mailstore.Store, mailbox mailstore.MailboxMeta, path, existingPath string) (*mailstore.Draft, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	fields, err := parseDraftFile(string(content))
	if err != nil {
		return nil, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	input := mailstore.DraftInput{Mailbox: mailbox, Subject: fields.Subject, To: fields.To, CC: fields.CC, BCC: fields.BCC, Body: fields.Body, SourceMessageID: fields.SourceMessageID, ConversationID: fields.ConversationID, DraftKind: fields.DraftKind, Now: time.Now()}
	var draft *mailstore.Draft
	if existingPath != "" {
		draft, err = store.UpdateDraft(existingPath, input)
	} else {
		draft, err = store.CreateDraft(input)
	}
	if err != nil {
		return nil, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return draft, nil
}

func parseDraftFile(content string) (draftFields, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	parts := strings.SplitN(content, "\n\n", 2)
	if len(parts) != 2 {
		return draftFields{}, fmt.Errorf("draft must contain headers, a blank line, then body")
	}
	fields := draftFields{Body: parts[1]}
	for _, line := range strings.Split(parts[0], "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "from":
			fields.From = strings.TrimSpace(value)
		case "to":
			fields.To = splitDraftAddresses(value)
		case "cc":
			fields.CC = splitDraftAddresses(value)
		case "bcc":
			fields.BCC = splitDraftAddresses(value)
		case "subject":
			fields.Subject = strings.TrimSpace(value)
		case "x-telex-source-message-id":
			fields.SourceMessageID = parseDraftInt(value)
		case "x-telex-conversation-id":
			fields.ConversationID = parseDraftInt(value)
		case "x-telex-draft-kind":
			fields.DraftKind = strings.TrimSpace(value)
		}
	}
	if strings.TrimSpace(fields.Subject) == "" {
		return draftFields{}, fmt.Errorf("subject is required")
	}
	return fields, nil
}

func parseDraftInt(value string) int64 {
	var parsed int64
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed)
	return parsed
}

func splitDraftAddresses(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' })
	addresses := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			addresses = append(addresses, part)
		}
	}
	return addresses
}

func quotedReplyBody(message mailstore.CachedMessage) string {
	body := quotedSourceBody(message)
	if body == "" {
		return "\n\n"
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = "> " + line
	}
	return "\n\n" + strings.Join(lines, "\n") + "\n"
}

func quotedForwardBody(message mailstore.CachedMessage) string {
	body := quotedSourceBody(message)
	var b strings.Builder
	b.WriteString("\n\n---------- Forwarded message ---------\n")
	b.WriteString(fmt.Sprintf("From: %s\n", senderLine(message)))
	if len(message.Meta.To) > 0 {
		b.WriteString(fmt.Sprintf("To: %s\n", strings.Join(message.Meta.To, ", ")))
	}
	if len(message.Meta.CC) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(message.Meta.CC, ", ")))
	}
	if !message.Meta.ReceivedAt.IsZero() {
		b.WriteString(fmt.Sprintf("Date: %s\n", message.Meta.ReceivedAt.Format(time.RFC1123)))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\n\n", message.Meta.Subject))
	if body != "" {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func quotedSourceBody(message mailstore.CachedMessage) string {
	text := strings.TrimSpace(emailtext.DecodeQuotedPrintable(message.BodyText))
	if text != "" {
		if looksLikeHTMLBody(text) {
			return htmlBodyToText(text)
		}
		return text
	}
	return htmlBodyToText(strings.TrimSpace(emailtext.DecodeQuotedPrintable(message.BodyHTML)))
}

func htmlBodyToText(body string) string {
	if body == "" {
		return ""
	}
	markdown, err := emailtext.HTMLToMarkdown(body)
	if err != nil || strings.TrimSpace(markdown) == "" {
		return strings.TrimSpace(body)
	}
	return strings.TrimSpace(markdown)
}

func looksLikeHTMLBody(body string) bool {
	lower := strings.ToLower(body)
	for _, token := range []string{"<html", "<body", "<div", "<p", "<br", "<blockquote", "<table", "<ul", "<ol", "<li"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func senderLine(message mailstore.CachedMessage) string {
	if strings.TrimSpace(message.Meta.FromName) == "" {
		return message.Meta.FromAddress
	}
	return fmt.Sprintf("%s <%s>", message.Meta.FromName, message.Meta.FromAddress)
}

func draftsToCachedMessages(drafts []mailstore.Draft) []mailstore.CachedMessage {
	messages := make([]mailstore.CachedMessage, 0, len(drafts))
	for _, draft := range drafts {
		messages = append(messages, mailstore.CachedMessage{
			Meta: mailstore.MessageMeta{
				SchemaVersion: draft.Meta.SchemaVersion,
				Kind:          draft.Meta.Kind,
				RemoteID:      draft.Meta.RemoteID,
				DomainID:      draft.Meta.DomainID,
				DomainName:    draft.Meta.DomainName,
				InboxID:       draft.Meta.InboxID,
				Mailbox:       draft.Meta.Kind,
				Status:        draft.Meta.RemoteStatus,
				RemoteError:   draft.Meta.RemoteError,
				Attachments:   draft.Meta.Attachments,
				Subject:       draft.Meta.Subject,
				FromAddress:   draft.Meta.FromAddress,
				To:            draft.Meta.To,
				CC:            draft.Meta.CC,
				Read:          true,
				ReceivedAt:    draft.Meta.UpdatedAt,
				SyncedAt:      draft.Meta.UpdatedAt,
			},
			Path:     draft.Path,
			BodyText: draft.Body,
		})
	}
	return messages
}
