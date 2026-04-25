package drivesync

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

func TestRunDownloadsContentInFullMode(t *testing.T) {
	store := drivestore.New(t.TempDir())
	service := drive.NewService(&syncFakeClient{})
	result, err := Run(context.Background(), store, service, config.DriveSyncFull)
	if err != nil {
		t.Fatal(err)
	}
	if result.Files != 1 || result.Folders != 1 || result.DownloadedFiles != 1 {
		t.Fatalf("result = %#v", result)
	}
	if body, err := os.ReadFile(filepath.Join(store.DriveRoot(), "Projects", "plan.txt")); err != nil || string(body) != "file body" {
		t.Fatalf("body=%q err=%v", string(body), err)
	}
}

func TestRunSkipsContentInMetadataOnlyMode(t *testing.T) {
	store := drivestore.New(t.TempDir())
	service := drive.NewService(&syncFakeClient{})
	result, err := Run(context.Background(), store, service, config.DriveSyncMetadataOnly)
	if err != nil {
		t.Fatal(err)
	}
	if result.Files != 1 || result.DownloadedFiles != 0 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(store.DriveRoot(), "Projects", "plan.txt")); !os.IsNotExist(err) {
		t.Fatalf("content stat err = %v", err)
	}
	entries, err := store.List(filepath.Join(store.DriveRoot(), "Projects"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Cached {
		t.Fatalf("entries = %#v", entries)
	}
}

type syncFakeClient struct{}

func (f *syncFakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, int, error) {
	switch path {
	case "/api/v1/folders":
		if query.Get("parent_id") == "root" {
			return []byte(`{"data":[{"id":1,"name":"Projects"}],"meta":{"page":1,"per_page":100,"total_count":1}}`), 200, nil
		}
		return []byte(`{"data":[],"meta":{"page":1,"per_page":100,"total_count":0}}`), 200, nil
	case "/api/v1/files":
		if query.Get("folder_id") == "1" {
			return []byte(`{"data":[{"id":2,"folder_id":1,"filename":"plan.txt","byte_size":9,"downloadable":true,"download_url":"/api/v1/files/2/download"}],"meta":{"page":1,"per_page":100,"total_count":1}}`), 200, nil
		}
		return []byte(`{"data":[],"meta":{"page":1,"per_page":100,"total_count":0}}`), 200, nil
	default:
		return nil, 404, fmt.Errorf("unexpected GET %s", path)
	}
}

func (f *syncFakeClient) Post(context.Context, string, any) ([]byte, int, error) {
	return nil, 500, nil
}
func (f *syncFakeClient) Patch(context.Context, string, any) ([]byte, int, error) {
	return nil, 500, nil
}
func (f *syncFakeClient) Delete(context.Context, string) (int, error) { return 204, nil }
func (f *syncFakeClient) Download(context.Context, string) ([]byte, string, error) {
	return []byte("file body"), "text/plain", nil
}
func (f *syncFakeClient) PutRaw(context.Context, string, map[string]string, io.Reader) (int, error) {
	return 204, nil
}
