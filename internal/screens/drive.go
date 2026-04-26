package screens

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func NewDrive(store drivestore.Store, sync DriveSyncFunc) Drive {
	return Drive{store: store, sync: sync, path: store.DriveRoot(), loading: true, keys: DefaultDriveKeyMap()}
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

func DefaultDriveKeyMap() DriveKeyMap {
	return DriveKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open folder")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "parent")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync drive")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Details: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "details")),
		Upload:  key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "upload")),
		NewDir:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new folder")),
		Rename:  key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
		Delete:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
	}
}

func (d Drive) Init() tea.Cmd { return d.loadCmd(d.path) }

func (d Drive) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if d.pickerOpen {
		return d.handlePickerMsg(msg)
	}

	switch msg := msg.(type) {
	case driveLoadedMsg:
		d.loading = false
		d.err = msg.err
		if msg.err == nil {
			d.path = msg.path
			d.entries = msg.entries
			if d.index >= len(d.entries) {
				d.index = maxDriveIndex(len(d.entries))
			}
			d.breadcrumbs = d.pathParts()
		}
		return d, nil
	case driveSyncedMsg:
		d.syncing = false
		d.err = msg.err
		if msg.err == nil {
			d.status = fmt.Sprintf("Synced %d folder(s), %d file(s), downloaded %d", msg.result.Folders, msg.result.Files, msg.result.DownloadedFiles)
			if msg.result.DownloadFailures > 0 {
				d.status += fmt.Sprintf("; %d download warning(s)", msg.result.DownloadFailures)
			}
			d.path = msg.loaded.path
			d.entries = msg.loaded.entries
			d.breadcrumbs = d.pathParts()
		} else {
			d.status = ""
		}
		return d, nil
	case driveActionFinishedMsg:
		d.loading = false
		d.err = msg.err
		if msg.err != nil {
			d.status = fmt.Sprintf("Drive action failed: %v", msg.err)
			return d, nil
		}
		d.status = msg.status
		loaded := d.load(msg.path)
		d.path = loaded.path
		d.entries = loaded.entries
		d.err = loaded.err
		d.breadcrumbs = d.pathParts()
		if d.index >= len(d.visibleEntries()) {
			d.index = maxDriveIndex(len(d.visibleEntries()))
		}
		return d, nil
	case DriveActionMsg:
		return d.handleAction(msg.Action)
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	}
	return d, nil
}

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

func (d Drive) View(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)
	if d.loading {
		return style.Render("Loading local drive mirror...")
	}
	if d.err != nil {
		return style.Render(fmt.Sprintf("Drive cache error: %v\n\nRun `telex drive sync` to populate the local drive mirror.", d.err))
	}
	if d.pickerOpen {
		return style.Render(d.picker.View(width, height))
	}
	var b strings.Builder
	b.WriteString("Drive / " + strings.Join(d.breadcrumbs, " / ") + "\n")
	if d.status != "" {
		b.WriteString(d.status + "\n")
	}
	if d.filtering {
		b.WriteString("Filter: " + d.filter + "\n")
	}
	if d.prompt != drivePromptNone {
		b.WriteString(d.promptLabel() + d.promptInput + "\n")
	}
	if d.confirm != "" {
		b.WriteString(d.confirm + " [y/N]\n")
	}
	if d.syncing {
		b.WriteString("Syncing remote Drive...\n")
	}
	b.WriteString("\n")
	entries := d.visibleEntries()
	if len(entries) == 0 {
		b.WriteString("No mirrored Drive items found. Press S to sync.\n")
		return style.Render(b.String())
	}
	for i, entry := range entries {
		cursor := "  "
		if i == d.index {
			cursor = "> "
		}
		kind := "file"
		status := ""
		if entry.Kind == "folder" {
			kind = "dir "
		} else if !entry.Cached {
			status = " remote-only"
		}
		b.WriteString(fmt.Sprintf("%s%s  %s%s\n", cursor, kind, entry.Name, status))
	}
	if d.details {
		b.WriteString("\n" + d.detailsView())
	}
	return style.Render(b.String())
}

func (d Drive) Title() string { return "Drive" }

func (d Drive) KeyBindings() []key.Binding {
	return []key.Binding{d.keys.Up, d.keys.Down, d.keys.Open, d.keys.Back, d.keys.Refresh, d.keys.Sync, d.keys.Search, d.keys.Details, d.keys.Upload, d.keys.NewDir, d.keys.Rename, d.keys.Delete}
}

func (d Drive) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if d.pickerOpen {
		return d.handlePickerMsg(msg)
	}
	if d.confirm != "" {
		return d.handleConfirmKey(msg)
	}
	if d.prompt != drivePromptNone {
		return d.handlePromptKey(msg)
	}
	if d.filtering {
		return d.handleFilterKey(msg)
	}
	entries := d.visibleEntries()
	switch {
	case key.Matches(msg, d.keys.Up):
		if d.index > 0 {
			d.index--
		}
	case key.Matches(msg, d.keys.Down):
		if d.index < len(entries)-1 {
			d.index++
		}
	case key.Matches(msg, d.keys.Open):
		if len(entries) == 0 {
			return d, nil
		}
		entry := entries[d.index]
		if entry.Kind != "folder" {
			return d.openSelectedFile(entry)
		}
		path := entry.Path
		d.index = 0
		return d, d.loadCmd(path)
	case key.Matches(msg, d.keys.Back):
		if filepath.Clean(d.path) == filepath.Clean(d.store.DriveRoot()) {
			return d, nil
		}
		d.index = 0
		return d, d.loadCmd(filepath.Dir(d.path))
	case key.Matches(msg, d.keys.Refresh):
		return d, d.loadCmd(d.path)
	case key.Matches(msg, d.keys.Sync):
		if d.sync == nil || d.syncing {
			return d, nil
		}
		d.syncing = true
		d.status = ""
		return d, d.syncCmd()
	case key.Matches(msg, d.keys.Search):
		d.filtering = true
		d.filter = ""
		d.index = 0
	case key.Matches(msg, d.keys.Details):
		d.details = !d.details
	case key.Matches(msg, d.keys.Upload):
		cwd, _ := filepath.Abs(".")
		d.picker = filepicker.New("", cwd, filepicker.ModeOpenFile)
		d.pickerOpen = true
		d.status = "Select file to upload"
		return d, d.picker.Init()
	case key.Matches(msg, d.keys.NewDir):
		d.prompt = drivePromptNewFolder
		d.promptInput = ""
		d.status = ""
	case key.Matches(msg, d.keys.Rename):
		if len(entries) == 0 {
			return d, nil
		}
		d.prompt = drivePromptRename
		d.promptInput = entries[d.index].Name
		d.status = ""
	case key.Matches(msg, d.keys.Delete):
		if len(entries) == 0 {
			return d, nil
		}
		d.confirm = "Delete " + entries[d.index].Name + "?"
	}
	return d, nil
}

func (d Drive) loadCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return d.load(path)
	}
}

func (d Drive) load(path string) driveLoadedMsg {
	entries, err := d.store.List(path)
	return driveLoadedMsg{path: path, entries: entries, err: err}
}

func (d Drive) syncCmd() tea.Cmd {
	path := d.path
	return func() tea.Msg {
		result, err := d.sync(context.Background())
		entries, loadErr := d.store.List(path)
		if err == nil {
			err = loadErr
		}
		return driveSyncedMsg{result: result, loaded: driveLoadedMsg{path: path, entries: entries, err: loadErr}, err: err}
	}
}

func (d Drive) handleFilterKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.filtering = false
		d.filter = ""
		d.index = 0
	case "enter":
		d.filtering = false
	case "backspace":
		if len(d.filter) > 0 {
			d.filter = d.filter[:len(d.filter)-1]
		}
		d.index = 0
	default:
		if msg.Text != "" {
			d.filter += msg.Text
			d.index = 0
		}
	}
	return d, nil
}

func (d Drive) handlePromptKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.prompt = drivePromptNone
		d.promptInput = ""
		d.status = "Cancelled"
		return d, nil
	case "enter":
		value := strings.TrimSpace(d.promptInput)
		prompt := d.prompt
		d.prompt = drivePromptNone
		d.promptInput = ""
		if value == "" {
			d.status = "Name is required"
			return d, nil
		}
		if prompt == drivePromptNewFolder {
			return d, d.createFolderCmd(value)
		}
		return d, d.renameCmd(value)
	case "backspace":
		if len(d.promptInput) > 0 {
			d.promptInput = d.promptInput[:len(d.promptInput)-1]
		}
		return d, nil
	}
	if msg.Text != "" {
		d.promptInput += msg.Text
	}
	return d, nil
}

func (d Drive) handleConfirmKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		d.confirm = ""
		return d, d.deleteCmd()
	case "n", "N", "esc":
		d.confirm = ""
		d.status = "Cancelled"
	}
	return d, nil
}

func (d Drive) handlePickerMsg(msg tea.Msg) (Screen, tea.Cmd) {
	picker, action, cmd := d.picker.Update(msg)
	d.picker = picker
	switch action.Type {
	case filepicker.ActionCancel:
		d.pickerOpen = false
		d.status = "Cancelled"
		return d, nil
	case filepicker.ActionSelect:
		d.pickerOpen = false
		return d, d.uploadCmd(action.Path)
	}
	return d, cmd
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

func (d Drive) selectedEntry() (drivestore.Entry, bool) {
	entries := d.visibleEntries()
	if len(entries) == 0 || d.index < 0 || d.index >= len(entries) {
		return drivestore.Entry{}, false
	}
	return entries[d.index], true
}

func (d Drive) visibleEntries() []drivestore.Entry {
	filter := strings.ToLower(strings.TrimSpace(d.filter))
	if filter == "" {
		return d.entries
	}
	out := make([]drivestore.Entry, 0, len(d.entries))
	for _, entry := range d.entries {
		if strings.Contains(strings.ToLower(entry.Name), filter) {
			out = append(out, entry)
		}
	}
	return out
}

func (d Drive) promptLabel() string {
	if d.prompt == drivePromptNewFolder {
		return "New folder: "
	}
	return "Rename: "
}

func (d Drive) detailsView() string {
	entry, ok := d.selectedEntry()
	if !ok {
		return "Details: no selection\n"
	}
	var b strings.Builder
	b.WriteString("Details\n")
	b.WriteString(fmt.Sprintf("Kind: %s\nName: %s\nLocal path: %s\n", entry.Kind, entry.Name, entry.Path))
	if entry.Folder != nil {
		b.WriteString(fmt.Sprintf("Remote ID: %d\nSynced at: %s\n", entry.Folder.RemoteID, entry.Folder.SyncedAt.Format(time.RFC3339)))
	}
	if entry.File != nil {
		cached := "remote-only"
		if entry.Cached {
			cached = "cached"
		}
		b.WriteString(fmt.Sprintf("Remote ID: %d\nMIME type: %s\nByte size: %d\nCached state: %s\nSynced at: %s\nDownload URL: %t\n", entry.File.RemoteID, entry.File.MIMEType, entry.File.ByteSize, cached, entry.File.SyncedAt.Format(time.RFC3339), entry.File.DownloadURL != ""))
	}
	return b.String()
}

func (d Drive) pathParts() []string {
	rel, err := filepath.Rel(d.store.DriveRoot(), d.path)
	if err != nil || rel == "." {
		return nil
	}
	return strings.Split(rel, string(filepath.Separator))
}

func maxDriveIndex(length int) int {
	if length <= 0 {
		return 0
	}
	return length - 1
}
