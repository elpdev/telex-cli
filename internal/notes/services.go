package notes

import (
	"context"
	"fmt"
	"net/url"

	"github.com/elpdev/telex-cli/internal/api"
)

type Client interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
	Post(context.Context, string, any) ([]byte, int, error)
	Patch(context.Context, string, any) ([]byte, int, error)
	Delete(context.Context, string) (int, error)
}

type Service struct {
	client Client
}

func NewService(client Client) *Service { return &Service{client: client} }

func (s *Service) ListNotes(ctx context.Context, params ListNotesParams) ([]Note, *api.Pagination, error) {
	return api.List[Note](s.client, ctx, "/api/v1/notes", notesQuery(params))
}

func (s *Service) NotesTree(ctx context.Context) (*FolderTree, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/notes/tree", nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[FolderTree](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ShowNote(ctx context.Context, id int64) (*Note, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/notes/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Note](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateNote(ctx context.Context, input NoteInput) (*Note, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/notes", map[string]any{"note": noteInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Note](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateNote(ctx context.Context, id int64, input NoteInput) (*Note, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/notes/%d", id), map[string]any{"note": noteInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Note](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteNote(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/notes/%d", id))
	return err
}

func notesQuery(params ListNotesParams) url.Values {
	query := url.Values{}
	setPage(query, params.ListParams)
	if params.FolderID != nil {
		query.Set("folder_id", fmt.Sprintf("%d", *params.FolderID))
	}
	api.SetString(query, "updated_since", params.UpdatedSince)
	api.SetString(query, "sort", params.Sort)
	return query
}

func noteInputMap(input NoteInput) map[string]any {
	payload := map[string]any{
		"title": input.Title,
		"body":  input.Body,
	}
	if input.FolderID != nil {
		payload["folder_id"] = *input.FolderID
	}
	return payload
}

func setPage(query url.Values, params ListParams) {
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
}
