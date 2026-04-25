package drive

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/elpdev/telex-cli/internal/api"
)

type Client interface {
	Get(context.Context, string, url.Values) ([]byte, int, error)
	Post(context.Context, string, any) ([]byte, int, error)
	Patch(context.Context, string, any) ([]byte, int, error)
	Delete(context.Context, string) (int, error)
	Download(context.Context, string) ([]byte, string, error)
	PutRaw(context.Context, string, map[string]string, io.Reader) (int, error)
}

type Service struct {
	client Client
}

func NewService(client Client) *Service { return &Service{client: client} }

func (s *Service) ListFolders(ctx context.Context, params ListFoldersParams) ([]Folder, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/folders", folderQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]Folder](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
}

func (s *Service) ListFiles(ctx context.Context, params ListFilesParams) ([]File, *api.Pagination, error) {
	body, _, err := s.client.Get(ctx, "/api/v1/files", fileQuery(params))
	if err != nil {
		return nil, nil, err
	}
	envelope, err := api.DecodeEnvelope[[]File](body)
	if err != nil {
		return nil, nil, err
	}
	return envelope.Data, api.DecodePagination(envelope.Meta), nil
}

func (s *Service) ShowFile(ctx context.Context, id int64) (*File, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/files/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[File](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) ShowFolder(ctx context.Context, id int64) (*Folder, error) {
	body, _, err := s.client.Get(ctx, fmt.Sprintf("/api/v1/folders/%d", id), nil)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Folder](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) CreateFolder(ctx context.Context, input FolderInput) (*Folder, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/folders", map[string]any{"folder": folderInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Folder](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateFolder(ctx context.Context, id int64, input FolderInput) (*Folder, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/folders/%d", id), map[string]any{"folder": folderInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[Folder](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) UpdateFile(ctx context.Context, id int64, input FileInput) (*File, error) {
	body, _, err := s.client.Patch(ctx, fmt.Sprintf("/api/v1/files/%d", id), map[string]any{"stored_file": fileInputMap(input)})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[File](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) RenameFile(ctx context.Context, id int64, name string) (*File, error) {
	return s.UpdateFile(ctx, id, FileInput{Filename: name})
}

func (s *Service) RenameFolder(ctx context.Context, id int64, name string) (*Folder, error) {
	return s.UpdateFolder(ctx, id, FolderInput{Name: name})
}

func (s *Service) MoveFile(ctx context.Context, id int64, folderID *int64) (*File, error) {
	return s.UpdateFile(ctx, id, FileInput{FolderID: folderID})
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/files/%d", id))
	return err
}

func (s *Service) DeleteFolder(ctx context.Context, id int64) error {
	_, err := s.client.Delete(ctx, fmt.Sprintf("/api/v1/folders/%d", id))
	return err
}

func (s *Service) CreateFile(ctx context.Context, input FileInput) (*File, error) {
	payload := map[string]any{"stored_file": fileInputMap(input)}
	if input.BlobSignedID != "" {
		payload["blob_signed_id"] = input.BlobSignedID
	}
	body, _, err := s.client.Post(ctx, "/api/v1/files", payload)
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[File](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func (s *Service) DownloadFile(ctx context.Context, file File) ([]byte, error) {
	if file.DownloadURL == "" {
		return nil, fmt.Errorf("file %d has no download URL", file.ID)
	}
	body, _, err := s.client.Download(ctx, file.DownloadURL)
	return body, err
}

func (s *Service) UploadFile(ctx context.Context, localPath string, folderID *int64) (*File, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", localPath)
	}
	checksum, err := fileChecksum(localPath)
	if err != nil {
		return nil, err
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(localPath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	direct, err := s.createDirectUpload(ctx, filepath.Base(localPath), info.Size(), checksum, contentType)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer closeSilently(file)
	if _, err := s.client.PutRaw(ctx, direct.DirectUpload.URL, direct.DirectUpload.Headers, file); err != nil {
		return nil, err
	}
	return s.CreateFile(ctx, FileInput{FolderID: folderID, Source: "local", BlobSignedID: direct.SignedID})
}

func (s *Service) createDirectUpload(ctx context.Context, filename string, byteSize int64, checksum, contentType string) (*DirectUpload, error) {
	body, _, err := s.client.Post(ctx, "/api/v1/direct_uploads", map[string]any{"blob": map[string]any{"filename": filename, "byte_size": byteSize, "checksum": checksum, "content_type": contentType}})
	if err != nil {
		return nil, err
	}
	envelope, err := api.DecodeEnvelope[DirectUpload](body)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

func folderQuery(params ListFoldersParams) url.Values {
	query := url.Values{}
	setPage(query, params.ListParams)
	if params.Root {
		query.Set("parent_id", "root")
	} else if params.ParentID != nil {
		query.Set("parent_id", fmt.Sprintf("%d", *params.ParentID))
	}
	setString(query, "q", params.Query)
	setString(query, "sort", params.Sort)
	return query
}

func fileQuery(params ListFilesParams) url.Values {
	query := url.Values{}
	setPage(query, params.ListParams)
	if params.Root {
		query.Set("folder_id", "root")
	} else if params.FolderID != nil {
		query.Set("folder_id", fmt.Sprintf("%d", *params.FolderID))
	}
	setString(query, "q", params.Query)
	setString(query, "sort", params.Sort)
	return query
}

func folderInputMap(input FolderInput) map[string]any {
	payload := map[string]any{"name": input.Name}
	if input.ParentID != nil {
		payload["parent_id"] = *input.ParentID
	}
	if input.Source != "" {
		payload["source"] = input.Source
	}
	return payload
}

func fileInputMap(input FileInput) map[string]any {
	payload := map[string]any{}
	if input.FolderID != nil {
		payload["folder_id"] = *input.FolderID
	}
	if input.Filename != "" {
		payload["filename"] = input.Filename
	}
	if input.MIMEType != "" {
		payload["mime_type"] = input.MIMEType
	}
	if input.ByteSize > 0 {
		payload["byte_size"] = input.ByteSize
	}
	if input.Source != "" {
		payload["source"] = input.Source
	}
	return payload
}

func setPage(query url.Values, params ListParams) {
	if params.Page > 0 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.PerPage > 0 {
		query.Set("per_page", fmt.Sprintf("%d", params.PerPage))
	}
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func fileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer closeSilently(file)
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

func closeSilently(closer io.Closer) { _ = closer.Close() }
