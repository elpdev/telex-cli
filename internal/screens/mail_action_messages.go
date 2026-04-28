package screens

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

type MailActionMsg struct {
	Action string
}

// MailSelection describes what the mail screen has focused right now. The
// command palette reads this to gate selection-aware commands and to render
// dynamic descriptions (e.g. the subject of the draft about to be sent).
type MailSelection struct {
	Box      string
	Subject  string
	HasItem  bool
	IsDraft  bool
	BoxLikes string
}

func (m Mail) Selection() MailSelection {
	box := m.currentBox()
	sel := MailSelection{Box: box, IsDraft: box == "drafts"}
	if box == "inbox" || box == "junk" || box == "archive" || box == "trash" {
		sel.BoxLikes = "message"
	} else if box == "drafts" {
		sel.BoxLikes = "draft"
	}
	if len(m.messages) == 0 || m.messageIndex < 0 || m.messageIndex >= len(m.messages) {
		return sel
	}
	msg := m.messages[m.messageIndex]
	sel.Subject = msg.Meta.Subject
	sel.HasItem = true
	return sel
}

func (m Mail) handleAction(action string) (Screen, tea.Cmd) {
	if m.confirm != "" || m.searching || m.savingAttachment || m.filePickerActive || m.forwarding {
		return m, nil
	}
	switch action {
	case "compose":
		return m.editComposeDraft()
	case "sync":
		if m.sync == nil || m.syncing {
			return m, nil
		}
		m.syncing = true
		m.status = "Syncing mailboxes, outbox, and inbox..."
		return m, m.syncCmd()
	case "send-draft":
		if m.currentBox() != "drafts" || len(m.messages) == 0 {
			return m, nil
		}
		return m.requestConfirm("send-draft", "Send this draft?")
	case "edit-draft":
		return m.editSelectedDraft()
	case "delete-draft":
		if m.currentBox() != "drafts" || len(m.messages) == 0 {
			return m, nil
		}
		return m.requestConfirm("delete-draft", "Delete this draft?")
	case "attach":
		return m.startAttachFile()
	case "reply":
		if m.currentBox() != "inbox" || len(m.messages) == 0 {
			return m, nil
		}
		return m.editReplyDraft()
	case "forward":
		if len(m.messages) == 0 {
			return m, nil
		}
		return m.startForward()
	case "archive":
		if m.currentBox() != "inbox" || len(m.messages) == 0 {
			return m, nil
		}
		return m.moveSelectedMessage("archive")
	case "junk":
		if m.currentBox() != "inbox" || len(m.messages) == 0 {
			return m, nil
		}
		return m.moveSelectedMessage("junk")
	case "not-junk":
		if m.currentBox() != "junk" || len(m.messages) == 0 {
			return m, nil
		}
		return m.moveSelectedMessage("not-junk")
	case "trash":
		if m.currentBox() != "inbox" || len(m.messages) == 0 {
			return m, nil
		}
		return m.requestConfirm("trash", "Move this message to trash?")
	case "restore":
		if len(m.messages) == 0 {
			return m, nil
		}
		return m.moveSelectedMessage("restore")
	case "toggle-star":
		return m.toggleSelectedStar()
	case "toggle-read":
		return m.toggleSelectedRead()
	case "block-sender", "unblock-sender", "block-domain", "unblock-domain", "trust-sender", "untrust-sender":
		return m.updateSelectedSenderPolicy(action)
	}
	return m, nil
}

func (m Mail) updateSelectedSenderPolicy(action string) (Screen, tea.Cmd) {
	if len(m.messages) == 0 || m.remoteResults || !m.currentBoxSupportsMessageActions() {
		return m, nil
	}
	remote := m.senderPolicyAction(action)
	if remote == nil {
		m.status = fmt.Sprintf("%s action is not configured", action)
		return m, nil
	}
	message := m.messages[m.messageIndex]
	m.status = "Updating sender policy..."
	return m, func() tea.Msg {
		if err := remote(context.Background(), message.Meta.RemoteID); err != nil {
			return messagePolicyUpdatedMsg{path: message.Path, action: action, err: err}
		}
		cached, err := mailstore.ReadCachedMessage(message.Path)
		if err != nil {
			return messagePolicyUpdatedMsg{path: message.Path, action: action, err: err}
		}
		switch action {
		case "block-sender":
			cached.Meta.SenderBlocked = true
			cached.Meta.SenderTrusted = false
		case "unblock-sender":
			cached.Meta.SenderBlocked = false
		case "trust-sender":
			cached.Meta.SenderTrusted = true
			cached.Meta.SenderBlocked = false
		case "untrust-sender":
			cached.Meta.SenderTrusted = false
		case "block-domain":
			cached.Meta.DomainBlocked = true
		case "unblock-domain":
			cached.Meta.DomainBlocked = false
		}
		remoteMessage := mailstoreToRemoteMessage(*cached)
		_, err = mailstore.UpdateCachedMessageFromRemote(message.Path, remoteMessage, time.Now())
		return messagePolicyUpdatedMsg{path: message.Path, action: action, err: err}
	}
}

func (m Mail) senderPolicyAction(action string) MessageActionFunc {
	switch action {
	case "block-sender":
		return m.blockSender
	case "unblock-sender":
		return m.unblockSender
	case "block-domain":
		return m.blockDomain
	case "unblock-domain":
		return m.unblockDomain
	case "trust-sender":
		return m.trustSender
	case "untrust-sender":
		return m.untrustSender
	default:
		return nil
	}
}

func policyStatus(action string) string {
	switch action {
	case "block-sender":
		return "Sender blocked"
	case "unblock-sender":
		return "Sender unblocked"
	case "block-domain":
		return "Domain blocked"
	case "unblock-domain":
		return "Domain unblocked"
	case "trust-sender":
		return "Sender trusted"
	case "untrust-sender":
		return "Sender untrusted"
	default:
		return "Sender policy updated"
	}
}

func mailstoreToRemoteMessage(message mailstore.CachedMessage) mail.Message {
	labels := make([]mail.Label, 0, len(message.Meta.Labels))
	for _, label := range message.Meta.Labels {
		labels = append(labels, mail.Label{ID: label.ID, Name: label.Name, Color: label.Color})
	}
	return mail.Message{ID: message.Meta.RemoteID, ConversationID: message.Meta.ConversationID, InboxID: message.Meta.InboxID, FromAddress: message.Meta.FromAddress, FromName: message.Meta.FromName, ToAddresses: message.Meta.To, CCAddresses: message.Meta.CC, Subject: message.Meta.Subject, Status: message.Meta.Status, SystemState: message.Meta.Mailbox, Read: message.Meta.Read, Starred: message.Meta.Starred, SenderBlocked: message.Meta.SenderBlocked, SenderTrusted: message.Meta.SenderTrusted, DomainBlocked: message.Meta.DomainBlocked, Labels: labels, ReceivedAt: message.Meta.ReceivedAt, Attachments: remoteAttachments(message.Meta.Attachments)}
}

func remoteAttachments(attachments []mailstore.AttachmentMeta) []mail.Attachment {
	out := make([]mail.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		out = append(out, mail.Attachment{ID: attachment.ID, Filename: attachment.Filename, ContentType: attachment.ContentType, ByteSize: attachment.ByteSize, Previewable: attachment.Previewable, PreviewKind: attachment.PreviewKind, PreviewURL: attachment.PreviewURL, DownloadURL: attachment.DownloadURL})
	}
	return out
}
