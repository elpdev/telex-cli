package mail

import (
	"net/url"

	"github.com/elpdev/telex-cli/internal/api"
)

func messageQuery(params MessageListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	api.SetInt64(query, "inbox_id", params.InboxID)
	api.SetInt64(query, "conversation_id", params.ConversationID)
	api.SetString(query, "mailbox", params.Mailbox)
	api.SetInt64(query, "label_id", params.LabelID)
	api.SetString(query, "q", params.Query)
	api.SetString(query, "sender", params.Sender)
	api.SetString(query, "recipient", params.Recipient)
	api.SetString(query, "status", params.Status)
	api.SetString(query, "subaddress", params.Subaddress)
	api.SetString(query, "received_from", params.ReceivedFrom)
	api.SetString(query, "received_to", params.ReceivedTo)
	api.SetString(query, "updated_since", params.UpdatedSince)
	api.SetString(query, "sort", params.Sort)
	return query
}

func outboundMessageQuery(params OutboundMessageListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	api.SetInt64(query, "domain_id", params.DomainID)
	api.SetInt64(query, "conversation_id", params.ConversationID)
	api.SetInt64(query, "source_message_id", params.SourceMessageID)
	api.SetString(query, "status", params.Status)
	api.SetString(query, "updated_since", params.UpdatedSince)
	api.SetString(query, "sort", params.Sort)
	return query
}

func domainQuery(params DomainListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	api.SetBool(query, "active", params.Active)
	api.SetString(query, "sort", params.Sort)
	return query
}

func inboxQuery(params InboxListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	api.SetInt64(query, "domain_id", params.DomainID)
	api.SetBool(query, "active", params.Active)
	api.SetString(query, "pipeline_key", params.PipelineKey)
	api.SetString(query, "count", params.Count)
	api.SetString(query, "sort", params.Sort)
	return query
}
