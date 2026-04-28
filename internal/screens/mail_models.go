package screens

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/mailstore"
)

type ToggleReadFunc func(context.Context, int64, bool) error
type ToggleStarFunc func(context.Context, int64, bool) error
type MessageActionFunc func(context.Context, int64) error
type SyncFunc func(context.Context) (MailSyncResult, error)
type SendDraftFunc func(context.Context, mailstore.MailboxMeta, mailstore.Draft) error
type UpdateDraftFunc func(context.Context, mailstore.Draft) error
type DeleteDraftFunc func(context.Context, mailstore.Draft) error
type ForwardFunc func(context.Context, int64, mailstore.Draft) (int64, string, error)
type DownloadAttachmentFunc func(context.Context, mailstore.AttachmentMeta) ([]byte, error)
type RemoteSearchFunc func(context.Context, MailSearchParams) ([]mailstore.CachedMessage, error)
type ConversationFunc func(context.Context, int64) ([]ConversationEntry, error)
type ConversationBodyFunc func(context.Context, ConversationEntry) (string, error)

type MailSearchParams struct {
	InboxID int64
	Mailbox string
	Query   string
	Page    int
	PerPage int
	Sort    string
}

type ConversationEntry struct {
	Kind           string
	RecordID       int64
	OccurredAt     time.Time
	Sender         string
	Recipients     []string
	Summary        string
	Status         string
	Subject        string
	ConversationID int64
}

type MailSyncResult struct {
	ActiveMailboxes  int
	SkippedMailboxes int
	OutboxItems      int
	DraftItems       int
	InboxMessages    int
	BodyErrors       int
	InboxErrors      int
}
