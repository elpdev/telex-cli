package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskssync"
	"github.com/elpdev/telex-cli/internal/taskstore"
	"github.com/spf13/cobra"
)

type tasksSyncResult = taskssync.Result

func newTasksCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "tasks", Short: "Tasks commands"}
	cmd.AddCommand(newTasksSyncCommand(rt))
	cmd.AddCommand(newTasksProjectsCommand(rt))
	cmd.AddCommand(newTasksProjectCommand(rt))
	cmd.AddCommand(newTasksBoardCommand(rt))
	cmd.AddCommand(newTasksCardsCommand(rt))
	cmd.AddCommand(newTasksCardCommand(rt))
	cmd.AddCommand(newTasksUseCommand(rt))
	return cmd
}

func newTasksUseCommand(rt *runtime) *cobra.Command {
	var clear bool
	cmd := &cobra.Command{
		Use:   "use [project-id]",
		Short: "Set, show, or clear the current task project for CLI commands",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prefs, err := rt.loadPrefs()
			if err != nil {
				return err
			}
			if clear {
				prefs.TasksProjectID = 0
				if err := rt.savePrefs(prefs); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Cleared current task project.")
				return nil
			}
			if len(args) == 0 {
				if prefs.TasksProjectID == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No current task project set.")
					return nil
				}
				name := lookupProjectName(rt, prefs.TasksProjectID)
				if name != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", prefs.TasksProjectID, name)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%d\n", prefs.TasksProjectID)
				}
				return nil
			}
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			prefs.TasksProjectID = id
			if err := rt.savePrefs(prefs); err != nil {
				return err
			}
			name := lookupProjectName(rt, id)
			if name != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Using project %d (%s).\n", id, name)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Using project %d.\n", id)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&clear, "clear", false, "clear the current task project")
	return cmd
}

func lookupProjectName(rt *runtime, id int64) string {
	project, err := taskstore.New(rt.dataPath).ReadProject(id)
	if err != nil {
		return ""
	}
	return project.Meta.Name
}

func resolveProjectID(rt *runtime, args []string) (int64, error) {
	if len(args) >= 1 {
		return parseID(args[0])
	}
	prefs, err := rt.loadPrefs()
	if err != nil {
		return 0, err
	}
	if prefs.TasksProjectID == 0 {
		return 0, fmt.Errorf("no project id given and no current project set; pass <project-id> or run `telex tasks use <id>`")
	}
	return prefs.TasksProjectID, nil
}

func resolveProjectAndCardID(rt *runtime, args []string) (int64, int64, error) {
	switch len(args) {
	case 2:
		return parseTwoIDs(args)
	case 1:
		cardID, err := parseID(args[0])
		if err != nil {
			return 0, 0, err
		}
		projectID, err := resolveProjectID(rt, nil)
		if err != nil {
			return 0, 0, err
		}
		return projectID, cardID, nil
	default:
		return 0, 0, fmt.Errorf("expected <card-id> or <project-id> <card-id>")
	}
}

func newTasksSyncCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{Use: "sync", Short: "Sync remote Tasks into the local cache", RunE: func(cmd *cobra.Command, args []string) error {
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		result, err := runTasksSync(rt.context(), taskstore.New(rt.dataPath), service)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Synced %d project(s), %d board(s), %d card(s).\n", result.Projects, result.Boards, result.Cards)
		return nil
	}}
}

func newTasksProjectsCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{Use: "projects", Short: "List cached task projects", RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := taskstore.New(rt.dataPath).ListProjects()
		if err != nil {
			return err
		}
		var currentID int64
		if prefs, err := rt.loadPrefs(); err == nil {
			currentID = prefs.TasksProjectID
		}
		rows := make([][]string, 0, len(projects))
		for _, project := range projects {
			marker := ""
			if currentID != 0 && project.Meta.RemoteID == currentID {
				marker = "*"
			}
			rows = append(rows, []string{marker, strconv.FormatInt(project.Meta.RemoteID, 10), project.Meta.Name, project.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04"), project.Path})
		}
		writeRows(cmd.OutOrStdout(), []string{"current", "id", "name", "updated_at", "path"}, rows)
		return nil
	}}
}

func newTasksProjectCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Task project commands"}
	cmd.AddCommand(&cobra.Command{Use: "show [id]", Short: "Show a cached task project (defaults to current)", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveProjectID(rt, args)
		if err != nil {
			return err
		}
		project, err := taskstore.New(rt.dataPath).ReadProject(id)
		if err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"id", strconv.FormatInt(project.Meta.RemoteID, 10)}, {"name", project.Meta.Name}, {"updated_at", project.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04")}, {"path", project.Path}})
		return nil
	}})
	cmd.AddCommand(newTasksProjectCreateCommand(rt))
	cmd.AddCommand(newTasksProjectRenameCommand(rt))
	cmd.AddCommand(&cobra.Command{Use: "delete <id>", Short: "Delete a remote task project and local cache", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		if err := service.DeleteProject(rt.context(), id); err != nil {
			return err
		}
		if err := taskstore.New(rt.dataPath).DeleteProject(id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted task project %d.\n", id)
		return nil
	}})
	return cmd
}

func newTasksProjectCreateCommand(rt *runtime) *cobra.Command {
	var name string
	cmd := &cobra.Command{Use: "create", Short: "Create a remote task project", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("--name is required")
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		project, err := service.CreateProject(rt.context(), tasks.ProjectInput{Name: name})
		if err != nil {
			return err
		}
		if err := storeTaskProject(taskstore.New(rt.dataPath), *project, time.Now()); err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, taskProjectFields(*project))
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "", "project name")
	return cmd
}

func newTasksProjectRenameCommand(rt *runtime) *cobra.Command {
	var name string
	cmd := &cobra.Command{Use: "rename <id>", Short: "Rename a remote task project", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("--name is required")
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		project, err := service.UpdateProject(rt.context(), id, tasks.ProjectInput{Name: name})
		if err != nil {
			return err
		}
		if err := storeTaskProject(taskstore.New(rt.dataPath), *project, time.Now()); err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, taskProjectFields(*project))
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "", "project name")
	return cmd
}

func newTasksBoardCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "board", Short: "Task board commands"}
	cmd.AddCommand(&cobra.Command{Use: "show [project-id]", Short: "Show a cached task board (defaults to current project)", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProjectID(rt, args)
		if err != nil {
			return err
		}
		board, err := taskstore.New(rt.dataPath).ReadBoard(projectID)
		if err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"id", strconv.FormatInt(board.Meta.RemoteID, 10)}, {"title", board.Meta.Title}, {"filename", board.Meta.Filename}, {"path", board.Path}})
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", board.Body)
		return nil
	}})
	cmd.AddCommand(newTasksBoardUpdateCommand(rt))
	return cmd
}

func newTasksBoardUpdateCommand(rt *runtime) *cobra.Command {
	var body, filePath string
	cmd := &cobra.Command{Use: "update [project-id]", Short: "Update a remote task board markdown body (defaults to current project)", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProjectID(rt, args)
		if err != nil {
			return err
		}
		body, err := taskBodyFromFlags(body, filePath)
		if err != nil {
			return err
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		board, err := service.UpdateBoard(rt.context(), projectID, tasks.BoardInput{Body: body})
		if err != nil {
			return err
		}
		if err := taskstore.New(rt.dataPath).StoreBoard(projectID, *board, time.Now()); err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, [][]string{{"id", strconv.FormatInt(board.ID, 10)}, {"title", board.Title}, {"filename", board.Filename}})
		return nil
	}}
	cmd.Flags().StringVar(&body, "body", "", "Markdown board body")
	cmd.Flags().StringVar(&filePath, "file", "", "read Markdown board body from file")
	return cmd
}

func newTasksCardsCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{Use: "cards [project-id]", Short: "List cached task cards (defaults to current project)", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProjectID(rt, args)
		if err != nil {
			return err
		}
		cards, err := taskstore.New(rt.dataPath).ListCards(projectID)
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(cards))
		for _, card := range cards {
			rows = append(rows, []string{strconv.FormatInt(card.Meta.RemoteID, 10), card.Meta.Title, card.Meta.Filename, card.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04"), card.Path})
		}
		writeRows(cmd.OutOrStdout(), []string{"id", "title", "filename", "updated_at", "path"}, rows)
		return nil
	}}
}

func newTasksCardCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "card", Short: "Task card commands"}
	cmd.AddCommand(&cobra.Command{Use: "show [project-id] <card-id>", Short: "Show a cached task card (project-id defaults to current)", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := resolveProjectAndCardID(rt, args)
		if err != nil {
			return err
		}
		card, err := taskstore.New(rt.dataPath).ReadCard(projectID, id)
		if err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, cachedTaskCardFields(*card))
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", card.Body)
		return nil
	}})
	cmd.AddCommand(newTasksCardCreateCommand(rt))
	cmd.AddCommand(newTasksCardEditCommand(rt))
	cmd.AddCommand(newTasksCardMoveCommand(rt))
	cmd.AddCommand(&cobra.Command{Use: "delete [project-id] <card-id>", Short: "Delete a remote task card and local cache (project-id defaults to current)", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := resolveProjectAndCardID(rt, args)
		if err != nil {
			return err
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		if err := service.DeleteCard(rt.context(), projectID, id); err != nil {
			return err
		}
		if err := taskstore.New(rt.dataPath).DeleteCard(projectID, id); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted task card %d.\n", id)
		return nil
	}})
	return cmd
}

func newTasksCardCreateCommand(rt *runtime) *cobra.Command {
	var title, body, filePath string
	cmd := &cobra.Command{Use: "create [project-id]", Short: "Create a remote task card (defaults to current project)", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProjectID(rt, args)
		if err != nil {
			return err
		}
		if strings.TrimSpace(title) == "" {
			return fmt.Errorf("--title is required")
		}
		body, err := taskBodyFromFlags(body, filePath)
		if err != nil {
			return err
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		card, err := service.CreateCard(rt.context(), projectID, tasks.CardInput{Title: title, Body: body})
		if err != nil {
			return err
		}
		if err := taskstore.New(rt.dataPath).StoreCard(projectID, *card, time.Now()); err != nil {
			return err
		}
		if err := addTaskCardToTodo(rt.context(), taskstore.New(rt.dataPath), service, projectID, *card); err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, taskCardFields(*card))
		return nil
	}}
	cmd.Flags().StringVar(&title, "title", "", "card title")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	cmd.Flags().StringVar(&filePath, "file", "", "read Markdown body from file")
	return cmd
}

func newTasksCardEditCommand(rt *runtime) *cobra.Command {
	var title, body, filePath string
	cmd := &cobra.Command{Use: "edit [project-id] <card-id>", Short: "Update a remote task card (project-id defaults to current)", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := resolveProjectAndCardID(rt, args)
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("body") && !cmd.Flags().Changed("file") {
			return fmt.Errorf("provide --title, --body, or --file")
		}
		current, err := taskCardForEdit(rt, projectID, id)
		if err != nil {
			return err
		}
		input := tasks.CardInput{Title: current.Title, Body: current.Body}
		if cmd.Flags().Changed("title") {
			input.Title = title
		}
		if cmd.Flags().Changed("body") || cmd.Flags().Changed("file") {
			input.Body, err = taskBodyFromFlags(body, filePath)
			if err != nil {
				return err
			}
		}
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		card, err := service.UpdateCard(rt.context(), projectID, id, input)
		if err != nil {
			return err
		}
		if err := replaceTaskCardBoardLink(rt.context(), taskstore.New(rt.dataPath), service, projectID, current.Filename, *card); err != nil {
			return err
		}
		if err := taskstore.New(rt.dataPath).StoreCard(projectID, *card, time.Now()); err != nil {
			return err
		}
		writeRows(cmd.OutOrStdout(), []string{"key", "value"}, taskCardFields(*card))
		return nil
	}}
	cmd.Flags().StringVar(&title, "title", "", "card title")
	cmd.Flags().StringVar(&body, "body", "", "Markdown body")
	cmd.Flags().StringVar(&filePath, "file", "", "read Markdown body from file")
	return cmd
}

func newTasksCardMoveCommand(rt *runtime) *cobra.Command {
	var column string
	cmd := &cobra.Command{Use: "move [project-id] <card-id>", Short: "Move a task card to a column (project-id defaults to current)", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := resolveProjectAndCardID(rt, args)
		if err != nil {
			return err
		}
		if strings.TrimSpace(column) == "" {
			return fmt.Errorf("--column is required")
		}
		store := taskstore.New(rt.dataPath)
		service, err := tasksService(rt)
		if err != nil {
			return err
		}
		card, err := taskCardForEdit(rt, projectID, id)
		if err != nil {
			return err
		}
		if err := moveTaskCardToColumn(rt.context(), store, service, projectID, card.Filename, column); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Moved card %d to %s.\n", id, column)
		return nil
	}}
	cmd.Flags().StringVar(&column, "column", "", "target column name (e.g. Todo, Doing, Done)")
	return cmd
}

func tasksService(rt *runtime) (*tasks.Service, error) {
	client, err := rt.apiClient()
	if err != nil {
		return nil, err
	}
	return tasks.NewService(client), nil
}

func runTasksSync(ctx context.Context, store taskstore.Store, service *tasks.Service) (tasksSyncResult, error) {
	return taskssync.Run(ctx, store, service)
}

func storeTaskProject(store taskstore.Store, project tasks.Project, syncedAt time.Time) error {
	return taskssync.StoreProject(store, project, syncedAt)
}

func taskCardForEdit(rt *runtime, projectID, id int64) (*tasks.Card, error) {
	if cached, err := taskstore.New(rt.dataPath).ReadCard(projectID, id); err == nil {
		return &tasks.Card{TaskFile: tasks.TaskFile{ID: cached.Meta.RemoteID, UserID: cached.Meta.UserID, FolderID: cached.Meta.FolderID, Title: cached.Meta.Title, Filename: cached.Meta.Filename, MIMEType: cached.Meta.MIMEType, CreatedAt: cached.Meta.RemoteCreatedAt, UpdatedAt: cached.Meta.RemoteUpdatedAt}, Body: cached.Body}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	service, err := tasksService(rt)
	if err != nil {
		return nil, err
	}
	return service.ShowCard(rt.context(), projectID, id)
}

func addTaskCardToTodo(ctx context.Context, store taskstore.Store, service *tasks.Service, projectID int64, card tasks.Card) error {
	board, err := service.ShowBoard(ctx, projectID)
	if err != nil {
		return err
	}
	updatedBody := tasks.AddCardToColumn(board.Body, "Todo", "cards/"+card.Filename)
	if updatedBody == board.Body {
		return store.StoreBoard(projectID, *board, time.Now())
	}
	updated, err := service.UpdateBoard(ctx, projectID, tasks.BoardInput{Body: updatedBody})
	if err != nil {
		return err
	}
	return store.StoreBoard(projectID, *updated, time.Now())
}

func moveTaskCardToColumn(ctx context.Context, store taskstore.Store, service *tasks.Service, projectID int64, cardFilename, targetColumn string) error {
	if cardFilename == "" || targetColumn == "" {
		return fmt.Errorf("card filename and target column are required")
	}
	board, err := service.ShowBoard(ctx, projectID)
	if err != nil {
		return err
	}
	updatedBody := tasks.MoveCardToColumn(board.Body, "cards/"+cardFilename, targetColumn)
	if updatedBody == board.Body {
		return store.StoreBoard(projectID, *board, time.Now())
	}
	updated, err := service.UpdateBoard(ctx, projectID, tasks.BoardInput{Body: updatedBody})
	if err != nil {
		return err
	}
	return store.StoreBoard(projectID, *updated, time.Now())
}

func replaceTaskCardBoardLink(ctx context.Context, store taskstore.Store, service *tasks.Service, projectID int64, oldFilename string, card tasks.Card) error {
	if oldFilename == "" || oldFilename == card.Filename {
		return nil
	}
	board, err := service.ShowBoard(ctx, projectID)
	if err != nil {
		return err
	}
	updatedBody := tasks.ReplaceCardPath(board.Body, "cards/"+oldFilename, "cards/"+card.Filename)
	if updatedBody == board.Body {
		return nil
	}
	updated, err := service.UpdateBoard(ctx, projectID, tasks.BoardInput{Body: updatedBody})
	if err != nil {
		return err
	}
	return store.StoreBoard(projectID, *updated, time.Now())
}

func taskBodyFromFlags(body, filePath string) (string, error) {
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

func parseTwoIDs(args []string) (int64, int64, error) {
	first, err := parseID(args[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := parseID(args[1])
	if err != nil {
		return 0, 0, err
	}
	return first, second, nil
}

func taskProjectFields(project tasks.Project) [][]string {
	return [][]string{{"id", strconv.FormatInt(project.ID, 10)}, {"name", project.Name}, {"updated_at", project.UpdatedAt.Format("2006-01-02 15:04")}}
}

func taskCardFields(card tasks.Card) [][]string {
	return [][]string{{"id", strconv.FormatInt(card.ID, 10)}, {"folder_id", strconv.FormatInt(card.FolderID, 10)}, {"title", card.Title}, {"filename", card.Filename}, {"updated_at", card.UpdatedAt.Format("2006-01-02 15:04")}}
}

func cachedTaskCardFields(card taskstore.CachedCard) [][]string {
	return [][]string{{"id", strconv.FormatInt(card.Meta.RemoteID, 10)}, {"project_id", strconv.FormatInt(card.Meta.ProjectID, 10)}, {"folder_id", strconv.FormatInt(card.Meta.FolderID, 10)}, {"title", card.Meta.Title}, {"filename", card.Meta.Filename}, {"updated_at", card.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04")}, {"path", card.Path}}
}
