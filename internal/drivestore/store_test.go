package drivestore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/telex-cli/internal/drive"
)

func TestStoreMirrorsFoldersAndFiles(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	folderPath, err := store.StoreFolder("", drive.Folder{ID: 1, Name: "Invoices"}, syncedAt)
	if err != nil {
		t.Fatal(err)
	}
	filePath, err := store.StoreFile(folderPath, drive.File{ID: 2, Filename: "april.pdf", MIMEType: "application/pdf", ByteSize: 7, Downloadable: true, DownloadURL: "/api/v1/files/2/download"}, []byte("content"), syncedAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.DriveRoot(), "Invoices", "april.pdf")); err != nil {
		t.Fatal(err)
	}
	meta, err := ReadFileMeta(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if meta.RemoteID != 2 || !meta.LocalContentCached {
		t.Fatalf("meta = %#v", meta)
	}
	entries, err := store.List(folderPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "april.pdf" || !entries[0].Cached {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestMetadataOnlyFileIsListedWithoutContent(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.StoreFile("", drive.File{ID: 3, Filename: "remote.txt", ByteSize: 12}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "remote.txt" || entries[0].Cached {
		t.Fatalf("entries = %#v", entries)
	}
}
