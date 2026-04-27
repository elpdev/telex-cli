package notestore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/notes"
)

const SchemaVersion = 1

type Store struct {
	Root string
}

type StoreMeta struct {
	SchemaVersion int       `toml:"schema_version"`
	RootFolderID  int64     `toml:"root_folder_id"`
	SyncedAt      time.Time `toml:"synced_at"`
}

type FolderMeta struct {
	SchemaVersion    int       `toml:"schema_version"`
	RemoteID         int64     `toml:"remote_id"`
	ParentID         int64     `toml:"parent_id"`
	Name             string    `toml:"name"`
	Source           string    `toml:"source"`
	NoteCount        int       `toml:"note_count"`
	ChildFolderCount int       `toml:"child_folder_count"`
	RemoteCreatedAt  time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt  time.Time `toml:"remote_updated_at"`
	SyncedAt         time.Time `toml:"synced_at"`
}

type NoteMeta struct {
	SchemaVersion   int       `toml:"schema_version"`
	RemoteID        int64     `toml:"remote_id"`
	UserID          int64     `toml:"user_id"`
	FolderID        int64     `toml:"folder_id"`
	Title           string    `toml:"title"`
	Filename        string    `toml:"filename"`
	MIMEType        string    `toml:"mime_type"`
	RemoteCreatedAt time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt time.Time `toml:"remote_updated_at"`
	SyncedAt        time.Time `toml:"synced_at"`
}

type CachedNote struct {
	Meta NoteMeta
	Body string
	Path string
}

func New(root string) Store {
	return Store{Root: mailstore.RootOrDefault(root)}
}

func (s Store) NotesRoot() string { return filepath.Join(s.Root, "notes") }

func (s Store) EnsureRoot() error {
	for _, dir := range []string{s.NotesRoot(), s.foldersRoot(), s.notesRoot()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) StoreTree(tree *notes.FolderTree, syncedAt time.Time) error {
	if tree == nil {
		return fmt.Errorf("notes folder tree is required")
	}
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	meta := StoreMeta{SchemaVersion: SchemaVersion, RootFolderID: tree.ID, SyncedAt: syncedAt}
	if err := writeTOML(filepath.Join(s.NotesRoot(), "meta.toml"), meta); err != nil {
		return err
	}
	return s.storeFolderTree(*tree, syncedAt)
}

func (s Store) StoreNote(note notes.Note, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.notePath(note.ID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	meta := NoteMeta{SchemaVersion: SchemaVersion, RemoteID: note.ID, UserID: note.UserID, Title: note.Title, Filename: note.Filename, MIMEType: note.MIMEType, RemoteCreatedAt: note.CreatedAt, RemoteUpdatedAt: note.UpdatedAt, SyncedAt: syncedAt}
	if note.FolderID != nil {
		meta.FolderID = *note.FolderID
	}
	if err := writeTOML(filepath.Join(path, "meta.toml"), meta); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "body.md"), []byte(note.Body), 0o600)
}

func (s Store) ListNotes(folderID int64) ([]CachedNote, error) {
	if folderID == 0 {
		meta, err := s.readStoreMeta()
		if err == nil {
			folderID = meta.RootFolderID
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	entries, err := os.ReadDir(s.notesRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CachedNote{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cached, err := s.ReadNotePath(filepath.Join(s.notesRoot(), entry.Name()))
		if err != nil {
			continue
		}
		if cached.Meta.FolderID == folderID {
			out = append(out, *cached)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.ToLower(out[i].Meta.Title)
		right := strings.ToLower(out[j].Meta.Title)
		if left == right {
			return out[i].Meta.RemoteID < out[j].Meta.RemoteID
		}
		return left < right
	})
	return out, nil
}

func (s Store) ReadNote(id int64) (*CachedNote, error) {
	return s.ReadNotePath(s.notePath(id))
}

func (s Store) ReadNotePath(path string) (*CachedNote, error) {
	var meta NoteMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(path, "body.md"))
	if err != nil {
		return nil, err
	}
	return &CachedNote{Meta: meta, Body: string(body), Path: path}, nil
}

func (s Store) DeleteNote(id int64) error {
	err := os.RemoveAll(s.notePath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) AllNotes() ([]CachedNote, error) {
	entries, err := os.ReadDir(s.notesRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CachedNote{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cached, err := s.ReadNotePath(filepath.Join(s.notesRoot(), entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, *cached)
	}
	return out, nil
}

func (s Store) Counts() (totalNotes, totalFolders int, err error) {
	tree, err := s.FolderTree()
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if tree == nil {
		return 0, 0, nil
	}
	totalNotes, totalFolders = countTree(tree)
	return totalNotes, totalFolders, nil
}

func countTree(t *notes.FolderTree) (n, f int) {
	if t == nil {
		return 0, 0
	}
	n = t.NoteCount
	f = 1
	for i := range t.Children {
		cn, cf := countTree(&t.Children[i])
		n += cn
		f += cf
	}
	return
}

func (s Store) FolderTree() (*notes.FolderTree, error) {
	meta, err := s.readStoreMeta()
	if err != nil {
		return nil, err
	}
	folders, err := s.readFolders()
	if err != nil {
		return nil, err
	}
	byParent := map[int64][]FolderMeta{}
	byID := map[int64]FolderMeta{}
	for _, folder := range folders {
		byID[folder.RemoteID] = folder
		byParent[folder.ParentID] = append(byParent[folder.ParentID], folder)
	}
	root, ok := byID[meta.RootFolderID]
	if !ok {
		return nil, fmt.Errorf("notes root folder %d not found", meta.RootFolderID)
	}
	tree := folderMetaToTree(root)
	attachChildren(&tree, byParent)
	return &tree, nil
}

func (s Store) storeFolderTree(tree notes.FolderTree, syncedAt time.Time) error {
	meta := FolderMeta{SchemaVersion: SchemaVersion, RemoteID: tree.ID, Name: tree.Name, Source: tree.Source, NoteCount: tree.NoteCount, ChildFolderCount: tree.ChildFolderCount, RemoteCreatedAt: tree.CreatedAt, RemoteUpdatedAt: tree.UpdatedAt, SyncedAt: syncedAt}
	if tree.ParentID != nil {
		meta.ParentID = *tree.ParentID
	}
	if err := writeTOML(s.folderPath(tree.ID), meta); err != nil {
		return err
	}
	for _, child := range tree.Children {
		if err := s.storeFolderTree(child, syncedAt); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) readStoreMeta() (*StoreMeta, error) {
	var meta StoreMeta
	if _, err := toml.DecodeFile(filepath.Join(s.NotesRoot(), "meta.toml"), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s Store) readFolders() ([]FolderMeta, error) {
	entries, err := os.ReadDir(s.foldersRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	folders := []FolderMeta{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		var meta FolderMeta
		if _, err := toml.DecodeFile(filepath.Join(s.foldersRoot(), entry.Name()), &meta); err != nil {
			return nil, err
		}
		folders = append(folders, meta)
	}
	return folders, nil
}

func folderMetaToTree(meta FolderMeta) notes.FolderTree {
	return notes.FolderTree{FolderSummary: notes.FolderSummary{ID: meta.RemoteID, ParentID: parentPtr(meta.ParentID), Name: meta.Name, Source: meta.Source, CreatedAt: meta.RemoteCreatedAt, UpdatedAt: meta.RemoteUpdatedAt}, NoteCount: meta.NoteCount, ChildFolderCount: meta.ChildFolderCount}
}

func attachChildren(tree *notes.FolderTree, byParent map[int64][]FolderMeta) {
	children := byParent[tree.ID]
	sort.Slice(children, func(i, j int) bool { return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name) })
	for _, child := range children {
		childTree := folderMetaToTree(child)
		attachChildren(&childTree, byParent)
		tree.Children = append(tree.Children, childTree)
	}
}

func parentPtr(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func (s Store) foldersRoot() string { return filepath.Join(s.NotesRoot(), "folders") }

func (s Store) notesRoot() string { return filepath.Join(s.NotesRoot(), "notes") }

func (s Store) folderPath(id int64) string {
	return filepath.Join(s.foldersRoot(), fmt.Sprintf("%d.toml", id))
}

func (s Store) notePath(id int64) string { return filepath.Join(s.notesRoot(), fmt.Sprintf("%d", id)) }

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
