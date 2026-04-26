package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elpdev/telex-cli/internal/taskstore"
)

func TestTasksCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"tasks", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTasksSyncStoresProjectsBoardsAndCards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tasks/workspace":
			_, _ = w.Write([]byte(`{"data":{"root_folder":{"id":1,"name":"Tasks"},"projects_folder":{"id":2,"name":"Projects"},"projects":[{"id":4,"name":"Website"}]}}`))
		case "/api/v1/tasks/projects":
			_, _ = w.Write([]byte(`{"data":[{"id":4,"name":"Website"}],"meta":{"page":1,"per_page":100,"total_count":1}}`))
		case "/api/v1/tasks/projects/4":
			_, _ = w.Write([]byte(`{"data":{"id":4,"name":"Website","board":{"id":5,"title":"Board","filename":"board.md"},"cards":[{"id":9,"folder_id":7,"title":"Homepage","filename":"Homepage.md","body":"# Homepage"}]}}`))
		case "/api/v1/tasks/projects/4/board":
			_, _ = w.Write([]byte(`{"data":{"id":5,"title":"Board","filename":"board.md","body":"# Website\n\n## Todo\n- [[cards/Homepage.md]]\n","columns":[{"name":"Todo","cards":[{"path":"cards/Homepage.md","title":"Homepage","card":{"id":9,"title":"Homepage","filename":"Homepage.md"},"missing":false}]}]}}`))
		case "/api/v1/tasks/projects/4/cards":
			_, _ = w.Write([]byte(`{"data":[{"id":9,"folder_id":7,"title":"Homepage","filename":"Homepage.md","body":"# Homepage"}],"meta":{"page":1,"per_page":100,"total_count":1}}`))
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
	cmd.SetArgs([]string{"--config", configPath, "--data-dir", dataDir, "tasks", "sync"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Synced 1 project(s), 1 board(s), 1 card(s).") {
		t.Fatalf("output = %q", out.String())
	}
	card, err := taskstore.New(dataDir).ReadCard(4, 9)
	if err != nil {
		t.Fatal(err)
	}
	if card.Body != "# Homepage" || card.Meta.Title != "Homepage" {
		t.Fatalf("card = %#v", card)
	}
}
