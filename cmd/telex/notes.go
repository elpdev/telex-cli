package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/notessync"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/spf13/cobra"
)

type notesSyncResult = notessync.Result

func newNotesCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "notes", Short: "Notes commands"}
	cmd.AddCommand(newNotesSyncCommand(rt))
	cmd.AddCommand(newNotesTreeCommand(rt))
	cmd.AddCommand(newNotesListCommand(rt))
	cmd.AddCommand(newNotesShowCommand(rt))
	cmd.AddCommand(newNotesCreateCommand(rt))
	cmd.AddCommand(newNotesEditCommand(rt))
	cmd.AddCommand(newNotesDeleteCommand(rt))
	return cmd
}

func newNotesSyncCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync remote Notes into the local cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := notesService(rt)
			if err != nil {
				return err
			}
			result, err := runNotesSync(rt, service, notestore.New(rt.dataPath))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Synced %d folder(s), %d note(s).\n", result.Folders, result.Notes)
			return nil
		},
	}
}

func newNotesTreeCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "tree",
		Short: "List cached Notes folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			tree, err := notestore.New(rt.dataPath).FolderTree()
			if err != nil {
				return err
			}
			rows := [][]string{}
			appendNotesTreeRows(&rows, *tree, 0)
			writeRows(cmd.OutOrStdout(), []string{"id", "parent_id", "name", "notes", "children"}, rows)
			return nil
		},
	}
}

func newNotesListCommand(rt *runtime) *cobra.Command {
	var folderID int64
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cached notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cached, err := notestore.New(rt.dataPath).ListNotes(folderID)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(cached))
			for _, note := range cached {
				rows = append(rows, cachedNoteRow(note))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "folder_id", "title", "filename", "updated_at", "path"}, rows)
			return nil
		},
	}
	cmd.Flags().Int64Var(&folderID, "folder-id", 0, "cached Notes folder ID; omit for Notes root")
	return cmd
}

func newNotesShowCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a cached note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			cached, err := notestore.New(rt.dataPath).ReadNote(id)
			if err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, cachedNoteFields(*cached))
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", cached.Body)
			return nil
		},
	}
}

func newNotesCreateCommand(rt *runtime) *cobra.Command {
	var folderID int64
	var title string
	var body string
	var filePath string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a remote note and cache it locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("--title is required")
			}
			body, err := noteBodyFromFlags(body, filePath)
			if err != nil {
				return err
			}
			service, err := notesService(rt)
			if err != nil {
				return err
			}
			input := notes.NoteInput{Title: title, Body: body}
			if folderID > 0 {
				input.FolderID = &folderID
			}
			note, err := service.CreateNote(rt.context(), input)
			if err != nil {
				return err
			}
			if err := notestore.New(rt.dataPath).StoreNote(*note, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, noteFields(*note))
			return nil
		},
	}
	cmd.Flags().Int64Var(&folderID, "folder-id", 0, "remote Notes folder ID; omit for Notes root")
	cmd.Flags().StringVar(&title, "title", "", "note title")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	cmd.Flags().StringVar(&filePath, "file", "", "read Markdown body from file")
	return cmd
}

func newNotesEditCommand(rt *runtime) *cobra.Command {
	var title string
	var body string
	var filePath string
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Update a remote note and refresh the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("body") && !cmd.Flags().Changed("file") {
				return fmt.Errorf("provide --title, --body, or --file")
			}
			service, err := notesService(rt)
			if err != nil {
				return err
			}
			current, err := noteForEdit(rt, service, id)
			if err != nil {
				return err
			}
			input := notes.NoteInput{FolderID: current.FolderID, Title: current.Title, Body: current.Body}
			if cmd.Flags().Changed("title") {
				input.Title = title
			}
			if cmd.Flags().Changed("body") || cmd.Flags().Changed("file") {
				input.Body, err = noteBodyFromFlags(body, filePath)
				if err != nil {
					return err
				}
			}
			updated, err := service.UpdateNote(rt.context(), id, input)
			if err != nil {
				return err
			}
			if err := notestore.New(rt.dataPath).StoreNote(*updated, time.Now()); err != nil {
				return err
			}
			writeRows(cmd.OutOrStdout(), []string{"key", "value"}, noteFields(*updated))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "note title")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	cmd.Flags().StringVar(&filePath, "file", "", "read Markdown body from file")
	return cmd
}

func newNotesDeleteCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a remote note and remove the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := notesService(rt)
			if err != nil {
				return err
			}
			if err := service.DeleteNote(rt.context(), id); err != nil {
				return err
			}
			if err := notestore.New(rt.dataPath).DeleteNote(id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted note %d.\n", id)
			return nil
		},
	}
}

func runNotesSync(rt *runtime, service *notes.Service, store notestore.Store) (*notesSyncResult, error) {
	result, err := notessync.Run(rt.context(), store, service)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func notesService(rt *runtime) (*notes.Service, error) {
	client, err := rt.apiClient()
	if err != nil {
		return nil, err
	}
	return notes.NewService(client), nil
}

func noteForEdit(rt *runtime, service *notes.Service, id int64) (*notes.Note, error) {
	store := notestore.New(rt.dataPath)
	if cached, err := store.ReadNote(id); err == nil {
		folderID := cached.Meta.FolderID
		return &notes.Note{ID: cached.Meta.RemoteID, UserID: cached.Meta.UserID, FolderID: &folderID, Title: cached.Meta.Title, Filename: cached.Meta.Filename, MIMEType: cached.Meta.MIMEType, Body: cached.Body, CreatedAt: cached.Meta.RemoteCreatedAt, UpdatedAt: cached.Meta.RemoteUpdatedAt}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return service.ShowNote(rt.context(), id)
}

func noteBodyFromFlags(body, filePath string) (string, error) {
	if body != "" && filePath != "" {
		return "", fmt.Errorf("use --body or --file, not both")
	}
	if filePath == "" {
		return body, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func appendNotesTreeRows(rows *[][]string, folder notes.FolderTree, depth int) {
	parentID := ""
	if folder.ParentID != nil {
		parentID = strconv.FormatInt(*folder.ParentID, 10)
	}
	*rows = append(*rows, []string{strconv.FormatInt(folder.ID, 10), parentID, strings.Repeat("  ", depth) + folder.Name, strconv.Itoa(folder.NoteCount), strconv.Itoa(folder.ChildFolderCount)})
	for _, child := range folder.Children {
		appendNotesTreeRows(rows, child, depth+1)
	}
}

func cachedNoteRow(note notestore.CachedNote) []string {
	return []string{strconv.FormatInt(note.Meta.RemoteID, 10), formatOptionalID(note.Meta.FolderID), note.Meta.Title, note.Meta.Filename, note.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04"), note.Path}
}

func cachedNoteFields(note notestore.CachedNote) [][]string {
	return [][]string{{"id", strconv.FormatInt(note.Meta.RemoteID, 10)}, {"folder_id", formatOptionalID(note.Meta.FolderID)}, {"title", note.Meta.Title}, {"filename", note.Meta.Filename}, {"mime_type", note.Meta.MIMEType}, {"updated_at", note.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04")}, {"path", note.Path}}
}

func noteFields(note notes.Note) [][]string {
	folderID := int64(0)
	if note.FolderID != nil {
		folderID = *note.FolderID
	}
	return [][]string{{"id", strconv.FormatInt(note.ID, 10)}, {"folder_id", formatOptionalID(folderID)}, {"title", note.Title}, {"filename", note.Filename}, {"updated_at", note.UpdatedAt.Format("2006-01-02 15:04")}}
}

func formatOptionalID(id int64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}
