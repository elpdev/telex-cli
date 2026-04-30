package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elpdev/telex-cli/internal/frontmatter"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskssync"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

func (m *Model) syncTasks(ctx context.Context) (screens.TasksSyncResult, error) {
	service, err := m.tasksService()
	if err != nil {
		return screens.TasksSyncResult{}, err
	}
	result, err := runTasksSync(ctx, taskstore.New(m.dataPath), service)
	return screens.TasksSyncResult{Projects: result.Projects, Boards: result.Boards, Cards: result.Cards}, err
}

func (m *Model) createTaskProject(ctx context.Context, input tasks.ProjectInput) (*tasks.Project, error) {
	service, err := m.tasksService()
	if err != nil {
		return nil, err
	}
	project, err := service.CreateProject(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := storeTaskProject(taskstore.New(m.dataPath), *project, time.Now()); err != nil {
		return nil, err
	}
	return project, nil
}

func (m *Model) createTaskCard(ctx context.Context, projectID int64, input tasks.CardInput) (*tasks.Card, error) {
	service, err := m.tasksService()
	if err != nil {
		return nil, err
	}
	card, err := service.CreateCard(ctx, projectID, input)
	if err != nil {
		return nil, err
	}
	if err := taskstore.New(m.dataPath).StoreCard(projectID, *card, time.Now()); err != nil {
		return nil, err
	}
	if err := addTaskCardToColumn(ctx, taskstore.New(m.dataPath), service, projectID, *card, taskCardColumnFromBody(input.Body, "Todo")); err != nil {
		return nil, err
	}
	return card, nil
}

func (m *Model) updateTaskCard(ctx context.Context, projectID, id int64, input tasks.CardInput) (*tasks.Card, error) {
	service, err := m.tasksService()
	if err != nil {
		return nil, err
	}
	oldFilename := ""
	if cached, err := taskstore.New(m.dataPath).ReadCard(projectID, id); err == nil {
		oldFilename = cached.Meta.Filename
	}
	card, err := service.UpdateCard(ctx, projectID, id, input)
	if err != nil {
		return nil, err
	}
	if err := replaceTaskCardBoardLink(ctx, taskstore.New(m.dataPath), service, projectID, oldFilename, *card); err != nil {
		return nil, err
	}
	if column := taskCardColumnFromBody(input.Body, ""); column != "" {
		if err := moveTaskCardToColumn(ctx, taskstore.New(m.dataPath), service, projectID, card.Filename, column); err != nil {
			return nil, err
		}
	}
	if err := taskstore.New(m.dataPath).StoreCard(projectID, *card, time.Now()); err != nil {
		return nil, err
	}
	return card, nil
}

func (m *Model) deleteTaskCard(ctx context.Context, projectID, id int64) error {
	service, err := m.tasksService()
	if err != nil {
		return err
	}
	if err := service.DeleteCard(ctx, projectID, id); err != nil {
		return err
	}
	return taskstore.New(m.dataPath).DeleteCard(projectID, id)
}

func (m *Model) moveTaskCard(ctx context.Context, projectID int64, cardFilename, targetColumn string) error {
	service, err := m.tasksService()
	if err != nil {
		return err
	}
	return moveTaskCardToColumn(ctx, taskstore.New(m.dataPath), service, projectID, cardFilename, targetColumn)
}

type tasksSyncResult = taskssync.Result

func runTasksSync(ctx context.Context, store taskstore.Store, service *tasks.Service) (tasksSyncResult, error) {
	return taskssync.Run(ctx, store, service)
}

func storeTaskProject(store taskstore.Store, project tasks.Project, syncedAt time.Time) error {
	return taskssync.StoreProject(store, project, syncedAt)
}

func addTaskCardToColumn(ctx context.Context, store taskstore.Store, service *tasks.Service, projectID int64, card tasks.Card, column string) error {
	if strings.TrimSpace(column) == "" {
		column = "Todo"
	}
	board, err := service.ShowBoard(ctx, projectID)
	if err != nil {
		return err
	}
	updatedBody := tasks.AddCardToColumn(board.Body, column, "cards/"+card.Filename)
	if updatedBody == board.Body {
		return store.StoreBoard(projectID, *board, time.Now())
	}
	updated, err := service.UpdateBoard(ctx, projectID, tasks.BoardInput{Body: updatedBody})
	if err != nil {
		return err
	}
	return store.StoreBoard(projectID, *updated, time.Now())
}

func taskCardColumnFromBody(body, fallback string) string {
	doc, err := frontmatter.Parse(body)
	if err != nil {
		return fallback
	}
	column := strings.TrimSpace(doc.Fields["column"])
	if column == "" {
		return fallback
	}
	return column
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
