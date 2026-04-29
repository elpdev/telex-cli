package screens

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForEditedFileToSettleWaitsForDelayedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(path, []byte("Title: Old\n\nold"), 0o600); err != nil {
		t.Fatal(err)
	}
	initial, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = os.WriteFile(path, []byte("Title: New\n\nnew"), 0o600)
	}()

	started := time.Now()
	if err := waitForEditedFileToSettle(path, initial, 2*time.Second, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 150*time.Millisecond {
		t.Fatal("settle returned before delayed write")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "Title: New\n\nnew" {
		t.Fatalf("content = %q", content)
	}
}

func TestWaitForEditedFileToSettleAllowsUnchangedFileAfterGrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(path, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	initial, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	if err := waitForEditedFileToSettle(path, initial, time.Second, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 200*time.Millisecond {
		t.Fatal("settle returned before unchanged grace period")
	}
}
