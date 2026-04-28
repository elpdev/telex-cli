package screens

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

type Tasks struct {
	store         taskstore.Store
	sync          TasksSyncFunc
	createProject CreateTaskProjectFunc
	createCard    CreateTaskCardFunc
	updateCard    UpdateTaskCardFunc
	deleteCard    DeleteTaskCardFunc
	moveCard      MoveTaskCardFunc
	projects      []taskstore.CachedProject
	project       *taskstore.CachedProject
	board         *taskstore.CachedBoard
	cards         []taskstore.CachedCard
	rows          []taskRow
	rowList       list.Model
	index         int
	detail        *taskstore.CachedCard
	detailScroll  int
	filter        string
	filtering     bool
	picker        string
	picking       bool
	confirm       string
	loading       bool
	syncing       bool
	err           error
	status        string
	keys          TasksKeyMap
}

func NewTasks(store taskstore.Store, sync TasksSyncFunc) Tasks {
	return Tasks{store: store, sync: sync, loading: true, keys: DefaultTasksKeyMap(), rowList: newTaskList(nil, 0, 0, 0)}
}

func (t Tasks) WithActions(createProject CreateTaskProjectFunc, createCard CreateTaskCardFunc, updateCard UpdateTaskCardFunc, deleteCard DeleteTaskCardFunc, moveCard MoveTaskCardFunc) Tasks {
	t.createProject = createProject
	t.createCard = createCard
	t.updateCard = updateCard
	t.deleteCard = deleteCard
	t.moveCard = moveCard
	return t
}

func (t Tasks) Init() tea.Cmd { return t.loadCmd(0) }
