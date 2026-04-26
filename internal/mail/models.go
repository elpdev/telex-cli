package mail

import (
	"time"

	"github.com/elpdev/telex-cli/internal/contacts"
)

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
	ID                          int64          `json:"id"`
	UserID                      int64          `json:"user_id"`
	DriveFolderID               int64          `json:"drive_folder_id"`
	Name                        string         `json:"name"`
	Active                      bool           `json:"active"`
	OutboundFromName            string         `json:"outbound_from_name"`
	OutboundFromAddress         string         `json:"outbound_from_address"`
	UseFromAddressForReplyTo    bool           `json:"use_from_address_for_reply_to"`
	ReplyToAddress              string         `json:"reply_to_address"`
	SMTPHost                    string         `json:"smtp_host"`
	SMTPPort                    int            `json:"smtp_port"`
	SMTPAuthentication          string         `json:"smtp_authentication"`
	SMTPEnableStartTLSAuto      bool           `json:"smtp_enable_starttls_auto"`
	SMTPUsername                string         `json:"smtp_username"`
	OutboundReady               bool           `json:"outbound_ready"`
	OutboundConfigurationErrors []string       `json:"outbound_configuration_errors"`
	OutboundIdentity            map[string]any `json:"outbound_identity"`
	CreatedAt                   time.Time      `json:"created_at"`
	UpdatedAt                   time.Time      `json:"updated_at"`
}

type Inbox struct {
	ID                     int64            `json:"id"`
	DomainID               int64            `json:"domain_id"`
	DriveFolderID          int64            `json:"drive_folder_id"`
	EffectiveDriveFolderID int64            `json:"effective_drive_folder_id"`
	Address                string           `json:"address"`
	LocalPart              string           `json:"local_part"`
	PipelineKey            string           `json:"pipeline_key"`
	PipelineOverrides      map[string]any   `json:"pipeline_overrides"`
	ForwardingRules        []map[string]any `json:"forwarding_rules"`
	ActiveForwardingRules  []map[string]any `json:"active_forwarding_rules"`
	Description            string           `json:"description"`
	Active                 bool             `json:"active"`
	MessageCount           int              `json:"message_count"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
}

type DomainListParams struct {
	ListParams
	Active *bool
	Sort   string
}

type InboxListParams struct {
	ListParams
	DomainID    int64
	Active      *bool
	PipelineKey string
	Count       string
	Sort        string
}

type DomainInput struct {
	Name                     string
	Active                   *bool
	OutboundFromName         string
	OutboundFromAddress      string
	UseFromAddressForReplyTo *bool
	ReplyToAddress           string
	SMTPHost                 string
	SMTPPort                 *int
	SMTPAuthentication       string
	SMTPEnableStartTLSAuto   *bool
	SMTPUsername             string
	SMTPPassword             string
	DriveFolderID            *int64
}

type InboxInput struct {
	DomainID          *int64
	LocalPart         string
	PipelineKey       string
	Description       string
	Active            *bool
	DriveFolderID     *int64
	PipelineOverrides map[string]any
	ForwardingRules   []ForwardingRule
}

type ForwardingRule struct {
	Name               string   `json:"name"`
	Active             bool     `json:"active"`
	FromAddressPattern string   `json:"from_address_pattern"`
	SubjectPattern     string   `json:"subject_pattern"`
	SubaddressPattern  string   `json:"subaddress_pattern"`
	TargetAddresses    []string `json:"target_addresses"`
}

type DomainOutboundStatus struct {
	ID                          int64          `json:"id"`
	OutboundReady               bool           `json:"outbound_ready"`
	OutboundConfigurationErrors []string       `json:"outbound_configuration_errors"`
	OutboundIdentity            map[string]any `json:"outbound_identity"`
}

type DomainOutboundValidation struct {
	Valid                       bool                `json:"valid"`
	OutboundReady               bool                `json:"outbound_ready"`
	Errors                      map[string][]string `json:"errors"`
	OutboundConfigurationErrors []string            `json:"outbound_configuration_errors"`
}

type InboxPipeline struct {
	Key       string         `json:"key"`
	Steps     []string       `json:"steps"`
	Overrides map[string]any `json:"overrides"`
}

type ForwardingRuleValidation struct {
	Valid           bool             `json:"valid"`
	Errors          []string         `json:"errors"`
	ForwardingRules []ForwardingRule `json:"forwarding_rules"`
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
	ID             int64                    `json:"id"`
	InboxID        int64                    `json:"inbox_id"`
	ConversationID int64                    `json:"conversation_id"`
	MessageID      string                   `json:"message_id"`
	FromAddress    string                   `json:"from_address"`
	FromName       string                   `json:"from_name"`
	Contact        *contacts.ContactSummary `json:"contact"`
	SenderDisplay  string                   `json:"sender_display"`
	ToAddresses    []string                 `json:"to_addresses"`
	CCAddresses    []string                 `json:"cc_addresses"`
	Subject        string                   `json:"subject"`
	Subaddress     string                   `json:"subaddress"`
	Status         string                   `json:"status"`
	PreviewText    string                   `json:"preview_text"`
	TextBody       string                   `json:"text_body"`
	HTMLEmail      bool                     `json:"html_email"`
	Metadata       map[string]any           `json:"metadata"`
	Read           bool                     `json:"read"`
	ReadAt         *time.Time               `json:"read_at"`
	Starred        bool                     `json:"starred"`
	SystemState    string                   `json:"system_state"`
	SenderBlocked  bool                     `json:"sender_blocked"`
	SenderTrusted  bool                     `json:"sender_trusted"`
	DomainBlocked  bool                     `json:"domain_blocked"`
	Labels         []Label                  `json:"labels"`
	ReceivedAt     time.Time                `json:"received_at"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
	Attachments    []Attachment             `json:"attachments"`
}

type MessageBody struct {
	ID           int64         `json:"id"`
	HTML         string        `json:"html"`
	RawHTML      string        `json:"raw_html"`
	Text         string        `json:"text"`
	HTMLEmail    bool          `json:"html_email"`
	InlineAssets []InlineAsset `json:"inline_assets"`
}

type ConversationTimelineEntry struct {
	Kind           string    `json:"kind"`
	RecordID       int64     `json:"record_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Sender         string    `json:"sender"`
	Recipients     []string  `json:"recipients"`
	Summary        string    `json:"summary"`
	Status         string    `json:"status"`
	Subject        string    `json:"subject"`
	ConversationID int64     `json:"conversation_id"`
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

type OutboundMessageListParams struct {
	ListParams
	DomainID        int64
	ConversationID  int64
	SourceMessageID int64
	Status          string
	Sort            string
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
