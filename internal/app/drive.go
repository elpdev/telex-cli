package app

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/drivesync"
	"github.com/elpdev/telex-cli/internal/screens"
)

func (m *Model) syncDrive(ctx context.Context) (screens.DriveSyncResult, error) {
	service, cfg, err := m.driveService()
	if err != nil {
		return screens.DriveSyncResult{}, err
	}
	result, err := drivesync.Run(ctx, drivestore.New(m.dataPath), service, cfg.DriveSyncMode())
	return screens.DriveSyncResult{Folders: result.Folders, Files: result.Files, DownloadedFiles: result.DownloadedFiles, DownloadFailures: result.DownloadFailures}, err
}

func (m *Model) downloadDriveFile(ctx context.Context, meta drivestore.FileMeta) ([]byte, error) {
	service, _, err := m.driveService()
	if err != nil {
		return nil, err
	}
	remote, err := service.ShowFile(ctx, meta.RemoteID)
	if err != nil {
		return nil, err
	}
	return service.DownloadFile(ctx, *remote)
}

func (m *Model) openDriveFile(path string) error {
	opener := os.Getenv("OPENER")
	if opener != "" {
		return startDetached(openerCommand(opener, path))
	}
	if textFile(path) {
		if editor := os.Getenv("VISUAL"); editor != "" {
			if cmd := terminalCommand(editor, path); cmd != nil {
				return startDetached(cmd)
			}
		}
		if editor := os.Getenv("EDITOR"); editor != "" {
			if cmd := terminalCommand(editor, path); cmd != nil {
				return startDetached(cmd)
			}
		}
	}
	return startDetached(exec.Command("xdg-open", path))
}

func openerCommand(opener, path string) *exec.Cmd {
	parts := strings.Fields(opener)
	if len(parts) == 0 {
		return exec.Command("xdg-open", path)
	}
	return exec.Command(parts[0], append(parts[1:], path)...)
}

func terminalCommand(editor, path string) *exec.Cmd {
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return nil
	}
	terminal := os.Getenv("TERMINAL")
	if terminal != "" {
		if cmd := terminalCommandFor(terminal, editorParts, path); cmd != nil {
			return cmd
		}
	}
	for _, candidate := range []string{"ghostty", "alacritty", "kitty"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return terminalCommandFor(candidate, editorParts, path)
		}
	}
	return nil
}

func terminalCommandFor(terminal string, editorParts []string, path string) *exec.Cmd {
	terminalParts := strings.Fields(terminal)
	if len(terminalParts) == 0 {
		return nil
	}
	args := append([]string{}, terminalParts[1:]...)
	args = append(args, "-e")
	args = append(args, editorParts...)
	args = append(args, path)
	return exec.Command(terminalParts[0], args...)
}

func startDetached(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func textFile(path string) bool {
	lower := strings.ToLower(path)
	for _, suffix := range []string{".md", ".markdown", ".txt", ".text", ".log", ".csv", ".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".cfg"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func (m *Model) uploadDriveFile(ctx context.Context, path string, folderID *int64) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.UploadFile(ctx, path, folderID); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) createDriveFolder(ctx context.Context, input drive.FolderInput) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.CreateFolder(ctx, input); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) renameDriveFile(ctx context.Context, id int64, input drive.FileInput) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.UpdateFile(ctx, id, input); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) renameDriveFolder(ctx context.Context, id int64, input drive.FolderInput) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if _, err := service.UpdateFolder(ctx, id, input); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) deleteDriveFile(ctx context.Context, id int64) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if err := service.DeleteFile(ctx, id); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}

func (m *Model) deleteDriveFolder(ctx context.Context, id int64) error {
	service, _, err := m.driveService()
	if err != nil {
		return err
	}
	if err := service.DeleteFolder(ctx, id); err != nil {
		return err
	}
	_, err = drivesync.Run(ctx, drivestore.New(m.dataPath), service, config.DriveSyncMetadataOnly)
	return err
}
