package app

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notessync"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/screens"
)

func (m *Model) syncNotes(ctx context.Context) (screens.NotesSyncResult, error) {
	service, err := m.notesService()
	if err != nil {
		return screens.NotesSyncResult{}, err
	}
	result, err := runNotesSync(ctx, notestore.New(m.dataPath), service)
	return screens.NotesSyncResult{Folders: result.Folders, Notes: result.Notes}, err
}

func (m *Model) createNote(ctx context.Context, input notes.NoteInput) (*notes.Note, error) {
	service, err := m.notesService()
	if err != nil {
		return nil, err
	}
	note, err := service.CreateNote(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := notestore.New(m.dataPath).StoreNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) updateNote(ctx context.Context, id int64, input notes.NoteInput) (*notes.Note, error) {
	service, err := m.notesService()
	if err != nil {
		return nil, err
	}
	note, err := service.UpdateNote(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if err := notestore.New(m.dataPath).StoreNote(*note, time.Now()); err != nil {
		return nil, err
	}
	return note, nil
}

func (m *Model) deleteNote(ctx context.Context, id int64) error {
	service, err := m.notesService()
	if err != nil {
		return err
	}
	if err := service.DeleteNote(ctx, id); err != nil {
		return err
	}
	return notestore.New(m.dataPath).DeleteNote(id)
}

type notesSyncResult = notessync.Result

func runNotesSync(ctx context.Context, store notestore.Store, service *notes.Service) (notesSyncResult, error) {
	return notessync.Run(ctx, store, service)
}
