package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

func TestDriveCommandExists(t *testing.T) {
	cmd := newRootCommand(buildInfo{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"drive", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestDriveListCommandReadsLocalMirror(t *testing.T) {
	dataDir := t.TempDir()
	store := drivestore.New(dataDir)
	if _, err := store.StoreFile("", drive.File{ID: 12, Filename: "remote.txt", ByteSize: 9}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand(buildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", dataDir, "drive", "ls"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "remote.txt") || !strings.Contains(got, "remote-only") {
		t.Fatalf("output = %q", got)
	}
}
