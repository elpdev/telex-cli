package drivesync

import (
	"context"
	"fmt"
	"time"

	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

type Result struct {
	Folders          int
	Files            int
	DownloadedFiles  int
	DownloadFailures int
}

func Run(ctx context.Context, store drivestore.Store, service *drive.Service, syncMode string) (Result, error) {
	if syncMode == "" {
		syncMode = config.DriveSyncFull
	}
	if err := store.EnsureRoot(); err != nil {
		return Result{}, err
	}
	return syncFolder(ctx, store, service, store.DriveRoot(), nil, syncMode, time.Now())
}

func syncFolder(ctx context.Context, store drivestore.Store, service *drive.Service, localPath string, parentID *int64, syncMode string, syncedAt time.Time) (Result, error) {
	result := Result{}
	files, err := listAllFiles(ctx, service, parentID)
	if err != nil {
		return result, err
	}
	for _, file := range files {
		var content []byte
		if syncMode == config.DriveSyncFull && file.Downloadable {
			body, err := service.DownloadFile(ctx, file)
			if err != nil {
				result.DownloadFailures++
			} else {
				content = body
				result.DownloadedFiles++
			}
		}
		if _, err := store.StoreFile(localPath, file, content, syncedAt); err != nil {
			return result, fmt.Errorf("store file %d: %w", file.ID, err)
		}
		result.Files++
	}
	folders, err := listAllFolders(ctx, service, parentID)
	if err != nil {
		return result, err
	}
	for _, folder := range folders {
		folderPath, err := store.StoreFolder(localPath, folder, syncedAt)
		if err != nil {
			return result, fmt.Errorf("store folder %d: %w", folder.ID, err)
		}
		result.Folders++
		childID := folder.ID
		child, err := syncFolder(ctx, store, service, folderPath, &childID, syncMode, syncedAt)
		result.Folders += child.Folders
		result.Files += child.Files
		result.DownloadedFiles += child.DownloadedFiles
		result.DownloadFailures += child.DownloadFailures
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func listAllFolders(ctx context.Context, service *drive.Service, parentID *int64) ([]drive.Folder, error) {
	page := 1
	all := []drive.Folder{}
	for {
		folders, pagination, err := service.ListFolders(ctx, drive.ListFoldersParams{ListParams: drive.ListParams{Page: page, PerPage: 100}, ParentID: parentID, Root: parentID == nil, Sort: "name"})
		if err != nil {
			return all, fmt.Errorf("list folders page %d: %w", page, err)
		}
		all = append(all, folders...)
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount || len(folders) == 0 {
			return all, nil
		}
		page++
	}
}

func listAllFiles(ctx context.Context, service *drive.Service, folderID *int64) ([]drive.File, error) {
	page := 1
	all := []drive.File{}
	for {
		files, pagination, err := service.ListFiles(ctx, drive.ListFilesParams{ListParams: drive.ListParams{Page: page, PerPage: 100}, FolderID: folderID, Root: folderID == nil, Sort: "filename"})
		if err != nil {
			return all, fmt.Errorf("list files page %d: %w", page, err)
		}
		all = append(all, files...)
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount || len(files) == 0 {
			return all, nil
		}
		page++
	}
}
