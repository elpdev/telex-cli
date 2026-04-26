package tasks

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestTasksServiceUsesWorkspaceEndpoint(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"root_folder":{"id":1,"name":"Tasks"},"projects_folder":{"id":2,"name":"Projects"},"projects":[]}}`)}
	service := NewService(fake)
	workspace, err := service.Workspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/tasks/workspace" || workspace.RootFolder.Name != "Tasks" || workspace.ProjectsFolder.Name != "Projects" {
		t.Fatalf("path=%q workspace=%#v", fake.getPath, workspace)
	}
}

func TestTasksServiceCreatesCardPayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"title":"Homepage","filename":"Homepage.md","body":"# Homepage"}}`)}
	service := NewService(fake)
	card, err := service.CreateCard(context.Background(), 4, CardInput{Title: "Homepage", Body: "# Homepage"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"card": map[string]any{"title": "Homepage", "body": "# Homepage"}}
	if fake.postPath != "/api/v1/tasks/projects/4/cards" || !jsonEqual(fake.postBody, want) || card.ID != 9 {
		t.Fatalf("path=%q body=%#v card=%#v", fake.postPath, fake.postBody, card)
	}
}

func TestTasksServiceUpdatesBoardPayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":5,"title":"Board","filename":"board.md","body":"# Board"}}`)}
	service := NewService(fake)
	board, err := service.UpdateBoard(context.Background(), 4, BoardInput{Body: "# Board"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"board": map[string]any{"body": "# Board"}}
	if fake.patchPath != "/api/v1/tasks/projects/4/board" || !jsonEqual(fake.patchBody, want) || board.Body != "# Board" {
		t.Fatalf("path=%q body=%#v board=%#v", fake.patchPath, fake.patchBody, board)
	}
}

type fakeClient struct {
	body       []byte
	query      url.Values
	getPath    string
	postPath   string
	postBody   any
	patchPath  string
	patchBody  any
	deletePath string
}

func (f *fakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, int, error) {
	f.getPath = path
	f.query = query
	return f.body, 200, nil
}

func (f *fakeClient) Post(_ context.Context, path string, body any) ([]byte, int, error) {
	f.postPath = path
	f.postBody = normalizeJSON(body)
	return f.body, 201, nil
}

func (f *fakeClient) Patch(_ context.Context, path string, body any) ([]byte, int, error) {
	f.patchPath = path
	f.patchBody = normalizeJSON(body)
	return f.body, 200, nil
}

func (f *fakeClient) Delete(_ context.Context, path string) (int, error) {
	f.deletePath = path
	return 204, nil
}

func normalizeJSON(value any) any {
	payload, _ := json.Marshal(value)
	var out any
	_ = json.Unmarshal(payload, &out)
	return out
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
