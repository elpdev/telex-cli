package screens

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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
