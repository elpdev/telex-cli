package tasks

import "time"

type ListParams struct {
	Page         int
	PerPage      int
	UpdatedSince string
}

type Workspace struct {
	RootFolder     FolderSummary    `json:"root_folder"`
	ProjectsFolder FolderSummary    `json:"projects_folder"`
	Projects       []ProjectSummary `json:"projects"`
}

type FolderSummary struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"user_id"`
	ParentID  *int64         `json:"parent_id"`
	Name      string         `json:"name"`
	Source    string         `json:"source"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ProjectSummary struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"user_id"`
	ParentID  *int64         `json:"parent_id"`
	Name      string         `json:"name"`
	Source    string         `json:"source"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type Project struct {
	ProjectSummary
	Manifest *TaskFile `json:"manifest"`
	Board    *TaskFile `json:"board"`
	Cards    []Card    `json:"cards"`
}

type TaskFile struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"user_id"`
	FolderID  int64          `json:"folder_id"`
	Title     string         `json:"title"`
	Filename  string         `json:"filename"`
	MIMEType  string         `json:"mime_type"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type Board struct {
	TaskFile
	Body    string        `json:"body"`
	Columns []BoardColumn `json:"columns"`
}

type BoardColumn struct {
	Name  string          `json:"name"`
	Cards []BoardCardLink `json:"cards"`
}

type BoardCardLink struct {
	Path    string    `json:"path"`
	Title   string    `json:"title"`
	Card    *TaskFile `json:"card"`
	Missing bool      `json:"missing"`
}

type Card struct {
	TaskFile
	Body string `json:"body"`
}

type ProjectInput struct {
	Name string
}

type BoardInput struct {
	Body string
}

type CardInput struct {
	Title string
	Body  string
}
