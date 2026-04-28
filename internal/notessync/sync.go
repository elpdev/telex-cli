package notessync

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notestore"
)

type Result struct {
	Folders int
	Notes   int
}

func Run(ctx context.Context, store notestore.Store, service *notes.Service) (Result, error) {
	tree, err := service.NotesTree(ctx)
	if err != nil {
		return Result{}, err
	}
	syncedAt := time.Now()
	if err := store.StoreTree(tree, syncedAt); err != nil {
		return Result{}, err
	}
	var result Result
	updatedSince := latestNoteUpdatedSince(store)
	if err := syncFolder(ctx, store, service, *tree, syncedAt, updatedSince, &result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func syncFolder(ctx context.Context, store notestore.Store, service *notes.Service, folder notes.FolderTree, syncedAt time.Time, updatedSince string, result *Result) error {
	result.Folders++
	page := 1
	for {
		cached, pagination, err := service.ListNotes(ctx, notes.ListNotesParams{ListParams: notes.ListParams{Page: page, PerPage: 100}, FolderID: &folder.ID, UpdatedSince: updatedSince, Sort: "filename"})
		if err != nil {
			return err
		}
		for _, note := range cached {
			if err := store.StoreNote(note, syncedAt); err != nil {
				return err
			}
			result.Notes++
		}
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount {
			break
		}
		page++
	}
	for _, child := range folder.Children {
		if err := syncFolder(ctx, store, service, child, syncedAt, updatedSince, result); err != nil {
			return err
		}
	}
	return nil
}

func latestNoteUpdatedSince(store notestore.Store) string {
	notes, err := store.AllNotes()
	if err != nil {
		return ""
	}
	var latest time.Time
	for _, note := range notes {
		if note.Meta.RemoteUpdatedAt.After(latest) {
			latest = note.Meta.RemoteUpdatedAt
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339Nano)
}
