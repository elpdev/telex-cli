package screens

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"context"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
)

type NotesSyncFunc func(context.Context) (NotesSyncResult, error)
type CreateNoteFunc func(context.Context, notes.NoteInput) (*notes.Note, error)
type UpdateNoteFunc func(context.Context, int64, notes.NoteInput) (*notes.Note, error)
type DeleteNoteFunc func(context.Context, int64) error

type NotesSyncResult struct {
	Folders int
	Notes   int
}

type Notes struct {
	store        notestore.Store
	sync         NotesSyncFunc
	create       CreateNoteFunc
	update       UpdateNoteFunc
	delete       DeleteNoteFunc
	tree         *notes.FolderTree
	folder       *notes.FolderTree
	notes        []notestore.CachedNote
	rows         []noteRow
	rowList      list.Model
	index        int
	detail       *notestore.CachedNote
	detailScroll int
	filter       string
	editing      bool
	confirm      string
	loading      bool
	syncing      bool
	sortMode     string
	flat         bool
	err          error
	status       string
	keys         NotesKeyMap
}

type NotesKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Sync    key.Binding
	Search  key.Binding
	New     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Order   key.Binding
	Flat    key.Binding
}

type noteRow struct {
	Kind   string
	Name   string
	Folder *notes.FolderTree
	Note   *notestore.CachedNote
}

type noteListItem struct {
	row noteRow
}

func (i noteListItem) FilterValue() string { return i.row.Name }

type notesLoadedMsg struct {
	tree   *notes.FolderTree
	folder *notes.FolderTree
	notes  []notestore.CachedNote
	err    error
}

type notesSyncedMsg struct {
	result NotesSyncResult
	loaded notesLoadedMsg
	err    error
}

type noteActionFinishedMsg struct {
	status string
	loaded notesLoadedMsg
	err    error
}

type NotesActionMsg struct{ Action string }
