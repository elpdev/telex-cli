package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/drivesync"
	"github.com/spf13/cobra"
)

func newDriveCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "drive", Short: "Drive commands"}
	cmd.AddCommand(newDriveSyncCommand(rt))
	cmd.AddCommand(newDriveListCommand(rt))
	cmd.AddCommand(newDriveUploadCommand(rt))
	cmd.AddCommand(newDriveDownloadCommand(rt))
	return cmd
}

func newDriveSyncCommand(rt *runtime) *cobra.Command {
	var metadataOnly bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync remote Drive into the local mirror",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, cfg, err := driveService(rt)
			if err != nil {
				return err
			}
			syncMode := cfg.DriveSyncMode()
			if metadataOnly {
				syncMode = config.DriveSyncMetadataOnly
			}
			result, err := drivesync.Run(rt.context(), drivestore.New(rt.dataPath), service, syncMode)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Synced %d folder(s), %d file(s).\n", result.Folders, result.Files)
			if syncMode == config.DriveSyncFull {
				fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %d file content(s).\n", result.DownloadedFiles)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Metadata-only mode: file contents were not downloaded.")
			}
			if result.DownloadFailures > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "Skipped %d file download(s) due to remote errors; metadata was still cached.\n", result.DownloadFailures)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&metadataOnly, "metadata-only", false, "override config and sync folder/file metadata only")
	return cmd
}

func newDriveListCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "ls [path]",
		Short: "List local mirrored Drive contents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := drivestore.New(rt.dataPath)
			path, err := driveLocalPath(store, args)
			if err != nil {
				return err
			}
			entries, err := store.List(path)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(entries))
			for _, entry := range entries {
				remoteID := ""
				status := "local"
				if entry.Kind == "folder" && entry.Folder != nil {
					remoteID = strconv.FormatInt(entry.Folder.RemoteID, 10)
				} else if entry.File != nil {
					remoteID = strconv.FormatInt(entry.File.RemoteID, 10)
					if !entry.Cached {
						status = "remote-only"
					}
				}
				rows = append(rows, []string{entry.Kind, remoteID, entry.Name, status, strconv.FormatInt(entry.ByteSize, 10), entry.Path})
			}
			writeRows(cmd.OutOrStdout(), []string{"kind", "remote_id", "name", "status", "bytes", "path"}, rows)
			return nil
		},
	}
}

func newDriveUploadCommand(rt *runtime) *cobra.Command {
	var folderID int64
	cmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "Upload a local file to Drive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := driveService(rt)
			if err != nil {
				return err
			}
			var target *int64
			if folderID > 0 {
				target = &folderID
			}
			file, err := service.UploadFile(rt.context(), args[0], target)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"remote_id", strconv.FormatInt(file.ID, 10)}, {"filename", file.Filename}, {"status", "uploaded"}})
			return nil
		},
	}
	cmd.Flags().Int64Var(&folderID, "folder-id", 0, "remote Drive folder ID; omit for root")
	return cmd
}

func newDriveDownloadCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "download <local-mirror-path>",
		Short: "Download a remote-only mirrored file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := drivestore.New(rt.dataPath)
			path, err := driveLocalPath(store, args)
			if err != nil {
				return err
			}
			meta, err := drivestore.ReadFileMeta(path)
			if err != nil {
				return err
			}
			service, _, err := driveService(rt)
			if err != nil {
				return err
			}
			remote, err := service.ShowFile(rt.context(), meta.RemoteID)
			if err != nil {
				return err
			}
			body, err := service.DownloadFile(rt.context(), *remote)
			if err != nil {
				return err
			}
			if _, err := store.StoreFile(filepath.Dir(path), *remote, body, meta.SyncedAt); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %s\n", path)
			return nil
		},
	}
}

func driveService(rt *runtime) (*drive.Service, *config.Config, error) {
	cfg, err := rt.loadConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := rt.apiClient()
	if err != nil {
		return nil, nil, err
	}
	return drive.NewService(client), cfg, nil
}

func driveLocalPath(store drivestore.Store, args []string) (string, error) {
	root := store.DriveRoot()
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" || args[0] == "." || args[0] == "/" {
		return root, nil
	}
	path := args[0]
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("drive path %q is outside %s", args[0], root)
	}
	return cleanPath, nil
}
