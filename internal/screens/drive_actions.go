package screens

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

func (d Drive) handleAction(action string) (Screen, tea.Cmd) {
	if d.pickerOpen || d.confirm != "" || d.prompt != drivePromptNone || d.filtering {
		return d, nil
	}
	switch action {
	case "sync":
		if d.sync == nil || d.syncing {
			return d, nil
		}
		d.syncing = true
		d.status = ""
		return d, d.syncCmd()
	case "upload":
		cwd, _ := filepath.Abs(".")
		d.picker = filepicker.New("", cwd, filepicker.ModeOpenFile)
		d.pickerOpen = true
		d.status = "Select file to upload"
		return d, d.picker.Init()
	case "new-folder":
		d.prompt = drivePromptNewFolder
		d.promptInput = ""
		d.status = ""
	case "rename":
		entry, ok := d.selectedEntry()
		if ok {
			d.prompt = drivePromptRename
			d.promptInput = entry.Name
			d.status = ""
		}
	case "delete":
		entry, ok := d.selectedEntry()
		if ok {
			d.confirm = "Delete " + entry.Name + "?"
		}
	case "details":
		d.details = !d.details
	}
	return d, nil
}

func (d Drive) openSelectedFile(entry drivestore.Entry) (Screen, tea.Cmd) {
	if entry.File == nil || d.open == nil {
		d.status = "Open is not configured"
		return d, nil
	}
	if !entry.Cached && d.download == nil {
		d.status = "Download is not configured"
		return d, nil
	}
	d.status = "Opening file..."
	return d, func() tea.Msg {
		meta := *entry.File
		if !entry.Cached {
			body, err := d.download(context.Background(), meta)
			if err != nil {
				return driveActionFinishedMsg{path: d.path, err: err}
			}
			if err := d.store.WriteFileContent(entry.Path, meta, body, time.Now()); err != nil {
				return driveActionFinishedMsg{path: d.path, err: err}
			}
		}
		if err := d.open(entry.Path); err != nil {
			return driveActionFinishedMsg{path: d.path, err: err}
		}
		return driveActionFinishedMsg{path: d.path, status: "Opened " + entry.Name}
	}
}

func (d Drive) uploadCmd(path string) tea.Cmd {
	if d.upload == nil {
		return func() tea.Msg {
			return driveActionFinishedMsg{path: d.path, err: fmt.Errorf("upload is not configured")}
		}
	}
	currentPath := d.path
	return func() tea.Msg {
		folderID, err := d.store.CurrentFolderRemoteID(currentPath)
		if err != nil {
			return driveActionFinishedMsg{path: currentPath, err: err}
		}
		if err := d.upload(context.Background(), path, folderID); err != nil {
			return driveActionFinishedMsg{path: currentPath, err: err}
		}
		return driveActionFinishedMsg{path: currentPath, status: "Uploaded " + filepath.Base(path)}
	}
}

func (d Drive) createFolderCmd(name string) tea.Cmd {
	if d.create == nil {
		return func() tea.Msg {
			return driveActionFinishedMsg{path: d.path, err: fmt.Errorf("create folder is not configured")}
		}
	}
	currentPath := d.path
	return func() tea.Msg {
		parentID, err := d.store.CurrentFolderRemoteID(currentPath)
		if err != nil {
			return driveActionFinishedMsg{path: currentPath, err: err}
		}
		if err := d.create(context.Background(), drive.FolderInput{ParentID: parentID, Name: name}); err != nil {
			return driveActionFinishedMsg{path: currentPath, err: err}
		}
		return driveActionFinishedMsg{path: currentPath, status: "Created folder " + name}
	}
}

func (d Drive) renameCmd(name string) tea.Cmd {
	entry, ok := d.selectedEntry()
	if !ok {
		return nil
	}
	currentPath := d.path
	return func() tea.Msg {
		if entry.Kind == "folder" && entry.Folder != nil {
			if d.renameDir == nil {
				return driveActionFinishedMsg{path: currentPath, err: fmt.Errorf("rename folder is not configured")}
			}
			if err := d.renameDir(context.Background(), entry.Folder.RemoteID, drive.FolderInput{Name: name}); err != nil {
				return driveActionFinishedMsg{path: currentPath, err: err}
			}
		} else if entry.File != nil {
			if d.renameFile == nil {
				return driveActionFinishedMsg{path: currentPath, err: fmt.Errorf("rename file is not configured")}
			}
			if err := d.renameFile(context.Background(), entry.File.RemoteID, drive.FileInput{Filename: name}); err != nil {
				return driveActionFinishedMsg{path: currentPath, err: err}
			}
		}
		return driveActionFinishedMsg{path: currentPath, status: "Renamed to " + name}
	}
}

func (d Drive) deleteCmd() tea.Cmd {
	entry, ok := d.selectedEntry()
	if !ok {
		return nil
	}
	currentPath := d.path
	return func() tea.Msg {
		if entry.Kind == "folder" && entry.Folder != nil {
			if d.deleteDir == nil {
				return driveActionFinishedMsg{path: currentPath, err: fmt.Errorf("delete folder is not configured")}
			}
			if err := d.deleteDir(context.Background(), entry.Folder.RemoteID); err != nil {
				return driveActionFinishedMsg{path: currentPath, err: err}
			}
		} else if entry.File != nil {
			if d.deleteFile == nil {
				return driveActionFinishedMsg{path: currentPath, err: fmt.Errorf("delete file is not configured")}
			}
			if err := d.deleteFile(context.Background(), entry.File.RemoteID); err != nil {
				return driveActionFinishedMsg{path: currentPath, err: err}
			}
		}
		return driveActionFinishedMsg{path: currentPath, status: "Deleted " + entry.Name}
	}
}
