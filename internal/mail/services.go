package mail

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/elpdev/telex-cli/internal/api"
)

type Client interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
	Post(context.Context, string, any) ([]byte, int, error)
	PostMultipartFile(context.Context, string, string, string) ([]byte, int, error)
	Patch(context.Context, string, any) ([]byte, int, error)
	Delete(context.Context, string) (int, error)
}

type Service struct {
	client Client
}

func NewService(client Client) *Service { return &Service{client: client} }

func (s *Service) ListDomains(ctx context.Context, params DomainListParams) ([]Domain, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/domains", domainQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Domain](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
}

func (s *Service) ShowDomain(ctx context.Context, id int64) (*Domain, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/domains/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Domain](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateDomain(ctx context.Context, input DomainInput) (*Domain, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/domains", map[string]any{"domain": domainInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Domain](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateDomain(ctx context.Context, id int64, input DomainInput) (*Domain, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/domains/%d", id), map[string]any{"domain": domainInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Domain](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteDomain(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/domains/%d", id))
	return err
}

func (s *Service) DomainOutboundStatus(ctx context.Context, id int64) (*DomainOutboundStatus, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/domains/%d/outbound_status", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[DomainOutboundStatus](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ValidateDomainOutbound(ctx context.Context, id int64, input *DomainInput) (*DomainOutboundValidation, error) {
	var payload any
	if input != nil {
		payload = map[string]any{"domain": domainInputMap(*input)}
	}
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/domains/%d/validate_outbound", id), payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[DomainOutboundValidation](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ListInboxes(ctx context.Context, params InboxListParams) ([]Inbox, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/inboxes", inboxQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Inbox](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
}

func (s *Service) ShowInbox(ctx context.Context, id int64) (*Inbox, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/inboxes/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Inbox](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateInbox(ctx context.Context, input InboxInput) (*Inbox, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/inboxes", map[string]any{"inbox": inboxInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Inbox](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateInbox(ctx context.Context, id int64, input InboxInput) (*Inbox, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/inboxes/%d", id), map[string]any{"inbox": inboxInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Inbox](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteInbox(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/inboxes/%d", id))
	return err
}

func (s *Service) InboxPipeline(ctx context.Context, id int64) (*InboxPipeline, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/inboxes/%d/pipeline", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[InboxPipeline](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) TestInboxForwardingRules(ctx context.Context, id int64, rules []ForwardingRule) (*ForwardingRuleValidation, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/inboxes/%d/test_forwarding_rules", id), map[string]any{"inbox": map[string]any{"forwarding_rules": rules}})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[ForwardingRuleValidation](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) Mailboxes(ctx context.Context) (*MailboxBootstrap, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/mailboxes", nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[MailboxBootstrap](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) Labels(ctx context.Context) ([]Label, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/labels", nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Label](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ListMessages(ctx context.Context, params MessageListParams) ([]Message, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/messages", messageQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Message](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
}

func (s *Service) ShowMessage(ctx context.Context, id int64) (*Message, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/messages/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Message](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) MessageBody(ctx context.Context, id int64) (*MessageBody, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/messages/%d/body", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[MessageBody](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) AssignMessageLabels(ctx context.Context, id int64, labelIDs []int64) (*Message, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/messages/%d/labels", id), map[string]any{"label_ids": labelIDs})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Message](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ConversationTimeline(ctx context.Context, id int64) ([]ConversationTimelineEntry, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/conversations/%d/timeline", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]ConversationTimelineEntry](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ArchiveMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "archive", nil)
}

func (s *Service) RestoreMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "restore", nil)
}

func (s *Service) TrashMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "trash", nil)
}

func (s *Service) JunkMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "junk", nil)
}

func (s *Service) NotJunkMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "not_junk", nil)
}

func (s *Service) MarkMessageRead(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "mark_read", nil)
}

func (s *Service) MarkMessageUnread(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "mark_unread", nil)
}

func (s *Service) StarMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "star", nil)
}

func (s *Service) UnstarMessage(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "unstar", nil)
}

func (s *Service) BlockSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "block_sender", nil)
}

func (s *Service) UnblockSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "unblock_sender", nil)
}

func (s *Service) BlockDomain(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "block_domain", nil)
}

func (s *Service) UnblockDomain(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "unblock_domain", nil)
}

func (s *Service) TrustSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "trust_sender", nil)
}

func (s *Service) UntrustSender(ctx context.Context, id int64) (*Message, error) {
	return s.messageAction(ctx, id, http.MethodPost, "untrust_sender", nil)
}

func (s *Service) Reply(ctx context.Context, id int64) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "reply", nil)
}

func (s *Service) ReplyAll(ctx context.Context, id int64) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "reply_all", nil)
}

func (s *Service) Forward(ctx context.Context, id int64, targetAddresses []string) (*OutboundMessage, error) {
	return s.outboundAction(ctx, id, "forward", map[string]any{"target_addresses": targetAddresses})
}

func (s *Service) ListOutboundMessages(ctx context.Context, params OutboundMessageListParams) ([]OutboundMessage, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/outbound_messages", outboundMessageQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]OutboundMessage](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
}

func (s *Service) CreateOutboundMessage(ctx context.Context, input *OutboundMessageInput, queue bool) (*OutboundMessage, error) {
	payload := map[string]any{"outbound_message": outboundInputMap(input)}
	if queue {
		payload["queue"] = true
	}
	body, _, err := s.client.Post(ctx, "/api/v1/outbound_messages", payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateOutboundMessage(ctx context.Context, id int64, input *OutboundMessageInput) (*OutboundMessage, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d", id), map[string]any{"outbound_message": outboundInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteOutboundMessage(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d", id))
	return err
}

func (s *Service) SendOutboundMessage(ctx context.Context, id int64) (*OutboundMessage, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d/send_message", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) AttachOutboundMessageFile(ctx context.Context, outboundMessageID int64, filePath string) ([]Attachment, error) {
	body, _, err := s.client.PostMultipartFile(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d/attachments", outboundMessageID), "file", filePath)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Attachment](body)
	if err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func (s *Service) ShowOutboundMessage(ctx context.Context, id int64) (*OutboundMessage, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/outbound_messages/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) messageAction(ctx context.Context, id int64, method, action string, payload any) (*Message, error) {
	path := fmt.Sprintf("/api/v1/messages/%d/%s", id, action)
	var (
		body []byte
		err  error
	)
	if method == http.MethodPatch {
		body, _, err = s.client.Patch(ctx, path, payload)
	} else {
		body, _, err = s.client.Post(ctx, path, payload)
	}
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Message](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) outboundAction(ctx context.Context, id int64, action string, payload any) (*OutboundMessage, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/messages/%d/%s", id, action), payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[OutboundMessage](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func messageQuery(params MessageListParams) url.Values {
	query := url.Values{}
	setInt(query, "page", params.Page)
	setInt(query, "per_page", params.PerPage)
	setInt64(query, "inbox_id", params.InboxID)
	setInt64(query, "conversation_id", params.ConversationID)
	setString(query, "mailbox", params.Mailbox)
	setInt64(query, "label_id", params.LabelID)
	setString(query, "q", params.Query)
	setString(query, "sender", params.Sender)
	setString(query, "recipient", params.Recipient)
	setString(query, "status", params.Status)
	setString(query, "subaddress", params.Subaddress)
	setString(query, "received_from", params.ReceivedFrom)
	setString(query, "received_to", params.ReceivedTo)
	setString(query, "sort", params.Sort)
	return query
}

func outboundMessageQuery(params OutboundMessageListParams) url.Values {
	query := url.Values{}
	setInt(query, "page", params.Page)
	setInt(query, "per_page", params.PerPage)
	setInt64(query, "domain_id", params.DomainID)
	setInt64(query, "conversation_id", params.ConversationID)
	setInt64(query, "source_message_id", params.SourceMessageID)
	setString(query, "status", params.Status)
	setString(query, "sort", params.Sort)
	return query
}

func domainQuery(params DomainListParams) url.Values {
	query := url.Values{}
	setInt(query, "page", params.Page)
	setInt(query, "per_page", params.PerPage)
	if params.Active != nil {
		query.Set("active", fmt.Sprintf("%t", *params.Active))
	}
	setString(query, "sort", params.Sort)
	return query
}

func inboxQuery(params InboxListParams) url.Values {
	query := url.Values{}
	setInt(query, "page", params.Page)
	setInt(query, "per_page", params.PerPage)
	setInt64(query, "domain_id", params.DomainID)
	if params.Active != nil {
		query.Set("active", fmt.Sprintf("%t", *params.Active))
	}
	setString(query, "pipeline_key", params.PipelineKey)
	setString(query, "count", params.Count)
	setString(query, "sort", params.Sort)
	return query
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setInt(query url.Values, key string, value int) {
	if value > 0 {
		query.Set(key, fmt.Sprintf("%d", value))
	}
}

func setInt64(query url.Values, key string, value int64) {
	if value > 0 {
		query.Set(key, fmt.Sprintf("%d", value))
	}
}

func outboundInputMap(input *OutboundMessageInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	payload := map[string]any{}
	if input.DomainID != nil {
		payload["domain_id"] = *input.DomainID
	}
	if input.InboxID != nil {
		payload["inbox_id"] = *input.InboxID
	}
	if input.SourceMessageID != nil {
		payload["source_message_id"] = *input.SourceMessageID
	}
	if input.ConversationID != nil {
		payload["conversation_id"] = *input.ConversationID
	}
	if len(input.ToAddresses) > 0 {
		payload["to_addresses"] = input.ToAddresses
	}
	if len(input.CCAddresses) > 0 {
		payload["cc_addresses"] = input.CCAddresses
	}
	if len(input.BCCAddresses) > 0 {
		payload["bcc_addresses"] = input.BCCAddresses
	}
	if input.Subject != "" {
		payload["subject"] = input.Subject
	}
	if input.Body != "" {
		payload["body"] = input.Body
	}
	if input.Status != "" {
		payload["status"] = input.Status
	}
	if input.InReplyToMessageID != "" {
		payload["in_reply_to_message_id"] = input.InReplyToMessageID
	}
	if len(input.ReferenceMessageIDs) > 0 {
		payload["reference_message_ids"] = input.ReferenceMessageIDs
	}
	if len(input.Metadata) > 0 {
		payload["metadata"] = input.Metadata
	}
	return payload
}

func domainInputMap(input DomainInput) map[string]any {
	payload := map[string]any{}
	if input.Name != "" {
		payload["name"] = input.Name
	}
	if input.Active != nil {
		payload["active"] = *input.Active
	}
	if input.OutboundFromName != "" {
		payload["outbound_from_name"] = input.OutboundFromName
	}
	if input.OutboundFromAddress != "" {
		payload["outbound_from_address"] = input.OutboundFromAddress
	}
	if input.UseFromAddressForReplyTo != nil {
		payload["use_from_address_for_reply_to"] = *input.UseFromAddressForReplyTo
	}
	if input.ReplyToAddress != "" {
		payload["reply_to_address"] = input.ReplyToAddress
	}
	if input.SMTPHost != "" {
		payload["smtp_host"] = input.SMTPHost
	}
	if input.SMTPPort != nil {
		payload["smtp_port"] = *input.SMTPPort
	}
	if input.SMTPAuthentication != "" {
		payload["smtp_authentication"] = input.SMTPAuthentication
	}
	if input.SMTPEnableStartTLSAuto != nil {
		payload["smtp_enable_starttls_auto"] = *input.SMTPEnableStartTLSAuto
	}
	if input.SMTPUsername != "" {
		payload["smtp_username"] = input.SMTPUsername
	}
	if input.SMTPPassword != "" {
		payload["smtp_password"] = input.SMTPPassword
	}
	if input.DriveFolderID != nil {
		payload["drive_folder_id"] = *input.DriveFolderID
	}
	return payload
}

func inboxInputMap(input InboxInput) map[string]any {
	payload := map[string]any{}
	if input.DomainID != nil {
		payload["domain_id"] = *input.DomainID
	}
	if input.LocalPart != "" {
		payload["local_part"] = input.LocalPart
	}
	if input.PipelineKey != "" {
		payload["pipeline_key"] = input.PipelineKey
	}
	if input.Description != "" {
		payload["description"] = input.Description
	}
	if input.Active != nil {
		payload["active"] = *input.Active
	}
	if input.DriveFolderID != nil {
		payload["drive_folder_id"] = *input.DriveFolderID
	}
	if input.PipelineOverrides != nil {
		payload["pipeline_overrides"] = input.PipelineOverrides
	}
	if input.ForwardingRules != nil {
		payload["forwarding_rules"] = input.ForwardingRules
	}
	return payload
}
