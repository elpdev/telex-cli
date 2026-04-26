package screens

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

func TestDriveScreenLoadsLocalMirror(t *testing.T) {
	store := drivestore.New(t.TempDir())
	if _, err := store.StoreFile("", drive.File{ID: 1, Filename: "remote.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewDrive(store, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	view := screen.View(80, 20)
	if !strings.Contains(view, "remote.txt") || !strings.Contains(view, "remote-only") {
		t.Fatalf("view = %q", view)
	}
}

func TestDriveScreenOpensFolder(t *testing.T) {
	store := drivestore.New(t.TempDir())
	folderPath, err := store.StoreFolder("", drive.Folder{ID: 1, Name: "Projects"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreFile(folderPath, drive.File{ID: 2, Filename: "plan.txt"}, []byte("body"), time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewDrive(store, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Drive)
	msg := cmd()
	updated, _ = screen.Update(msg)
	screen = updated.(Drive)
	if !strings.Contains(screen.View(80, 20), "plan.txt") {
		t.Fatalf("view = %q", screen.View(80, 20))
	}
}

func TestDriveScreenOpensCachedFile(t *testing.T) {
	store := drivestore.New(t.TempDir())
	path, err := store.StoreFile("", drive.File{ID: 2, Filename: "plan.txt"}, []byte("body"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var opened string
	screen := NewDrive(store, nil).WithActions(nil, func(path string) error { opened = path; return nil }, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Drive)
	if cmd == nil {
		t.Fatal("expected open command")
	}
	updated, _ = screen.Update(cmd())
	if opened != path {
		t.Fatalf("opened = %q, want %q", opened, path)
	}
}

func TestDriveScreenDownloadsRemoteOnlyFileThenOpens(t *testing.T) {
	store := drivestore.New(t.TempDir())
	path, err := store.StoreFile("", drive.File{ID: 2, Filename: "remote.txt"}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var opened string
	screen := NewDrive(store, nil).WithActions(func(ctx context.Context, meta drivestore.FileMeta) ([]byte, error) {
		if meta.RemoteID != 2 {
			t.Fatalf("remote id = %d", meta.RemoteID)
		}
		return []byte("downloaded"), nil
	}, func(path string) error { opened = path; return nil }, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Drive)
	updated, _ = screen.Update(cmd())
	if opened != path {
		t.Fatalf("opened = %q, want %q", opened, path)
	}
	meta, err := store.FileMetaForPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if !meta.LocalContentCached {
		t.Fatalf("meta = %#v", meta)
	}
}

func TestDriveScreenLocalFilterAndDetails(t *testing.T) {
	store := drivestore.New(t.TempDir())
	if _, err := store.StoreFile("", drive.File{ID: 1, Filename: "alpha.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreFile("", drive.File{ID: 2, Filename: "beta.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewDrive(store, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}))
	screen = updated.(Drive)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "b", Code: 'b'}))
	screen = updated.(Drive)
	view := screen.View(80, 20)
	if strings.Contains(view, "alpha.txt") || !strings.Contains(view, "beta.txt") {
		t.Fatalf("view = %q", view)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	screen = updated.(Drive)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "i", Code: 'i'}))
	screen = updated.(Drive)
	view = screen.View(80, 20)
	if !strings.Contains(view, "Details") || !strings.Contains(view, "Remote ID: 2") {
		t.Fatalf("view = %q", view)
	}
}

func TestDriveScreenUsesListNavigation(t *testing.T) {
	store := drivestore.New(t.TempDir())
	if _, err := store.StoreFile("", drive.File{ID: 1, Filename: "alpha.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreFile("", drive.File{ID: 2, Filename: "beta.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewDrive(store, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)

	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd}))
	if cmd != nil {
		t.Fatal("expected no command")
	}
	screen = updated.(Drive)
	if screen.index != len(screen.visibleEntries())-1 {
		t.Fatalf("index = %d, want last entry", screen.index)
	}
	entry, ok := screen.selectedEntry()
	if !ok || entry.Name != "beta.txt" {
		t.Fatalf("entry = %#v ok = %v", entry, ok)
	}
	if !strings.Contains(screen.View(80, 20), "> file  beta.txt") {
		t.Fatalf("view missing selected beta entry:\n%s", screen.View(80, 20))
	}
}

func TestDriveScreenUploadPickerSelectionInvokesUpload(t *testing.T) {
	store := drivestore.New(t.TempDir())
	source := t.TempDir() + "/upload.txt"
	if err := os.WriteFile(source, []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotPath string
	screen := NewDrive(store, nil).WithActions(nil, nil, func(ctx context.Context, path string, folderID *int64) error {
		gotPath = path
		if folderID != nil {
			t.Fatalf("folderID = %#v, want nil", folderID)
		}
		return nil
	}, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "u", Code: 'u'}))
	screen = updated.(Drive)
	if !screen.pickerOpen {
		t.Fatal("expected picker")
	}
	screen.picker = filepicker.New("", filepath.Dir(source), filepicker.ModeOpenFile)
	updated, _ = screen.Update(screen.picker.Init()())
	screen = updated.(Drive)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	screen = updated.(Drive)
	if cmd == nil {
		t.Fatal("expected upload command")
	}
	updated, _ = screen.Update(cmd())
	if gotPath != source {
		t.Fatalf("gotPath = %q, want %q", gotPath, source)
	}
}

func TestDriveScreenDownloadFailureShowsStatus(t *testing.T) {
	store := drivestore.New(t.TempDir())
	if _, err := store.StoreFile("", drive.File{ID: 2, Filename: "remote.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	screen := NewDrive(store, nil).WithActions(func(ctx context.Context, meta drivestore.FileMeta) ([]byte, error) {
		return nil, errors.New("network down")
	}, func(path string) error { return nil }, nil, nil, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	screen = updated.(Drive)
	updated, _ = screen.Update(cmd())
	screen = updated.(Drive)
	view := screen.View(80, 20)
	if !strings.Contains(view, "network down") {
		t.Fatalf("view = %q", view)
	}
}

func TestDriveScreenCreateFolderUnderCurrentFolder(t *testing.T) {
	store := drivestore.New(t.TempDir())
	folderPath, err := store.StoreFolder("", drive.Folder{ID: 7, Name: "Docs"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var got drive.FolderInput
	screen := NewDrive(store, nil).WithActions(nil, nil, nil, func(ctx context.Context, input drive.FolderInput) error { got = input; return nil }, nil, nil, nil, nil)
	updated, _ := screen.Update(screen.load(folderPath))
	screen = updated.(Drive)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "n", Code: 'n'}))
	screen = updated.(Drive)
	for _, r := range "New Docs" {
		updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
		screen = updated.(Drive)
	}
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	screen = updated.(Drive)
	if cmd == nil {
		t.Fatal("expected create command")
	}
	updated, _ = screen.Update(cmd())
	if got.ParentID == nil || *got.ParentID != 7 || got.Name != "New Docs" {
		t.Fatalf("input = %#v", got)
	}
}

func TestDriveScreenRenameFileAndFolder(t *testing.T) {
	store := drivestore.New(t.TempDir())
	folderPath, err := store.StoreFolder("", drive.Folder{ID: 7, Name: "Docs"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.StoreFile("", drive.File{ID: 9, Filename: "plan.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	var fileName, folderName string
	screen := NewDrive(store, nil).WithActions(nil, nil, nil, nil, func(ctx context.Context, id int64, input drive.FileInput) error {
		fileName = input.Filename
		return nil
	}, func(ctx context.Context, id int64, input drive.FolderInput) error {
		folderName = input.Name
		return nil
	}, nil, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "R", Code: 'R'}))
	screen = updated.(Drive)
	screen.promptInput = "Docs Renamed"
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	screen = updated.(Drive)
	updated, _ = screen.Update(cmd())
	if folderName != "Docs Renamed" {
		t.Fatalf("folderName = %q", folderName)
	}
	updated, _ = screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	screen.index = 1
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "R", Code: 'R'}))
	screen = updated.(Drive)
	screen.promptInput = "plan-renamed.txt"
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "enter"}))
	screen = updated.(Drive)
	updated, _ = screen.Update(cmd())
	if fileName != "plan-renamed.txt" || folderPath == "" {
		t.Fatalf("fileName = %q", fileName)
	}
}

func TestDriveScreenDeleteRequiresConfirmation(t *testing.T) {
	store := drivestore.New(t.TempDir())
	if _, err := store.StoreFile("", drive.File{ID: 9, Filename: "plan.txt"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	var deleted int64
	screen := NewDrive(store, nil).WithActions(nil, nil, nil, nil, nil, nil, func(ctx context.Context, id int64) error { deleted = id; return nil }, nil)
	updated, _ := screen.Update(screen.load(store.DriveRoot()))
	screen = updated.(Drive)
	updated, cmd := screen.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	screen = updated.(Drive)
	if cmd != nil || deleted != 0 || screen.confirm == "" {
		t.Fatalf("cmd=%v deleted=%d confirm=%q", cmd, deleted, screen.confirm)
	}
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "n", Code: 'n'}))
	screen = updated.(Drive)
	if cmd != nil || deleted != 0 {
		t.Fatalf("cmd=%v deleted=%d", cmd, deleted)
	}
	updated, _ = screen.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	screen = updated.(Drive)
	updated, cmd = screen.Update(tea.KeyPressMsg(tea.Key{Text: "y", Code: 'y'}))
	screen = updated.(Drive)
	if cmd == nil {
		t.Fatal("expected delete command")
	}
	updated, _ = screen.Update(cmd())
	if deleted != 9 {
		t.Fatalf("deleted = %d", deleted)
	}
}
