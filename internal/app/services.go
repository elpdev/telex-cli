package app

import (
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/calendar"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/contacts"
	"github.com/elpdev/telex-cli/internal/drive"
	"github.com/elpdev/telex-cli/internal/mail"
	"github.com/elpdev/telex-cli/internal/notes"
	"github.com/elpdev/telex-cli/internal/tasks"
)

func (m *Model) mailService() (*mail.Service, error) {
	if m.client == nil {
		configFile, tokenFile := config.Paths(m.configPath)
		cfg, err := config.LoadFrom(configFile)
		if err != nil {
			return nil, err
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		m.client = api.NewClient(cfg, tokenFile)
	}
	return mail.NewService(m.client), nil
}

func (m *Model) driveService() (*drive.Service, *config.Config, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return drive.NewService(m.client), cfg, nil
}

func (m *Model) notesService() (*notes.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return notes.NewService(m.client), nil
}

func (m *Model) tasksService() (*tasks.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return tasks.NewService(m.client), nil
}

func (m *Model) contactsService() (*contacts.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return contacts.NewService(m.client), nil
}

func (m *Model) calendarService() (*calendar.Service, error) {
	configFile, tokenFile := config.Paths(m.configPath)
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if m.client == nil {
		m.client = api.NewClient(cfg, tokenFile)
	}
	return calendar.NewService(m.client), nil
}
