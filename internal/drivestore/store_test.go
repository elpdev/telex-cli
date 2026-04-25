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

func TestStoreAvoidsDuplicateRemoteNameCollisions(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	first, err := store.StoreFile("", drive.File{ID: 1, Filename: "same.txt"}, []byte("one"), syncedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.StoreFile("", drive.File{ID: 2, Filename: "same.txt"}, []byte("two"), syncedAt)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "same.txt" || filepath.Base(second) != "same-2.txt" {
		t.Fatalf("paths = %q %q", first, second)
	}
	body, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "one" {
		t.Fatalf("first body = %q", body)
	}
	entries, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].File.RemoteID == entries[1].File.RemoteID {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestStoreHelpersReadMetadataAndWriteFileContent(t *testing.T) {
	store := New(t.TempDir())
	syncedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	folderPath, err := store.StoreFolder("", drive.Folder{ID: 7, Name: "Docs"}, syncedAt)
	if err != nil {
		t.Fatal(err)
	}
	filePath, err := store.StoreFile(folderPath, drive.File{ID: 9, Filename: "remote.txt"}, nil, syncedAt)
	if err != nil {
		t.Fatal(err)
	}
	folderID, err := store.CurrentFolderRemoteID(folderPath)
	if err != nil {
		t.Fatal(err)
	}
	if folderID == nil || *folderID != 7 {
		t.Fatalf("folderID = %#v", folderID)
	}
	meta, err := store.FileMetaForPath(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteFileContent(filePath, *meta, []byte("cached"), syncedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	readBack, err := store.FileMetaForPath(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !readBack.LocalContentCached {
		t.Fatalf("meta = %#v", readBack)
	}
	body, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "cached" {
		t.Fatalf("body = %q", body)
	}
}
