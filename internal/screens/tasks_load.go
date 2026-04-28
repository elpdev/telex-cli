package screens

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/taskstore"
)

func (t Tasks) loadCmd(projectID int64) tea.Cmd { return func() tea.Msg { return t.load(projectID) } }

func (t Tasks) load(projectID int64) tasksLoadedMsg {
	projects, err := t.store.ListProjects()
	if err != nil {
		return tasksLoadedMsg{err: err}
	}
	var project *taskstore.CachedProject
	if projectID == 0 && len(projects) == 1 {
		projectID = projects[0].Meta.RemoteID
	}
	if projectID > 0 {
		for i := range projects {
			if projects[i].Meta.RemoteID == projectID {
				project = &projects[i]
				break
			}
		}
	}
	if project == nil {
		return tasksLoadedMsg{projects: projects}
	}
	board, boardErr := t.store.ReadBoard(project.Meta.RemoteID)
	if boardErr != nil && !os.IsNotExist(boardErr) {
		return tasksLoadedMsg{err: boardErr}
	}
	cards, err := t.store.ListCards(project.Meta.RemoteID)
	return tasksLoadedMsg{projects: projects, project: project, board: board, cards: cards, err: err}
}

func (t Tasks) syncCmd() tea.Cmd {
	projectID := t.currentProjectID()
	return func() tea.Msg {
		result, err := t.sync(context.Background())
		loaded := t.load(projectID)
		if err == nil {
			err = loaded.err
		}
		return tasksSyncedMsg{result: result, loaded: loaded, err: err}
	}
}

func (t Tasks) createProjectCmd() tea.Cmd {
	if t.createProject == nil {
		t.status = "Create project is not configured"
		return nil
	}
	return func() tea.Msg {
		input, err := editTaskProjectTemplate(defaultTitle)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		project, err := t.createProject(context.Background(), input)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(project.ID)
		return taskActionFinishedMsg{status: "Created " + project.Name, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) createCardCmd() tea.Cmd {
	if t.createCard == nil {
		t.status = "Create card is not configured"
		return nil
	}
	projectID := t.currentProjectID()
	if projectID == 0 {
		t.status = "Open a project before creating a card"
		return nil
	}
	return func() tea.Msg {
		input, err := editTaskCardTemplate(defaultTitle, "")
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		card, err := t.createCard(context.Background(), projectID, input)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(projectID)
		return taskActionFinishedMsg{status: "Created " + card.Title, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) editCardCmd() tea.Cmd {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		t.status = "Select a card to edit"
		return nil
	}
	return t.editCachedCardCmd(*row.Card)
}

func (t Tasks) editCachedCardCmd(cached taskstore.CachedCard) tea.Cmd {
	if t.updateCard == nil {
		t.status = "Edit card is not configured"
		return nil
	}
	projectID := cached.Meta.ProjectID
	return func() tea.Msg {
		input, err := editTaskCardTemplate(cached.Meta.Title, cached.Body)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		card, err := t.updateCard(context.Background(), projectID, cached.Meta.RemoteID, input)
		if err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(projectID)
		return taskActionFinishedMsg{status: "Updated " + card.Title, loaded: loaded, err: loaded.err}
	}
}

func (t Tasks) deleteCardCmd() tea.Cmd {
	row, ok := t.selectedRow()
	if !ok || row.Card == nil {
		return nil
	}
	if t.deleteCard == nil {
		return func() tea.Msg { return taskActionFinishedMsg{err: fmt.Errorf("delete card is not configured")} }
	}
	card := *row.Card
	return func() tea.Msg {
		if err := t.deleteCard(context.Background(), card.Meta.ProjectID, card.Meta.RemoteID); err != nil {
			return taskActionFinishedMsg{err: err}
		}
		loaded := t.load(card.Meta.ProjectID)
		return taskActionFinishedMsg{status: "Deleted " + card.Meta.Title, loaded: loaded, err: loaded.err}
	}
}
