package screens

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

func NewDrive(store drivestore.Store, sync DriveSyncFunc) Drive {
	return Drive{store: store, sync: sync, path: store.DriveRoot(), loading: true, keys: DefaultDriveKeyMap(), entryList: newDriveList(nil, 0, 0, 0)}
}

func (d Drive) WithActions(download DriveDownloadFunc, open DriveOpenFunc, upload DriveUploadFunc, create DriveCreateFolderFunc, renameFile DriveRenameFileFunc, renameDir DriveRenameFolderFunc, deleteFile DriveDeleteFunc, deleteDir DriveDeleteFunc) Drive {
	d.download = download
	d.open = open
	d.upload = upload
	d.create = create
	d.renameFile = renameFile
	d.renameDir = renameDir
	d.deleteFile = deleteFile
	d.deleteDir = deleteDir
	return d
}

func (d Drive) Init() tea.Cmd { return d.loadCmd(d.path) }

func (d Drive) Title() string { return "Drive" }
