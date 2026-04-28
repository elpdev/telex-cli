package screens

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/notestore"
)

func NewNotes(store notestore.Store, sync NotesSyncFunc) Notes {
	return Notes{store: store, sync: sync, loading: true, keys: DefaultNotesKeyMap(), rowList: newNotesList(nil, 0, 0, 0)}
}

func (n Notes) WithActions(create CreateNoteFunc, update UpdateNoteFunc, delete DeleteNoteFunc) Notes {
	n.create = create
	n.update = update
	n.delete = delete
	return n
}

func (n Notes) Init() tea.Cmd { return n.loadCmd(0) }

func (n Notes) Title() string { return "Notes" }
