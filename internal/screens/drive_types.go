package screens

import (
	"context"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"github.com/elpdev/telex-cli/internal/components/filepicker"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

type DriveSyncFunc func(context.Context) (DriveSyncResult, error)
type DriveDownloadFunc func(context.Context, drivestore.FileMeta) ([]byte, error)
type DriveOpenFunc func(string) error
type DriveUploadFunc func(context.Context, string, *int64) error
type DriveCreateFolderFunc func(context.Context, drive.FolderInput) error
type DriveRenameFileFunc func(context.Context, int64, drive.FileInput) error
type DriveRenameFolderFunc func(context.Context, int64, drive.FolderInput) error
type DriveDeleteFunc func(context.Context, int64) error

type DriveSyncResult struct {
	Folders          int
	Files            int
	DownloadedFiles  int
	DownloadFailures int
}

type Drive struct {
	store       drivestore.Store
	sync        DriveSyncFunc
	download    DriveDownloadFunc
	open        DriveOpenFunc
	upload      DriveUploadFunc
	create      DriveCreateFolderFunc
	renameFile  DriveRenameFileFunc
	renameDir   DriveRenameFolderFunc
	deleteFile  DriveDeleteFunc
	deleteDir   DriveDeleteFunc
	path        string
	entries     []drivestore.Entry
	entryList   list.Model
	index       int
	filter      string
	filtering   bool
	details     bool
	prompt      drivePrompt
	promptInput string
	confirm     string
	picker      filepicker.Model
	pickerOpen  bool
	loading     bool
	syncing     bool
	err         error
	status      string
	keys        DriveKeyMap
	breadcrumbs []string
}

type DriveKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Search  key.Binding
	Details key.Binding
	Upload  key.Binding
	NewDir  key.Binding
	Rename  key.Binding
	Delete  key.Binding
}

type drivePrompt int

const (
	drivePromptNone drivePrompt = iota
	drivePromptNewFolder
	drivePromptRename
)

type driveLoadedMsg struct {
	path    string
	entries []drivestore.Entry
	err     error
}

type driveSyncedMsg struct {
	result DriveSyncResult
	loaded driveLoadedMsg
	err    error
}

type driveActionFinishedMsg struct {
	path   string
	status string
	err    error
}

type DriveActionMsg struct{ Action string }

type driveListItem struct {
	entry drivestore.Entry
}

func (i driveListItem) FilterValue() string { return i.entry.Name }
