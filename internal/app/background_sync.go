package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const mailAutoSyncInterval = 30 * time.Minute

type startupSyncFinishedMsg struct {
	errors []string
}

type mailAutoSyncTickMsg struct{}

type backgroundMailSyncedMsg struct {
	source  string
	skipped bool
	err     error
}

func (m Model) startupSyncCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		errs := []string{}
		if _, err := m.syncDrive(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("drive: %v", err))
		}
		if _, err := m.syncNotes(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("notes: %v", err))
		}
		if _, err := m.syncTasks(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("tasks: %v", err))
		}
		if _, err := m.syncCalendar(ctx, "", ""); err != nil {
			errs = append(errs, fmt.Sprintf("calendar: %v", err))
		}
		if _, err := m.syncContacts(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("contacts: %v", err))
		}
		return startupSyncFinishedMsg{errors: errs}
	}
}

func mailAutoSyncTickCmd() tea.Cmd {
	return tea.Tick(mailAutoSyncInterval, func(time.Time) tea.Msg {
		return mailAutoSyncTickMsg{}
	})
}

func (m Model) backgroundMailSyncCmd(source string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.syncMail(context.Background())
		return backgroundMailSyncedMsg{source: source, skipped: err == errMailSyncAlreadyRunning, err: err}
	}
}

func (m Model) logStartupSync(msg startupSyncFinishedMsg) {
	if len(msg.errors) == 0 {
		m.logs.Info("Startup background sync completed")
		return
	}
	m.logs.Warn("Startup background sync completed with errors: " + strings.Join(msg.errors, "; "))
}
