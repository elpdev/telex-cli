package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskstore"
	"github.com/spf13/cobra"
)

type tasksSyncResult struct {
	Projects int
	Boards   int
	Cards    int
}

func newTasksCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "tasks", Short: "Tasks commands"}
	cmd.AddCommand(newTasksSyncCommand(rt))
	cmd.AddCommand(newTasksProjectsCommand(rt))
	cmd.AddCommand(newTasksProjectCommand(rt))
	cmd.AddCommand(newTasksBoardCommand(rt))
	cmd.AddCommand(newTasksCardsCommand(rt))
	cmd.AddCommand(newTasksCardCommand(rt))
	return cmd
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
		rows := make([][]string, 0, len(projects))
		for _, project := range projects {
			rows = append(rows, []string{strconv.FormatInt(project.Meta.RemoteID, 10), project.Meta.Name, project.Meta.RemoteUpdatedAt.Format("2006-01-02 15:04"), project.Path})
		}
		writeRows(cmd.OutOrStdout(), []string{"id", "name", "updated_at", "path"}, rows)
		return nil
	}}
}

func newTasksProjectCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Task project commands"}
	cmd.AddCommand(&cobra.Command{Use: "show <id>", Short: "Show a cached task project", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
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
	cmd.AddCommand(&cobra.Command{Use: "show <project-id>", Short: "Show a cached task board", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID(args[0])
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
	cmd := &cobra.Command{Use: "update <project-id>", Short: "Update a remote task board markdown body", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID(args[0])
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
	return &cobra.Command{Use: "cards <project-id>", Short: "List cached task cards", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID(args[0])
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
	cmd.AddCommand(&cobra.Command{Use: "show <project-id> <card-id>", Short: "Show a cached task card", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := parseTwoIDs(args)
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
	cmd.AddCommand(&cobra.Command{Use: "delete <project-id> <card-id>", Short: "Delete a remote task card and local cache", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := parseTwoIDs(args)
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
	cmd := &cobra.Command{Use: "create <project-id>", Short: "Create a remote task card", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID(args[0])
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
	cmd := &cobra.Command{Use: "edit <project-id> <card-id>", Short: "Update a remote task card", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		projectID, id, err := parseTwoIDs(args)
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

func tasksService(rt *runtime) (*tasks.Service, error) {
	client, err := rt.apiClient()
	if err != nil {
		return nil, err
	}
	return tasks.NewService(client), nil
}

func runTasksSync(ctx context.Context, store taskstore.Store, service *tasks.Service) (tasksSyncResult, error) {
	syncedAt := time.Now()
	workspace, err := service.Workspace(ctx)
	if err != nil {
		return tasksSyncResult{}, err
	}
	if err := store.StoreWorkspace(workspace, syncedAt); err != nil {
		return tasksSyncResult{}, err
	}
	projects, err := listAllTaskProjects(ctx, service)
	if err != nil {
		return tasksSyncResult{}, err
	}
	result := tasksSyncResult{}
	for _, summary := range projects {
		project, err := service.ShowProject(ctx, summary.ID)
		if err != nil {
			return result, err
		}
		if err := storeTaskProject(store, *project, syncedAt); err != nil {
			return result, err
		}
		result.Projects++
		board, err := service.ShowBoard(ctx, project.ID)
		if err != nil {
			return result, err
		}
		if err := store.StoreBoard(project.ID, *board, syncedAt); err != nil {
			return result, err
		}
		result.Boards++
		cards, err := listAllTaskCards(ctx, service, project.ID)
		if err != nil {
			return result, err
		}
		for _, card := range cards {
			if err := store.StoreCard(project.ID, card, syncedAt); err != nil {
				return result, err
			}
			result.Cards++
		}
	}
	return result, nil
}

func listAllTaskProjects(ctx context.Context, service *tasks.Service) ([]tasks.Project, error) {
	page := 1
	all := []tasks.Project{}
	for {
		projects, pagination, err := service.ListProjects(ctx, tasks.ListParams{Page: page, PerPage: 100})
		if err != nil {
			return all, err
		}
		all = append(all, projects...)
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount || len(projects) == 0 {
			return all, nil
		}
		page++
	}
}

func listAllTaskCards(ctx context.Context, service *tasks.Service, projectID int64) ([]tasks.Card, error) {
	page := 1
	all := []tasks.Card{}
	for {
		cards, pagination, err := service.ListCards(ctx, projectID, tasks.ListParams{Page: page, PerPage: 100})
		if err != nil {
			return all, err
		}
		all = append(all, cards...)
		if pagination == nil || page*pagination.PerPage >= pagination.TotalCount || len(cards) == 0 {
			return all, nil
		}
		page++
	}
}

func storeTaskProject(store taskstore.Store, project tasks.Project, syncedAt time.Time) error {
	if err := store.StoreProject(project, syncedAt); err != nil {
		return err
	}
	if project.Board != nil {
		board := tasks.Board{TaskFile: *project.Board}
		_ = store.StoreBoard(project.ID, board, syncedAt)
	}
	for _, card := range project.Cards {
		if err := store.StoreCard(project.ID, card, syncedAt); err != nil {
			return err
		}
	}
	return nil
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
