package mail

import "time"

type ListParams struct {
	Page    int
	PerPage int
}

type Label struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MailboxCounts struct {
	Inbox    int `json:"inbox"`
	Junk     int `json:"junk"`
	Archived int `json:"archived"`
	Trash    int `json:"trash"`
	Sent     int `json:"sent"`
	Drafts   int `json:"drafts"`
}

type MailboxBootstrap struct {
	Counts  MailboxCounts `json:"counts"`
	Labels  []Label       `json:"labels"`
	Inboxes []Inbox       `json:"inboxes"`
	Domains []Domain      `json:"domains"`
}

type Domain struct {
	ID                          int64     `json:"id"`
	Name                        string    `json:"name"`
	Active                      bool      `json:"active"`
	OutboundFromName            string    `json:"outbound_from_name"`
	OutboundFromAddress         string    `json:"outbound_from_address"`
	UseFromAddressForReplyTo    bool      `json:"use_from_address_for_reply_to"`
	ReplyToAddress              string    `json:"reply_to_address"`
	SMTPHost                    string    `json:"smtp_host"`
	SMTPPort                    int       `json:"smtp_port"`
	SMTPAuthentication          string    `json:"smtp_authentication"`
	SMTPEnableStartTLSAuto      bool      `json:"smtp_enable_starttls_auto"`
	SMTPUsername                string    `json:"smtp_username"`
	OutboundReady               bool      `json:"outbound_ready"`
	OutboundConfigurationErrors []string  `json:"outbound_configuration_errors"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

type Inbox struct {
	ID                    int64            `json:"id"`
	DomainID              int64            `json:"domain_id"`
	Address               string           `json:"address"`
	LocalPart             string           `json:"local_part"`
	PipelineKey           string           `json:"pipeline_key"`
	PipelineOverrides     map[string]any   `json:"pipeline_overrides"`
	ForwardingRules       []map[string]any `json:"forwarding_rules"`
	ActiveForwardingRules []map[string]any `json:"active_forwarding_rules"`
	Description           string           `json:"description"`
	Active                bool             `json:"active"`
	MessageCount          int              `json:"message_count"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
}

type Attachment struct {
	ID          int64     `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	ByteSize    int64     `json:"byte_size"`
	CreatedAt   time.Time `json:"created_at"`
	Previewable bool      `json:"previewable"`
	PreviewKind string    `json:"preview_kind"`
	PreviewURL  string    `json:"preview_url"`
	DownloadURL string    `json:"download_url"`
}

type Message struct {
	ID             int64          `json:"id"`
	InboxID        int64          `json:"inbox_id"`
	ConversationID int64          `json:"conversation_id"`
	MessageID      string         `json:"message_id"`
	FromAddress    string         `json:"from_address"`
	FromName       string         `json:"from_name"`
	SenderDisplay  string         `json:"sender_display"`
	ToAddresses    []string       `json:"to_addresses"`
	CCAddresses    []string       `json:"cc_addresses"`
	Subject        string         `json:"subject"`
	Subaddress     string         `json:"subaddress"`
	Status         string         `json:"status"`
	PreviewText    string         `json:"preview_text"`
	TextBody       string         `json:"text_body"`
	HTMLEmail      bool           `json:"html_email"`
	Metadata       map[string]any `json:"metadata"`
	Read           bool           `json:"read"`
	ReadAt         *time.Time     `json:"read_at"`
	Starred        bool           `json:"starred"`
	SystemState    string         `json:"system_state"`
	Labels         []Label        `json:"labels"`
	ReceivedAt     time.Time      `json:"received_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	Attachments    []Attachment   `json:"attachments"`
}

type MessageBody struct {
	ID           int64         `json:"id"`
	HTML         string        `json:"html"`
	RawHTML      string        `json:"raw_html"`
	Text         string        `json:"text"`
	HTMLEmail    bool          `json:"html_email"`
	InlineAssets []InlineAsset `json:"inline_assets"`
}

type InlineAsset struct {
	Token       string `json:"token"`
	ContentID   string `json:"content_id"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
}

type MessageListParams struct {
	ListParams
	InboxID        int64
	ConversationID int64
	Mailbox        string
	LabelID        int64
	Query          string
	Sender         string
	Recipient      string
	Status         string
	Subaddress     string
	ReceivedFrom   string
	ReceivedTo     string
	Sort           string
}

type OutboundMessage struct {
	ID                  int64          `json:"id"`
	DomainID            int64          `json:"domain_id"`
	InboxID             int64          `json:"inbox_id"`
	SourceMessageID     int64          `json:"source_message_id"`
	ConversationID      int64          `json:"conversation_id"`
	ToAddresses         []string       `json:"to_addresses"`
	CCAddresses         []string       `json:"cc_addresses"`
	BCCAddresses        []string       `json:"bcc_addresses"`
	Subject             string         `json:"subject"`
	BodyHTML            string         `json:"body_html"`
	BodyText            string         `json:"body_text"`
	Status              string         `json:"status"`
	DeliveryAttempts    int            `json:"delivery_attempts"`
	MailMessageID       string         `json:"mail_message_id"`
	InReplyToMessageID  string         `json:"in_reply_to_message_id"`
	ReferenceMessageIDs []string       `json:"reference_message_ids"`
	Metadata            map[string]any `json:"metadata"`
	LastError           string         `json:"last_error"`
	QueuedAt            *time.Time     `json:"queued_at"`
	SentAt              *time.Time     `json:"sent_at"`
	FailedAt            *time.Time     `json:"failed_at"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	Attachments         []Attachment   `json:"attachments"`
}

type OutboundMessageInput struct {
	DomainID            *int64
	InboxID             *int64
	SourceMessageID     *int64
	ConversationID      *int64
	ToAddresses         []string
	CCAddresses         []string
	BCCAddresses        []string
	Subject             string
	Body                string
	Status              string
	InReplyToMessageID  string
	ReferenceMessageIDs []string
	Metadata            map[string]any
}
