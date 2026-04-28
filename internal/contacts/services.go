package contacts

import (
	"context"
	"fmt"
	"net/url"

	"github.com/elpdev/telex-cli/internal/api"
)

type Client interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
	Post(context.Context, string, any) ([]byte, int, error)
	Put(context.Context, string, any) ([]byte, int, error)
	Patch(context.Context, string, any) ([]byte, int, error)
	Delete(context.Context, string) (int, error)
}

type Service struct {
	client Client
}

func NewService(client Client) *Service { return &Service{client: client} }

func (s *Service) ListContacts(ctx context.Context, params ListContactsParams) ([]Contact, *api.Pagination, error) {
	return api.List[Contact](s.client, ctx, "/api/v1/contacts", contactsQuery(params))
}

func (s *Service) ShowContact(ctx context.Context, id int64, includeNote bool) (*Contact, error) {
	query := url.Values{}
	if includeNote {
		query.Set("include_note", "true")
	}
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/contacts/%d", id), query)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Contact](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateContact(ctx context.Context, input ContactInput) (*Contact, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/contacts", map[string]any{"contact": contactInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Contact](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateContact(ctx context.Context, id int64, input ContactInput) (*Contact, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/contacts/%d", id), map[string]any{"contact": contactInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Contact](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteContact(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/contacts/%d", id))
	return err
}

func (s *Service) ContactNote(ctx context.Context, id int64) (*ContactNote, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/contacts/%d/note", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[ContactNote](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateContactNote(ctx context.Context, id int64, input ContactNoteInput) (*ContactNote, error) {
	body, _, err := s.client.Put(ctx, fmt.Sprintf("/api/v1/contacts/%d/note", id), map[string]any{"note": contactNoteInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[ContactNote](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ContactCommunications(ctx context.Context, id int64, params ListParams) ([]ContactCommunication, *api.Pagination, error) {
	return api.List[ContactCommunication](s.client, ctx, fmt.Sprintf("/api/v1/contacts/%d/communications", id), listQuery(params))
}

func contactsQuery(params ListContactsParams) url.Values {
	query := listQuery(params.ListParams)
	api.SetString(query, "contact_type", params.ContactType)
	api.SetString(query, "q", params.Query)
	api.SetString(query, "updated_since", params.UpdatedSince)
	api.SetString(query, "sort", params.Sort)
	return query
}

func listQuery(params ListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	return query
}

func contactInputMap(input ContactInput) map[string]any {
	payload := map[string]any{}
	setPayloadString(payload, "contact_type", input.ContactType)
	setPayloadString(payload, "name", input.Name)
	setPayloadString(payload, "company_name", input.CompanyName)
	setPayloadString(payload, "title", input.Title)
	setPayloadString(payload, "phone", input.Phone)
	setPayloadString(payload, "website", input.Website)
	if input.EmailAddresses != nil {
		payload["email_addresses"] = input.EmailAddresses
	}
	if input.Metadata != nil {
		payload["metadata"] = input.Metadata
	}
	return payload
}

func contactNoteInputMap(input ContactNoteInput) map[string]any {
	payload := map[string]any{}
	setPayloadString(payload, "title", input.Title)
	payload["body"] = input.Body
	return payload
}

func setPayloadString(payload map[string]any, key, value string) {
	if value != "" {
		payload[key] = value
	}
}
