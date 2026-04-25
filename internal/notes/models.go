package notes

import "time"

type ListParams struct {
	Page    int
	PerPage int
}

type ListNotesParams struct {
	ListParams
	FolderID *int64
	Sort     string
}

type Note struct {
	ID        int64         `json:"id"`
	UserID    int64         `json:"user_id"`
	FolderID  *int64        `json:"folder_id"`
	Title     string        `json:"title"`
	Filename  string        `json:"filename"`
	MIMEType  string        `json:"mime_type"`
	Folder    FolderSummary `json:"folder"`
	Body      string        `json:"body"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
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

type FolderTree struct {
	FolderSummary
	NoteCount        int          `json:"note_count"`
	ChildFolderCount int          `json:"child_folder_count"`
	Children         []FolderTree `json:"children"`
}

type NoteInput struct {
	FolderID *int64
	Title    string
	Body     string
}
