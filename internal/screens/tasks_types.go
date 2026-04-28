package screens

import (
	"context"

	"charm.land/bubbles/v2/list"
	"github.com/elpdev/telex-cli/internal/tasks"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

type TasksSyncFunc func(context.Context) (TasksSyncResult, error)
type CreateTaskProjectFunc func(context.Context, tasks.ProjectInput) (*tasks.Project, error)
type CreateTaskCardFunc func(context.Context, int64, tasks.CardInput) (*tasks.Card, error)
type UpdateTaskCardFunc func(context.Context, int64, int64, tasks.CardInput) (*tasks.Card, error)
type DeleteTaskCardFunc func(context.Context, int64, int64) error
type MoveTaskCardFunc func(ctx context.Context, projectID int64, cardFilename, targetColumn string) error

type TasksSyncResult struct {
	Projects int
	Boards   int
	Cards    int
}

type taskRow struct {
	Kind    string
	Name    string
	Project *taskstore.CachedProject
	Column  *tasks.BoardColumn
	Card    *taskstore.CachedCard
	Missing bool
}

type taskListItem struct{ row taskRow }

func (i taskListItem) FilterValue() string { return i.row.Name }

type tasksLoadedMsg struct {
	projects []taskstore.CachedProject
	project  *taskstore.CachedProject
	board    *taskstore.CachedBoard
	cards    []taskstore.CachedCard
	err      error
}

type tasksSyncedMsg struct {
	result TasksSyncResult
	loaded tasksLoadedMsg
	err    error
}

type taskActionFinishedMsg struct {
	status string
	loaded tasksLoadedMsg
	err    error
}

type TasksActionMsg struct{ Action string }

type TasksSelection struct {
	Kind    string
	Subject string
	HasItem bool
}

type taskListDelegate struct{ simpleDelegate }

var _ list.Item = taskListItem{}
