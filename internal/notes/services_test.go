package notes

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

func TestListNotesBuildsFolderSortPaginationQuery(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[{"id":9,"title":"Plan","body":"body"}],"meta":{"page":2,"per_page":50,"total_count":1}}`)}
	service := NewService(fake)
	folderID := int64(42)
	notes, pagination, err := service.ListNotes(context.Background(), ListNotesParams{ListParams: ListParams{Page: 2, PerPage: 50}, FolderID: &folderID, Sort: "updated_at"})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, fake.query, "folder_id", "42")
	assertQuery(t, fake.query, "page", "2")
	assertQuery(t, fake.query, "per_page", "50")
	assertQuery(t, fake.query, "sort", "updated_at")
	if len(notes) != 1 || notes[0].Body != "body" {
		t.Fatalf("notes = %#v", notes)
	}
	if pagination == nil || pagination.Page != 2 || pagination.PerPage != 50 || pagination.TotalCount != 1 {
		t.Fatalf("pagination = %#v", pagination)
	}
}

func TestNotesTreeDecodesChildren(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":1,"name":"Notes","note_count":2,"child_folder_count":1,"children":[{"id":2,"parent_id":1,"name":"Projects","note_count":3,"children":[]}]}}`)}
	service := NewService(fake)
	tree, err := service.NotesTree(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/notes/tree" || tree.Name != "Notes" || tree.NoteCount != 2 || len(tree.Children) != 1 || tree.Children[0].Name != "Projects" {
		t.Fatalf("path=%q tree=%#v", fake.getPath, tree)
	}
}

func TestShowNoteUsesShowEndpointAndDecodesBody(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"title":"Plan","filename":"plan.md","body":"# Plan"}}`)}
	service := NewService(fake)
	note, err := service.ShowNote(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if fake.getPath != "/api/v1/notes/9" || note.Body != "# Plan" || note.Filename != "plan.md" {
		t.Fatalf("path=%q note=%#v", fake.getPath, note)
	}
}

func TestCreateNoteUsesNotePayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"title":"Plan"}}`)}
	service := NewService(fake)
	folderID := int64(42)
	note, err := service.CreateNote(context.Background(), NoteInput{FolderID: &folderID, Title: "Plan", Body: "body"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"note": map[string]any{"folder_id": float64(42), "title": "Plan", "body": "body"}}
	if fake.postPath != "/api/v1/notes" || !jsonEqual(fake.postBody, want) || note.ID != 9 {
		t.Fatalf("path=%q body=%#v note=%#v", fake.postPath, fake.postBody, note)
	}
}

func TestUpdateNoteUsesPatchPayload(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"id":9,"title":"Updated"}}`)}
	service := NewService(fake)
	_, err := service.UpdateNote(context.Background(), 9, NoteInput{Title: "Updated", Body: "new body"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"note": map[string]any{"title": "Updated", "body": "new body"}}
	if fake.patchPath != "/api/v1/notes/9" || !jsonEqual(fake.patchBody, want) {
		t.Fatalf("path=%q body=%#v", fake.patchPath, fake.patchBody)
	}
}

func TestDeleteNoteUsesDeleteEndpoint(t *testing.T) {
	fake := &fakeClient{}
	service := NewService(fake)
	if err := service.DeleteNote(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if fake.deletePath != "/api/v1/notes/9" {
		t.Fatalf("deletePath = %q", fake.deletePath)
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

func assertQuery(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Fatalf("query[%s] = %q, want %q", key, got, want)
	}
}
