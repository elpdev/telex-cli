package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
)

func TestNotesCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"notes", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotesListAndShowReadLocalCache(t *testing.T) {
	dataDir := t.TempDir()
	store := notestore.New(dataDir)
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	folderID := int64(1)
	if err := store.StoreTree(&notes.FolderTree{FolderSummary: notes.FolderSummary{ID: folderID, Name: "Notes"}}, syncedAt); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreNote(notes.Note{ID: 9, FolderID: &folderID, Title: "Cached Note", Filename: "cached-note.md", Body: "# Cached"}, syncedAt); err != nil {
		t.Fatal(err)
	}

	list := newRootCommand(buildInfo{})
	var listOut bytes.Buffer
	list.SetOut(&listOut)
	list.SetErr(&bytes.Buffer{})
	list.SetArgs([]string{"--data-dir", dataDir, "notes", "list"})
	if err := list.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := listOut.String(); !strings.Contains(got, "Cached Note") || !strings.Contains(got, "cached-note.md") {
		t.Fatalf("list output = %q", got)
	}

	show := newRootCommand(buildInfo{})
	var showOut bytes.Buffer
	show.SetOut(&showOut)
	show.SetErr(&bytes.Buffer{})
	show.SetArgs([]string{"--data-dir", dataDir, "notes", "show", "9"})
	if err := show.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := showOut.String(); !strings.Contains(got, "Cached Note") || !strings.Contains(got, "# Cached") {
		t.Fatalf("show output = %q", got)
	}
}

func TestNotesSyncStoresTreeAndNotes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/notes/tree":
			_, _ = w.Write([]byte(`{"data":{"id":1,"name":"Notes","note_count":1,"child_folder_count":1,"children":[{"id":2,"parent_id":1,"name":"Projects","note_count":1,"child_folder_count":0,"children":[]}]}}`))
		case "/api/v1/notes":
			folderID := r.URL.Query().Get("folder_id")
			if folderID == "1" {
				_, _ = w.Write([]byte(`{"data":[{"id":9,"folder_id":1,"title":"Root Note","filename":"root-note.md","body":"root body"}],"meta":{"page":1,"per_page":100,"total_count":1}}`))
				return
			}
			if folderID == "2" {
				_, _ = w.Write([]byte(`{"data":[{"id":10,"folder_id":2,"title":"Project Note","filename":"project-note.md","body":"project body"}],"meta":{"page":1,"per_page":100,"total_count":1}}`))
				return
			}
			t.Fatalf("unexpected folder_id %q", folderID)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	configPath := writeNotesTestConfig(t, server.URL)
	dataDir := t.TempDir()
	cmd := newRootCommand(buildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", configPath, "--data-dir", dataDir, "notes", "sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Synced 2 folder(s), 2 note(s).") {
		t.Fatalf("output = %q", out.String())
	}
	cached, err := notestore.New(dataDir).ReadNote(10)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Body != "project body" || cached.Meta.FolderID != 2 {
		t.Fatalf("cached = %#v", cached)
	}
}

func TestNotesCreateReadsFileAndCachesRemoteNote(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notes" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"data":{"id":9,"folder_id":42,"title":"Created","filename":"created.md","body":"from file"}}`))
	}))
	defer server.Close()

	configPath := writeNotesTestConfig(t, server.URL)
	dataDir := t.TempDir()
	bodyPath := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(bodyPath, []byte("from file"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", configPath, "--data-dir", dataDir, "notes", "create", "--folder-id", "42", "--title", "Created", "--file", bodyPath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"note": map[string]any{"folder_id": float64(42), "title": "Created", "body": "from file"}}
	if !notesJSONEqual(payload, want) {
		t.Fatalf("payload = %#v", payload)
	}
	cached, err := notestore.New(dataDir).ReadNote(9)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Body != "from file" || cached.Meta.Title != "Created" {
		t.Fatalf("cached = %#v", cached)
	}
}

func notesJSONEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func TestNotesDeleteRemovesCacheAfterRemoteSuccess(t *testing.T) {
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notes/9" || r.Method != http.MethodDelete {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	configPath := writeNotesTestConfig(t, server.URL)
	dataDir := t.TempDir()
	if err := notestore.New(dataDir).StoreNote(notes.Note{ID: 9, Title: "Cached", Body: "body"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", configPath, "--data-dir", dataDir, "notes", "delete", "9"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected remote delete request")
	}
	if _, err := notestore.New(dataDir).ReadNote(9); err == nil {
		t.Fatal("expected cached note to be deleted")
	}
}

func writeNotesTestConfig(t *testing.T, baseURL string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := (&config.Config{BaseURL: baseURL, ClientID: "id", SecretKey: "secret"}).SaveTo(configPath); err != nil {
		t.Fatal(err)
	}
	_, tokenPath := config.Paths(configPath)
	if err := config.SaveTokenTo(tokenPath, &config.TokenCache{Token: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return configPath
}
