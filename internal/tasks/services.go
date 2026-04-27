package tasks

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

func (s *Service) Workspace(ctx context.Context) (*Workspace, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/tasks/workspace", nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Workspace](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ListProjects(ctx context.Context, params ListParams) ([]Project, *api.Pagination, error) {
	return api.List[Project](s.client, ctx, "/api/v1/tasks/projects", listQuery(params))
}

func (s *Service) ShowProject(ctx context.Context, id int64) (*Project, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Project](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateProject(ctx context.Context, input ProjectInput) (*Project, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/tasks/projects", map[string]any{"project": projectInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Project](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateProject(ctx context.Context, id int64, input ProjectInput) (*Project, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d", id), map[string]any{"project": projectInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Project](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteProject(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d", id))
	return err
}

func (s *Service) ShowBoard(ctx context.Context, projectID int64) (*Board, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/board", projectID), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Board](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateBoard(ctx context.Context, projectID int64, input BoardInput) (*Board, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/board", projectID), map[string]any{"board": map[string]any{"body": input.Body}})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Board](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ListCards(ctx context.Context, projectID int64, params ListParams) ([]Card, *api.Pagination, error) {
	return api.List[Card](s.client, ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/cards", projectID), listQuery(params))
}

func (s *Service) ShowCard(ctx context.Context, projectID, id int64) (*Card, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/cards/%d", projectID, id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Card](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateCard(ctx context.Context, projectID int64, input CardInput) (*Card, error) {
	body, _, err := s.client.Post(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/cards", projectID), map[string]any{"card": cardInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Card](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateCard(ctx context.Context, projectID, id int64, input CardInput) (*Card, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/cards/%d", projectID, id), map[string]any{"card": cardInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Card](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DeleteCard(ctx context.Context, projectID, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/tasks/projects/%d/cards/%d", projectID, id))
	return err
}

func listQuery(params ListParams) url.Values {
	query := url.Values{}
	api.SetInt(query, "page", params.Page)
	api.SetInt(query, "per_page", params.PerPage)
	return query
}

func projectInputMap(input ProjectInput) map[string]any {
	return map[string]any{"name": input.Name}
}

func cardInputMap(input CardInput) map[string]any {
	return map[string]any{"title": input.Title, "body": input.Body}
}
