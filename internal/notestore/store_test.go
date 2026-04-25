package notestore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/notes"
)

func TestStoreReadNoteBodyAndMetadata(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	folderID := int64(1)
	updatedAt := syncedAt.Add(-time.Hour)
	if err := store.StoreNote(notes.Note{ID: 9, UserID: 3, FolderID: &folderID, Title: "Plan", Filename: "plan.md", MIMEType: "text/markdown", Body: "# Plan", UpdatedAt: updatedAt}, syncedAt); err != nil {
		t.Fatal(err)
	}
	cached, err := store.ReadNote(9)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Meta.RemoteID != 9 || cached.Meta.FolderID != 1 || cached.Meta.Title != "Plan" || cached.Meta.RemoteUpdatedAt != updatedAt || cached.Body != "# Plan" {
		t.Fatalf("cached = %#v", cached)
	}
	if _, err := os.Stat(filepath.Join(store.NotesRoot(), "notes", "9", "body.md")); err != nil {
		t.Fatal(err)
	}
}

func TestStoreTreeAndRebuildFolderTree(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	rootID := int64(1)
	tree := &notes.FolderTree{
		FolderSummary:    notes.FolderSummary{ID: rootID, Name: "Notes"},
		NoteCount:        2,
		ChildFolderCount: 2,
		Children: []notes.FolderTree{
			{FolderSummary: notes.FolderSummary{ID: 3, ParentID: &rootID, Name: "Zeta"}, NoteCount: 1},
			{FolderSummary: notes.FolderSummary{ID: 2, ParentID: &rootID, Name: "Alpha"}, NoteCount: 4},
		},
	}
	if err := store.StoreTree(tree, syncedAt); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := store.FolderTree()
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.ID != 1 || rebuilt.Name != "Notes" || rebuilt.NoteCount != 2 || len(rebuilt.Children) != 2 {
		t.Fatalf("tree = %#v", rebuilt)
	}
	if rebuilt.Children[0].Name != "Alpha" || rebuilt.Children[1].Name != "Zeta" {
		t.Fatalf("children not sorted: %#v", rebuilt.Children)
	}
	if rebuilt.Children[0].ParentID == nil || *rebuilt.Children[0].ParentID != 1 {
		t.Fatalf("child parent = %#v", rebuilt.Children[0].ParentID)
	}
}

func TestListNotesUsesRootFolderAndStableTitleOrdering(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	rootID := int64(1)
	otherID := int64(2)
	if err := store.StoreTree(&notes.FolderTree{FolderSummary: notes.FolderSummary{ID: rootID, Name: "Notes"}}, syncedAt); err != nil {
		t.Fatal(err)
	}
	for _, note := range []notes.Note{
		{ID: 11, FolderID: &rootID, Title: "beta", Body: "second"},
		{ID: 10, FolderID: &rootID, Title: "Alpha", Body: "first"},
		{ID: 12, FolderID: &otherID, Title: "Other", Body: "other"},
	} {
		if err := store.StoreNote(note, syncedAt); err != nil {
			t.Fatal(err)
		}
	}
	cached, err := store.ListNotes(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != 2 || cached[0].Meta.RemoteID != 10 || cached[1].Meta.RemoteID != 11 {
		t.Fatalf("cached = %#v", cached)
	}
	if cached[0].Body != "first" || cached[1].Body != "second" {
		t.Fatalf("cached bodies = %#v", cached)
	}
}

func TestDeleteNoteRemovesCache(t *testing.T) {
	store := New(t.TempDir())
	if err := store.StoreNote(notes.Note{ID: 9, Title: "Plan", Body: "body"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteNote(9); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadNote(9); !os.IsNotExist(err) {
		t.Fatalf("ReadNote err = %v", err)
	}
}
