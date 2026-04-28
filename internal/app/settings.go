package app

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/opener"
	"github.com/elpdev/telex-cli/internal/screens"
	"github.com/elpdev/telex-cli/internal/theme"
)

func (m *Model) buildSettings() screens.Settings {
	state := m.computeSettingsState()
	actions := m.settingsActions()
	if existing, ok := m.screens["settings"].(screens.Settings); ok {
		return existing.Reconfigure(state, m.theme, theme.BuiltIns(), actions)
	}
	return screens.NewSettings(state, m.theme, theme.BuiltIns(), actions)
}

func (m *Model) computeSettingsState() screens.SettingsState {
	configFile, tokenFile := config.Paths(m.configPath)

	driveSync := config.DriveSyncFull
	if cfg, err := config.LoadFrom(configFile); err == nil && cfg != nil {
		driveSync = cfg.DriveSyncMode()
	}

	authStatus := "Not signed in"
	signedIn := false
	if tc, err := config.LoadTokenFrom(tokenFile); err == nil && tc != nil && tc.Token != "" {
		if tc.Valid() {
			signedIn = true
			authStatus = "Signed in · expires in " + humanDuration(time.Until(tc.ExpiresAt))
		} else {
			authStatus = "Token expired"
		}
	}

	return screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Instance:       m.instance,
		AuthStatus:     authStatus,
		SignedIn:       signedIn,
		DataDir:        m.dataPath,
		ConfigDir:      m.configDirPath(),
		CacheSize:      computeCacheSize(m.dataPath),
		DriveSyncMode:  driveSync,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	}
}

func (m *Model) settingsActions() screens.SettingsActions {
	return screens.SettingsActions{
		OpenPath: func(path string) tea.Cmd {
			return func() tea.Msg {
				if path == "" {
					return nil
				}
				cmd, err := opener.Command(path)
				if err == nil {
					_ = startDetached(cmd)
				}
				return nil
			}
		},
		OpenURL: func(target string) tea.Cmd {
			return func() tea.Msg {
				if target == "" {
					return nil
				}
				cmd, err := opener.Command(target)
				if err == nil {
					_ = startDetached(cmd)
				}
				return nil
			}
		},
		OpenMailAdmin: func() tea.Cmd {
			return func() tea.Msg { return routeMsg{"mail-admin"} }
		},
		SignOut: func() tea.Cmd {
			return func() tea.Msg { return settingsSignOutMsg{} }
		},
	}
}

func (m *Model) configDirPath() string {
	if m.configPath != "" {
		return filepath.Dir(filepath.Clean(m.configPath))
	}
	return config.Dir()
}

func computeCacheSize(dataPath string) int64 {
	if dataPath == "" {
		return 0
	}
	var size int64
	_ = filepath.WalkDir(dataPath, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func humanDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
