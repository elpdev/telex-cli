package app

import (
	"fmt"
	"time"

	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

func cachedRemoteMessage(message mail.Message) mailstore.CachedMessage {
	return mailstore.CachedMessage{
		Meta: mailstore.MessageMeta{
			SchemaVersion:  mailstore.SchemaVersion,
			Kind:           "remote-message",
			RemoteID:       message.ID,
			ConversationID: message.ConversationID,
			InboxID:        message.InboxID,
			Mailbox:        message.SystemState,
			Status:         message.Status,
			Subject:        message.Subject,
			FromAddress:    message.FromAddress,
			FromName:       message.FromName,
			To:             message.ToAddresses,
			CC:             message.CCAddresses,
			Read:           message.Read,
			Starred:        message.Starred,
			SenderBlocked:  message.SenderBlocked,
			SenderTrusted:  message.SenderTrusted,
			DomainBlocked:  message.DomainBlocked,
			Labels:         remoteLabelMetas(message.Labels),
			Attachments:    remoteAttachmentMetas(message.Attachments),
			ReceivedAt:     message.ReceivedAt,
			SyncedAt:       time.Now(),
		},
		Path:     fmt.Sprintf("remote:%d", message.ID),
		BodyText: message.TextBody,
	}
}

func remoteLabelMetas(labels []mail.Label) []mailstore.LabelMeta {
	metas := make([]mailstore.LabelMeta, 0, len(labels))
	for _, label := range labels {
		metas = append(metas, mailstore.LabelMeta{ID: label.ID, Name: label.Name, Color: label.Color})
	}
	return metas
}

func remoteAttachmentMetas(attachments []mail.Attachment) []mailstore.AttachmentMeta {
	metas := make([]mailstore.AttachmentMeta, 0, len(attachments))
	for _, attachment := range attachments {
		metas = append(metas, mailstore.AttachmentMeta{
			ID:          attachment.ID,
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			ByteSize:    attachment.ByteSize,
			Previewable: attachment.Previewable,
			PreviewKind: attachment.PreviewKind,
			PreviewURL:  attachment.PreviewURL,
			DownloadURL: attachment.DownloadURL,
		})
	}
	return metas
}
