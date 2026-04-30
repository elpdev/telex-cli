package drivestore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/mailstore"
)

const SchemaVersion = 1

type Store struct {
	Root string
}

type FolderMeta struct {
	SchemaVersion   int       `toml:"schema_version"`
	Kind            string    `toml:"kind"`
	RemoteID        int64     `toml:"remote_id"`
	ParentID        int64     `toml:"parent_id"`
	Name            string    `toml:"name"`
	Source          string    `toml:"source"`
	Provider        string    `toml:"provider"`
	ProviderID      string    `toml:"provider_id"`
	RemoteCreatedAt time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt time.Time `toml:"remote_updated_at"`
	SyncedAt        time.Time `toml:"synced_at"`
}

type FileMeta struct {
	SchemaVersion      int       `toml:"schema_version"`
	Kind               string    `toml:"kind"`
	RemoteID           int64     `toml:"remote_id"`
	FolderID           int64     `toml:"folder_id"`
	Filename           string    `toml:"filename"`
	MIMEType           string    `toml:"mime_type"`
	ByteSize           int64     `toml:"byte_size"`
	Source             string    `toml:"source"`
	Provider           string    `toml:"provider"`
	ProviderID         string    `toml:"provider_id"`
	DownloadURL        string    `toml:"download_url"`
	Downloadable       bool      `toml:"downloadable"`
	LocalContentCached bool      `toml:"local_content_cached"`
	RemoteCreatedAt    time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt    time.Time `toml:"remote_updated_at"`
	SyncedAt           time.Time `toml:"synced_at"`
}

type Entry struct {
	Name     string
	Path     string
	Kind     string
	Folder   *FolderMeta
	File     *FileMeta
	Cached   bool
	ByteSize int64
}

func New(root string) Store {
	return Store{Root: mailstore.RootOrDefault(root)}
}

func (s Store) DriveRoot() string { return filepath.Join(s.Root, "drive") }

func (s Store) EnsureRoot() error {
	return os.MkdirAll(s.DriveRoot(), 0o700)
}

func (s Store) StoreFolder(parentPath string, folder drive.Folder, syncedAt time.Time) (string, error) {
	if parentPath == "" {
		parentPath = s.DriveRoot()
	}
	path := s.folderPath(parentPath, folder)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", err
	}
	meta := FolderMeta{SchemaVersion: SchemaVersion, Kind: "folder", RemoteID: folder.ID, Name: folder.Name, Source: folder.Source, Provider: folder.Provider, ProviderID: folder.ProviderIdentifier, RemoteCreatedAt: folder.CreatedAt, RemoteUpdatedAt: folder.UpdatedAt, SyncedAt: syncedAt}
	if folder.ParentID != nil {
		meta.ParentID = *folder.ParentID
	}
	if err := writeTOML(filepath.Join(path, "meta.toml"), meta); err != nil {
		return "", err
	}
	return path, nil
}

func (s Store) StoreFile(parentPath string, file drive.File, content []byte, syncedAt time.Time) (string, error) {
	if parentPath == "" {
		parentPath = s.DriveRoot()
	}
	if err := os.MkdirAll(parentPath, 0o700); err != nil {
		return "", err
	}
	path := s.filePath(parentPath, file)
	if content != nil {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return "", err
		}
	}
	meta := FileMeta{SchemaVersion: SchemaVersion, Kind: "file", RemoteID: file.ID, Filename: file.Filename, MIMEType: file.MIMEType, ByteSize: file.ByteSize, Source: file.Source, Provider: file.Provider, ProviderID: file.ProviderIdentifier, DownloadURL: file.DownloadURL, Downloadable: file.Downloadable, LocalContentCached: content != nil, RemoteCreatedAt: file.CreatedAt, RemoteUpdatedAt: file.UpdatedAt, SyncedAt: syncedAt}
	if file.FolderID != nil {
		meta.FolderID = *file.FolderID
	}
	if err := writeTOML(fileMetaPath(path), meta); err != nil {
		return "", err
	}
	return path, nil
}

func (s Store) FolderMetaForPath(path string) (*FolderMeta, error) {
	return ReadFolderMeta(path)
}

func (s Store) FileMetaForPath(path string) (*FileMeta, error) {
	return ReadFileMeta(path)
}

func (s Store) CurrentFolderRemoteID(path string) (*int64, error) {
	if path == "" || filepath.Clean(path) == filepath.Clean(s.DriveRoot()) {
		return nil, nil
	}
	meta, err := ReadFolderMeta(path)
	if err != nil {
		return nil, err
	}
	return &meta.RemoteID, nil
}

func (s Store) WriteFileContent(path string, meta FileMeta, content []byte, syncedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	meta.LocalContentCached = true
	meta.SyncedAt = syncedAt
	return writeTOML(fileMetaPath(path), meta)
}

func (s Store) AllFiles() ([]FileMeta, error) {
	root := s.DriveRoot()
	out := []FileMeta{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".meta.toml") {
			return nil
		}
		var meta FileMeta
		if _, err := toml.DecodeFile(path, &meta); err != nil {
			return nil
		}
		if meta.Kind != "file" {
			return nil
		}
		out = append(out, meta)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func (s Store) List(path string) ([]Entry, error) {
	if path == "" {
		path = s.DriveRoot()
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []Entry{}
	for _, entry := range entries {
		name := entry.Name()
		if name == "meta.toml" || strings.HasSuffix(name, ".meta.toml") {
			continue
		}
		fullPath := filepath.Join(path, name)
		if entry.IsDir() {
			meta, err := ReadFolderMeta(fullPath)
			if err != nil {
				continue
			}
			out = append(out, Entry{Name: meta.Name, Path: fullPath, Kind: "folder", Folder: meta})
			continue
		}
		meta, err := ReadFileMeta(fullPath)
		if err != nil {
			continue
		}
		out = append(out, Entry{Name: meta.Filename, Path: fullPath, Kind: "file", File: meta, Cached: meta.LocalContentCached, ByteSize: meta.ByteSize})
	}
	addMetadataOnlyFiles(path, &out)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "folder"
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s Store) PruneMissing(keepFolders, keepFiles map[string]bool) error {
	root := s.DriveRoot()
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if keepFolders == nil {
		keepFolders = map[string]bool{}
	}
	if keepFiles == nil {
		keepFiles = map[string]bool{}
	}
	keepFolders[filepath.Clean(root)] = true
	folders := []string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		clean := filepath.Clean(path)
		if entry.IsDir() {
			if clean != filepath.Clean(root) {
				if _, err := os.Stat(filepath.Join(path, "meta.toml")); err == nil {
					folders = append(folders, clean)
				}
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".meta.toml") {
			return nil
		}
		filePath := strings.TrimSuffix(path, ".meta.toml")
		if keepFiles[filepath.Clean(filePath)] {
			return nil
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	sort.Slice(folders, func(i, j int) bool { return len(folders[i]) > len(folders[j]) })
	for _, folder := range folders {
		if keepFolders[folder] {
			continue
		}
		if err := os.RemoveAll(folder); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func ReadFolderMeta(path string) (*FolderMeta, error) {
	var meta FolderMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func ReadFileMeta(path string) (*FileMeta, error) {
	var meta FileMeta
	if _, err := toml.DecodeFile(fileMetaPath(path), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func fileMetaPath(path string) string { return path + ".meta.toml" }

func addMetadataOnlyFiles(path string, entries *[]Entry) {
	metas, _ := filepath.Glob(filepath.Join(path, "*.meta.toml"))
	seen := map[string]bool{}
	for _, entry := range *entries {
		seen[fileMetaPath(entry.Path)] = true
	}
	for _, metaPath := range metas {
		if seen[metaPath] {
			continue
		}
		var meta FileMeta
		if _, err := toml.DecodeFile(metaPath, &meta); err != nil || meta.Kind != "file" {
			continue
		}
		filePath := strings.TrimSuffix(metaPath, ".meta.toml")
		*entries = append(*entries, Entry{Name: meta.Filename, Path: filePath, Kind: "file", File: &meta, Cached: false, ByteSize: meta.ByteSize})
	}
}

func (s Store) folderPath(parentPath string, folder drive.Folder) string {
	base := safePathName(folder.Name, "folder")
	path := filepath.Join(parentPath, base)
	meta, err := ReadFolderMeta(path)
	if err == nil && meta.RemoteID == folder.ID {
		return path
	}
	if err == nil || pathExists(path) {
		return filepath.Join(parentPath, withRemoteIDSuffix(base, folder.ID))
	}
	return path
}

func (s Store) filePath(parentPath string, file drive.File) string {
	base := safePathName(file.Filename, "file")
	path := filepath.Join(parentPath, base)
	meta, err := ReadFileMeta(path)
	if err == nil && meta.RemoteID == file.ID {
		return path
	}
	if err == nil || pathExists(path) || pathExists(fileMetaPath(path)) {
		return filepath.Join(parentPath, withRemoteIDSuffix(base, file.ID))
	}
	return path
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func withRemoteIDSuffix(name string, remoteID int64) string {
	if remoteID == 0 {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		base = strings.TrimPrefix(name, ext)
	}
	return fmt.Sprintf("%s-%d%s", base, remoteID, ext)
}

func safePathName(value, fallback string) string {
	value = strings.TrimSpace(filepath.Base(value))
	value = strings.Trim(value, ".")
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == ' ' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.TrimSpace(strings.Trim(b.String(), "-"))
	if name == "" {
		return fallback
	}
	return name
}

func writeTOML(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(value); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
