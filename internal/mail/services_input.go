package mail

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
