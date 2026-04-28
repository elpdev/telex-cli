package taskssync

import (
	"context"
	"time"

	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

type Result struct {
	Projects int
	Boards   int
	Cards    int
}

func Run(ctx context.Context, store taskstore.Store, service *tasks.Service) (Result, error) {
	syncedAt := time.Now()
	workspace, err := service.Workspace(ctx)
	if err != nil {
		return Result{}, err
	}
	if err := store.StoreWorkspace(workspace, syncedAt); err != nil {
		return Result{}, err
	}
	updatedSince := latestTaskUpdatedSince(store)
	projects, err := listAllProjects(ctx, service, updatedSince)
	if err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, summary := range projects {
		project, err := service.ShowProject(ctx, summary.ID)
		if err != nil {
			return result, err
		}
		if err := StoreProject(store, *project, syncedAt); err != nil {
			return result, err
		}
		result.Projects++
	}
	cachedProjects, err := store.ListProjects()
	if err != nil {
		return result, err
	}
	for _, cached := range cachedProjects {
		projectID := cached.Meta.RemoteID
		board, err := service.ShowBoard(ctx, projectID)
		if err != nil {
			return result, err
		}
		if err := store.StoreBoard(projectID, *board, syncedAt); err != nil {
			return result, err
		}
		result.Boards++
		cards, err := listAllCards(ctx, service, projectID, updatedSince)
		if err != nil {
			return result, err
		}
		for _, card := range cards {
			if err := store.StoreCard(projectID, card, syncedAt); err != nil {
				return result, err
			}
			result.Cards++
		}
	}
	return result, nil
}

func listAllProjects(ctx context.Context, service *tasks.Service, updatedSince string) ([]tasks.Project, error) {
	page := 1
	all := []tasks.Project{}
	for {
		projects, pagination, err := service.ListProjects(ctx, tasks.ListParams{Page: page, PerPage: 100, UpdatedSince: updatedSince})
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

func listAllCards(ctx context.Context, service *tasks.Service, projectID int64, updatedSince string) ([]tasks.Card, error) {
	page := 1
	all := []tasks.Card{}
	for {
		cards, pagination, err := service.ListCards(ctx, projectID, tasks.ListParams{Page: page, PerPage: 100, UpdatedSince: updatedSince})
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

func latestTaskUpdatedSince(store taskstore.Store) string {
	projects, err := store.ListProjects()
	if err != nil {
		return ""
	}
	var latest time.Time
	for _, project := range projects {
		if project.Meta.RemoteUpdatedAt.After(latest) {
			latest = project.Meta.RemoteUpdatedAt
		}
		if board, err := store.ReadBoard(project.Meta.RemoteID); err == nil && board.Meta.RemoteUpdatedAt.After(latest) {
			latest = board.Meta.RemoteUpdatedAt
		}
		cards, err := store.ListCards(project.Meta.RemoteID)
		if err != nil {
			continue
		}
		for _, card := range cards {
			if card.Meta.RemoteUpdatedAt.After(latest) {
				latest = card.Meta.RemoteUpdatedAt
			}
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339Nano)
}

func StoreProject(store taskstore.Store, project tasks.Project, syncedAt time.Time) error {
	if err := store.StoreProject(project, syncedAt); err != nil {
		return err
	}
	if project.Board != nil {
		_ = store.StoreBoard(project.ID, tasks.Board{TaskFile: *project.Board}, syncedAt)
	}
	for _, card := range project.Cards {
		if err := store.StoreCard(project.ID, card, syncedAt); err != nil {
			return err
		}
	}
	return nil
}
