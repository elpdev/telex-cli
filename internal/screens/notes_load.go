package screens

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/notestore"
)

func (n Notes) loadCmd(folderID int64) tea.Cmd {
	return func() tea.Msg { return n.load(folderID) }
}

func (n Notes) load(folderID int64) notesLoadedMsg {
	tree, err := n.store.FolderTree()
	if err != nil {
		return notesLoadedMsg{err: err}
	}
	folder := findNotesFolder(tree, folderID)
	if folder == nil {
		folder = tree
	}
	cached, err := n.store.ListNotes(folder.ID)
	return notesLoadedMsg{tree: tree, folder: folder, notes: cached, err: err}
}

func (n Notes) syncCmd() tea.Cmd {
	folderID := n.currentFolderID()
	return func() tea.Msg {
		result, err := n.sync(context.Background())
		loaded := n.load(folderID)
		if err == nil {
			err = loaded.err
		}
		return notesSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (n Notes) createCmd() tea.Cmd {
	if n.create == nil {
		n.status = "Create is not configured"
		return nil
	}
	folderID := n.currentFolderID()
	return func() tea.Msg {
		input, err := editNoteTemplate(defaultTitle, "")
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		if folderID > 0 {
			input.FolderID = &folderID
		}
		created, err := n.create(context.Background(), input)
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		loaded := n.load(folderID)
		return noteActionFinishedMsg{status: "Created " + created.Title, loaded: loaded, err: loaded.err}
	}
}

func (n Notes) editCmd() tea.Cmd {
	row, ok := n.selectedRow()
	if !ok || row.Note == nil {
		n.status = "Select a note to edit"
		return nil
	}
	return n.editCachedCmd(*row.Note)
}

func (n Notes) editCachedCmd(cached notestore.CachedNote) tea.Cmd {
	if n.update == nil {
		n.status = "Edit is not configured"
		return nil
	}
	folderID := n.currentFolderID()
	return func() tea.Msg {
		input, err := editNoteTemplate(cached.Meta.Title, cached.Body)
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		if cached.Meta.FolderID > 0 {
			input.FolderID = &cached.Meta.FolderID
		}
		updated, err := n.update(context.Background(), cached.Meta.RemoteID, input)
		if err != nil {
			return noteActionFinishedMsg{err: err}
		}
		loaded := n.load(folderID)
		return noteActionFinishedMsg{status: "Updated " + updated.Title, loaded: loaded, err: loaded.err}
	}
}

func (n Notes) deleteCmd() tea.Cmd {
	row, ok := n.selectedRow()
	if !ok || row.Note == nil {
		return nil
	}
	if n.delete == nil {
		return func() tea.Msg { return noteActionFinishedMsg{err: fmt.Errorf("delete is not configured")} }
	}
	folderID := n.currentFolderID()
	note := *row.Note
	return func() tea.Msg {
		if err := n.delete(context.Background(), note.Meta.RemoteID); err != nil {
			return noteActionFinishedMsg{err: err}
		}
		loaded := n.load(folderID)
		return noteActionFinishedMsg{status: "Deleted " + note.Meta.Title, loaded: loaded, err: loaded.err}
	}
}
