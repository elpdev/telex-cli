package mailstore

import (
	"time"
)

type DraftInput struct {
	Mailbox         MailboxMeta
	Subject         string
	To              []string
	CC              []string
	BCC             []string
	Body            string
	SourceMessageID int64
	ConversationID  int64
	DraftKind       string
	Now             time.Time
}

type DraftMeta struct {
	SchemaVersion   int              `toml:"schema_version"`
	Kind            string           `toml:"kind"`
	ID              string           `toml:"id"`
	DomainID        int64            `toml:"domain_id"`
	DomainName      string           `toml:"domain_name"`
	InboxID         int64            `toml:"inbox_id"`
	FromAddress     string           `toml:"from_address"`
	RemoteID        int64            `toml:"remote_id"`
	SourceMessageID int64            `toml:"source_message_id"`
	ConversationID  int64            `toml:"conversation_id"`
	RemoteStatus    string           `toml:"remote_status"`
	RemoteError     string           `toml:"remote_error"`
	DraftKind       string           `toml:"draft_kind"`
	Subject         string           `toml:"subject"`
	To              []string         `toml:"to"`
	CC              []string         `toml:"cc"`
	BCC             []string         `toml:"bcc"`
	Attachments     []AttachmentMeta `toml:"attachments"`
	CreatedAt       time.Time        `toml:"created_at"`
	UpdatedAt       time.Time        `toml:"updated_at"`
}

type Draft struct {
	Meta DraftMeta
	Path string
	Body string
}
