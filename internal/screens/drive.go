package screens

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/telex-cli/internal/drivestore"
)

type DriveSyncFunc func(context.Context) (DriveSyncResult, error)

type DriveSyncResult struct {
	Folders          int
	Files            int
	DownloadedFiles  int
	DownloadFailures int
}

type Drive struct {
	store       drivestore.Store
	sync        DriveSyncFunc
	path        string
	entries     []drivestore.Entry
	index       int
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
}

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

func NewDrive(store drivestore.Store, sync DriveSyncFunc) Drive {
	return Drive{store: store, sync: sync, path: store.DriveRoot(), loading: true, keys: DefaultDriveKeyMap()}
}

func DefaultDriveKeyMap() DriveKeyMap {
	return DriveKeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "item up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "item down")),
		Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open folder")),
		Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "parent")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload cache")),
		Sync:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sync drive")),
	}
}

func (d Drive) Init() tea.Cmd { return d.loadCmd(d.path) }

func (d Drive) Update(msg tea.Msg) (Screen, tea.Cmd) {
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
	case tea.KeyPressMsg:
		return d.handleKey(msg)
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
	var b strings.Builder
	b.WriteString("Drive / " + strings.Join(d.breadcrumbs, " / ") + "\n")
	if d.status != "" {
		b.WriteString(d.status + "\n")
	}
	if d.syncing {
		b.WriteString("Syncing remote Drive...\n")
	}
	b.WriteString("\n")
	if len(d.entries) == 0 {
		b.WriteString("No mirrored Drive items found. Press S to sync.\n")
		return style.Render(b.String())
	}
	for i, entry := range d.entries {
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
	return style.Render(b.String())
}

func (d Drive) Title() string { return "Drive" }

func (d Drive) KeyBindings() []key.Binding {
	return []key.Binding{d.keys.Up, d.keys.Down, d.keys.Open, d.keys.Back, d.keys.Refresh, d.keys.Sync}
}

func (d Drive) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, d.keys.Up):
		if d.index > 0 {
			d.index--
		}
	case key.Matches(msg, d.keys.Down):
		if d.index < len(d.entries)-1 {
			d.index++
		}
	case key.Matches(msg, d.keys.Open):
		if len(d.entries) == 0 || d.entries[d.index].Kind != "folder" {
			return d, nil
		}
		path := d.entries[d.index].Path
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
