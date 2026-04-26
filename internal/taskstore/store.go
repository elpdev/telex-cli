package taskstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elpdev/telex-cli/internal/mailstore"
	"github.com/elpdev/telex-cli/internal/tasks"
)

const SchemaVersion = 1

type Store struct {
	Root string
}

type StoreMeta struct {
	SchemaVersion    int       `toml:"schema_version"`
	RootFolderID     int64     `toml:"root_folder_id"`
	ProjectsFolderID int64     `toml:"projects_folder_id"`
	SyncedAt         time.Time `toml:"synced_at"`
}

type ProjectMeta struct {
	SchemaVersion   int       `toml:"schema_version"`
	RemoteID        int64     `toml:"remote_id"`
	UserID          int64     `toml:"user_id"`
	ParentID        int64     `toml:"parent_id"`
	Name            string    `toml:"name"`
	Source          string    `toml:"source"`
	ManifestID      int64     `toml:"manifest_id"`
	BoardID         int64     `toml:"board_id"`
	RemoteCreatedAt time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt time.Time `toml:"remote_updated_at"`
	SyncedAt        time.Time `toml:"synced_at"`
}

type FileMeta struct {
	SchemaVersion   int       `toml:"schema_version"`
	RemoteID        int64     `toml:"remote_id"`
	UserID          int64     `toml:"user_id"`
	ProjectID       int64     `toml:"project_id"`
	FolderID        int64     `toml:"folder_id"`
	Title           string    `toml:"title"`
	Filename        string    `toml:"filename"`
	MIMEType        string    `toml:"mime_type"`
	RemoteCreatedAt time.Time `toml:"remote_created_at"`
	RemoteUpdatedAt time.Time `toml:"remote_updated_at"`
	SyncedAt        time.Time `toml:"synced_at"`
}

type CachedProject struct {
	Meta ProjectMeta
	Path string
}

type CachedBoard struct {
	Meta    FileMeta
	Body    string
	Columns []tasks.BoardColumn
	Path    string
}

type CachedCard struct {
	Meta FileMeta
	Body string
	Path string
}

func New(root string) Store {
	if root == "" {
		root = mailstore.DefaultRoot()
	}
	return Store{Root: root}
}

func (s Store) TasksRoot() string { return filepath.Join(s.Root, "tasks") }

func (s Store) EnsureRoot() error {
	for _, dir := range []string{s.TasksRoot(), s.projectsRoot(), s.cardsRoot()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) StoreWorkspace(workspace *tasks.Workspace, syncedAt time.Time) error {
	if workspace == nil {
		return nil
	}
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	meta := StoreMeta{SchemaVersion: SchemaVersion, RootFolderID: workspace.RootFolder.ID, ProjectsFolderID: workspace.ProjectsFolder.ID, SyncedAt: syncedAt}
	return writeTOML(filepath.Join(s.TasksRoot(), "meta.toml"), meta)
}

func (s Store) StoreProject(project tasks.Project, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.projectPath(project.ID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	meta := ProjectMeta{SchemaVersion: SchemaVersion, RemoteID: project.ID, UserID: project.UserID, Name: project.Name, Source: project.Source, RemoteCreatedAt: project.CreatedAt, RemoteUpdatedAt: project.UpdatedAt, SyncedAt: syncedAt}
	if project.ParentID != nil {
		meta.ParentID = *project.ParentID
	}
	if project.Manifest != nil {
		meta.ManifestID = project.Manifest.ID
	}
	if project.Board != nil {
		meta.BoardID = project.Board.ID
	}
	return writeTOML(filepath.Join(path, "meta.toml"), meta)
}

func (s Store) StoreBoard(projectID int64, board tasks.Board, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.boardPath(projectID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	meta := fileMeta(projectID, board.TaskFile, syncedAt)
	if err := writeTOML(s.boardMetaPath(projectID), meta); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(board.Body), 0o600)
}

func (s Store) StoreCard(projectID int64, card tasks.Card, syncedAt time.Time) error {
	if err := s.EnsureRoot(); err != nil {
		return err
	}
	path := s.cardPath(projectID, card.ID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if err := writeTOML(filepath.Join(path, "meta.toml"), fileMeta(projectID, card.TaskFile, syncedAt)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "body.md"), []byte(card.Body), 0o600)
}

func (s Store) DeleteProject(id int64) error {
	err := os.RemoveAll(s.projectPath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) DeleteCard(projectID, id int64) error {
	err := os.RemoveAll(s.cardPath(projectID, id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s Store) ListProjects() ([]CachedProject, error) {
	entries, err := os.ReadDir(s.projectsRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CachedProject{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		project, err := s.ReadProjectPath(filepath.Join(s.projectsRoot(), entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, *project)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Meta.Name) < strings.ToLower(out[j].Meta.Name) })
	return out, nil
}

func (s Store) ReadProject(id int64) (*CachedProject, error) {
	return s.ReadProjectPath(s.projectPath(id))
}

func (s Store) ReadProjectPath(path string) (*CachedProject, error) {
	var meta ProjectMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	return &CachedProject{Meta: meta, Path: path}, nil
}

func (s Store) ReadBoard(projectID int64) (*CachedBoard, error) {
	var meta FileMeta
	if _, err := toml.DecodeFile(s.boardMetaPath(projectID), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(s.boardPath(projectID))
	if err != nil {
		return nil, err
	}
	return &CachedBoard{Meta: meta, Body: string(body), Columns: ParseBoard(string(body), s.cardTitleByFilename(projectID)), Path: s.boardPath(projectID)}, nil
}

func (s Store) ListCards(projectID int64) ([]CachedCard, error) {
	entries, err := os.ReadDir(s.projectCardsRoot(projectID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := []CachedCard{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		card, err := s.ReadCardPath(filepath.Join(s.projectCardsRoot(projectID), entry.Name()))
		if err != nil {
			continue
		}
		out = append(out, *card)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Meta.Title) < strings.ToLower(out[j].Meta.Title) })
	return out, nil
}

func (s Store) ReadCard(projectID, id int64) (*CachedCard, error) {
	return s.ReadCardPath(s.cardPath(projectID, id))
}

func (s Store) ReadCardPath(path string) (*CachedCard, error) {
	var meta FileMeta
	if _, err := toml.DecodeFile(filepath.Join(path, "meta.toml"), &meta); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(path, "body.md"))
	if err != nil {
		return nil, err
	}
	return &CachedCard{Meta: meta, Body: string(body), Path: path}, nil
}

func ParseBoard(markdown string, titles map[string]tasks.TaskFile) []tasks.BoardColumn {
	columns := []tasks.BoardColumn{}
	current := -1
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			columns = append(columns, tasks.BoardColumn{Name: strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))})
			current = len(columns) - 1
			continue
		}
		if current < 0 || !strings.HasPrefix(trimmed, "- [[") || !strings.HasSuffix(trimmed, "]]") {
			continue
		}
		path := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "- [["), "]]"))
		title := strings.TrimSuffix(filepath.Base(path), ".md")
		link := tasks.BoardCardLink{Path: path, Title: strings.ReplaceAll(strings.ReplaceAll(title, "-", " "), "_", " "), Missing: true}
		if card, ok := titles[path]; ok {
			stored := card
			link.Title = stored.Title
			link.Card = &stored
			link.Missing = false
		}
		columns[current].Cards = append(columns[current].Cards, link)
	}
	return columns
}

func fileMeta(projectID int64, file tasks.TaskFile, syncedAt time.Time) FileMeta {
	return FileMeta{SchemaVersion: SchemaVersion, RemoteID: file.ID, UserID: file.UserID, ProjectID: projectID, FolderID: file.FolderID, Title: file.Title, Filename: file.Filename, MIMEType: file.MIMEType, RemoteCreatedAt: file.CreatedAt, RemoteUpdatedAt: file.UpdatedAt, SyncedAt: syncedAt}
}

func (s Store) cardTitleByFilename(projectID int64) map[string]tasks.TaskFile {
	cards, _ := s.ListCards(projectID)
	out := map[string]tasks.TaskFile{}
	for _, card := range cards {
		out["cards/"+card.Meta.Filename] = tasks.TaskFile{ID: card.Meta.RemoteID, UserID: card.Meta.UserID, FolderID: card.Meta.FolderID, Title: card.Meta.Title, Filename: card.Meta.Filename, MIMEType: card.Meta.MIMEType, CreatedAt: card.Meta.RemoteCreatedAt, UpdatedAt: card.Meta.RemoteUpdatedAt}
	}
	return out
}

func (s Store) projectsRoot() string { return filepath.Join(s.TasksRoot(), "projects") }

func (s Store) cardsRoot() string { return filepath.Join(s.TasksRoot(), "cards") }

func (s Store) projectPath(id int64) string {
	return filepath.Join(s.projectsRoot(), fmt.Sprintf("%d", id))
}

func (s Store) boardPath(projectID int64) string {
	return filepath.Join(s.projectPath(projectID), "board.md")
}

func (s Store) boardMetaPath(projectID int64) string {
	return filepath.Join(s.projectPath(projectID), "board.toml")
}

func (s Store) projectCardsRoot(projectID int64) string {
	return filepath.Join(s.cardsRoot(), fmt.Sprintf("%d", projectID))
}

func (s Store) cardPath(projectID, id int64) string {
	return filepath.Join(s.projectCardsRoot(projectID), fmt.Sprintf("%d", id))
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
