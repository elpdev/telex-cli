package drive

import "time"

type ListParams struct {
	Page    int
	PerPage int
}

type ListFoldersParams struct {
	ListParams
	ParentID *int64
	Root     bool
	Query    string
	Sort     string
}

type ListFilesParams struct {
	ListParams
	FolderID *int64
	Root     bool
	Query    string
	Sort     string
}

type Folder struct {
	ID                 int64          `json:"id"`
	UserID             int64          `json:"user_id"`
	ParentID           *int64         `json:"parent_id"`
	Name               string         `json:"name"`
	Source             string         `json:"source"`
	Provider           string         `json:"provider"`
	ProviderIdentifier string         `json:"provider_identifier"`
	Metadata           map[string]any `json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

type AlbumSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type File struct {
	ID                  int64          `json:"id"`
	UserID              int64          `json:"user_id"`
	FolderID            *int64         `json:"folder_id"`
	ActiveStorageBlobID *int64         `json:"active_storage_blob_id"`
	Filename            string         `json:"filename"`
	MIMEType            string         `json:"mime_type"`
	ByteSize            int64          `json:"byte_size"`
	Source              string         `json:"source"`
	Provider            string         `json:"provider"`
	ProviderIdentifier  string         `json:"provider_identifier"`
	ProviderCreatedAt   *time.Time     `json:"provider_created_at"`
	ProviderUpdatedAt   *time.Time     `json:"provider_updated_at"`
	Metadata            map[string]any `json:"metadata"`
	DriveAlbumIDs       []int64        `json:"drive_album_ids"`
	DriveAlbums         []AlbumSummary `json:"drive_albums"`
	LocalBlob           bool           `json:"local_blob"`
	Downloadable        bool           `json:"downloadable"`
	ImageMetadata       map[string]any `json:"image_metadata"`
	DownloadURL         string         `json:"download_url"`
	UploadURL           string         `json:"upload_url"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type DirectUpload struct {
	SignedID     string             `json:"signed_id"`
	Filename     string             `json:"filename"`
	ByteSize     int64              `json:"byte_size"`
	Checksum     string             `json:"checksum"`
	ContentType  string             `json:"content_type"`
	Metadata     map[string]any     `json:"metadata"`
	DirectUpload DirectUploadTarget `json:"direct_upload"`
}

type DirectUploadTarget struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type FileInput struct {
	FolderID     *int64
	Filename     string
	MIMEType     string
	ByteSize     int64
	Source       string
	BlobSignedID string
}

type FolderInput struct {
	ParentID *int64
	Name     string
	Source   string
}
