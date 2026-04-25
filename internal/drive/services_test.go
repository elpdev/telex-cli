package drive

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestListFoldersBuildsRootSearchQuery(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[],"meta":{"page":1,"per_page":25,"total_count":0}}`)}
	service := NewService(fake)
	_, _, err := service.ListFolders(context.Background(), ListFoldersParams{ListParams: ListParams{Page: 1, PerPage: 25}, Root: true, Query: "invoice", Sort: "name"})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, fake.query, "parent_id", "root")
	assertQuery(t, fake.query, "q", "invoice")
	assertQuery(t, fake.query, "sort", "name")
}

func TestListFilesBuildsFolderQuery(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":[],"meta":{"page":1,"per_page":25,"total_count":0}}`)}
	service := NewService(fake)
	folderID := int64(42)
	_, _, err := service.ListFiles(context.Background(), ListFilesParams{ListParams: ListParams{Page: 2, PerPage: 50}, FolderID: &folderID, Query: "pdf"})
	if err != nil {
		t.Fatal(err)
	}
	assertQuery(t, fake.query, "folder_id", "42")
	assertQuery(t, fake.query, "page", "2")
	assertQuery(t, fake.query, "per_page", "50")
	assertQuery(t, fake.query, "q", "pdf")
}

func TestUploadFileUsesDirectUploadFlow(t *testing.T) {
	fake := &fakeClient{body: []byte(`{"data":{"signed_id":"signed","direct_upload":{"url":"/rails/active_storage/disk","headers":{"Content-Type":"text/plain"}}}}`)}
	service := NewService(fake)
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	fake.createFileBody = []byte(`{"data":{"id":9,"filename":"note.txt"}}`)
	file, err := service.UploadFile(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if fake.postPath != "/api/v1/files" || fake.rawURL != "/rails/active_storage/disk" || fake.rawBody != "hello" {
		t.Fatalf("postPath=%q rawURL=%q rawBody=%q", fake.postPath, fake.rawURL, fake.rawBody)
	}
	if file.ID != 9 || file.Filename != "note.txt" {
		t.Fatalf("file = %#v", file)
	}
}

type fakeClient struct {
	body           []byte
	createFileBody []byte
	query          url.Values
	getPath        string
	postPath       string
	postBody       any
	patchPath      string
	deletePath     string
	rawURL         string
	rawHeaders     map[string]string
	rawBody        string
}

func (f *fakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, int, error) {
	f.getPath = path
	f.query = query
	return f.body, 200, nil
}

func (f *fakeClient) Post(_ context.Context, path string, body any) ([]byte, int, error) {
	f.postPath = path
	f.postBody = normalizeJSON(body)
	if path == "/api/v1/files" && f.createFileBody != nil {
		return f.createFileBody, 201, nil
	}
	return f.body, 200, nil
}

func (f *fakeClient) Patch(_ context.Context, path string, _ any) ([]byte, int, error) {
	f.patchPath = path
	return f.body, 200, nil
}

func (f *fakeClient) Delete(_ context.Context, path string) (int, error) {
	f.deletePath = path
	return 204, nil
}

func (f *fakeClient) Download(_ context.Context, _ string) ([]byte, string, error) {
	return []byte("download"), "text/plain", nil
}

func (f *fakeClient) PutRaw(_ context.Context, rawURL string, headers map[string]string, body io.Reader) (int, error) {
	f.rawURL = rawURL
	f.rawHeaders = headers
	payload, err := io.ReadAll(body)
	if err != nil {
		return 0, err
	}
	f.rawBody = string(payload)
	return 204, nil
}

func normalizeJSON(value any) any {
	payload, _ := json.Marshal(value)
	var out any
	_ = json.Unmarshal(payload, &out)
	return out
}

func assertQuery(t *testing.T, query url.Values, key, want string) {
	t.Helper()
	if got := query.Get(key); got != want {
		t.Fatalf("query[%s] = %q, want %q", key, got, want)
	}
}
