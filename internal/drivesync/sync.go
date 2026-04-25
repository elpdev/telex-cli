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
	folders, err := listAllFolders(ctx, service)
	if err != nil {
		return Result{}, err
	}
	files, err := listAllFiles(ctx, service)
	if err != nil {
		return Result{}, err
	}
	return syncFolderTree(ctx, store, service, store.DriveRoot(), 0, groupFolders(folders), groupFiles(files), syncMode, time.Now())
}

func syncFolderTree(ctx context.Context, store drivestore.Store, service *drive.Service, localPath string, parentID int64, foldersByParent map[int64][]drive.Folder, filesByFolder map[int64][]drive.File, syncMode string, syncedAt time.Time) (Result, error) {
	result := Result{}
	for _, file := range filesByFolder[parentID] {
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
	for _, folder := range foldersByParent[parentID] {
		folderPath, err := store.StoreFolder(localPath, folder, syncedAt)
		if err != nil {
			return result, fmt.Errorf("store folder %d: %w", folder.ID, err)
		}
		result.Folders++
		child, err := syncFolderTree(ctx, store, service, folderPath, folder.ID, foldersByParent, filesByFolder, syncMode, syncedAt)
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

func listAllFolders(ctx context.Context, service *drive.Service) ([]drive.Folder, error) {
	page := 1
	all := []drive.Folder{}
	for {
		folders, pagination, err := service.ListFolders(ctx, drive.ListFoldersParams{ListParams: drive.ListParams{Page: page, PerPage: 100}, Sort: "name"})
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

func listAllFiles(ctx context.Context, service *drive.Service) ([]drive.File, error) {
	page := 1
	all := []drive.File{}
	for {
		files, pagination, err := service.ListFiles(ctx, drive.ListFilesParams{ListParams: drive.ListParams{Page: page, PerPage: 100}, Sort: "filename"})
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

func groupFolders(folders []drive.Folder) map[int64][]drive.Folder {
	grouped := make(map[int64][]drive.Folder)
	for _, folder := range folders {
		parentID := int64(0)
		if folder.ParentID != nil {
			parentID = *folder.ParentID
		}
		grouped[parentID] = append(grouped[parentID], folder)
	}
	return grouped
}

func groupFiles(files []drive.File) map[int64][]drive.File {
	grouped := make(map[int64][]drive.File)
	for _, file := range files {
		folderID := int64(0)
		if file.FolderID != nil {
			folderID = *file.FolderID
		}
		grouped[folderID] = append(grouped[folderID], file)
	}
	return grouped
}
