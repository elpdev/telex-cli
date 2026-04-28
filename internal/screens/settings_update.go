package screens

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (s Settings) Update(msg tea.Msg) (Screen, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	if s.mode == settingsModeThemes {
		return s.updateThemeSelect(keyMsg)
	}

	switch {
	case key.Matches(keyMsg, s.keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
		s.confirming = ""
		return s, nil
	case key.Matches(keyMsg, s.keys.Down):
		if s.cursor < len(focusableSettingsRowIdx)-1 {
			s.cursor++
		}
		s.confirming = ""
		return s, nil
	case key.Matches(keyMsg, s.keys.Back):
		s.confirming = ""
		return s, nil
	case key.Matches(keyMsg, s.keys.Enter):
		return s.activateRow()
	}
	return s, nil
}

func (s Settings) activateRow() (Screen, tea.Cmd) {
	row := s.focusedRow()
	switch row.id {
	case "theme":
		if len(s.themes) == 0 {
			return s, nil
		}
		s.mode = settingsModeThemes
		s.preTheme = s.state.ThemeName
		s.themeList = newSettingsThemeList(s.themes, s.state.ThemeName, s.preTheme, s.th, s.themeList.Width(), s.themeList.Height())
		return s, nil
	case "sidebar-visible":
		next := !s.state.SidebarVisible
		return s, func() tea.Msg { return SettingsSidebarChangedMsg{Visible: next} }
	case "drive-sync":
		next := nextDriveSyncMode(s.state.DriveSyncMode)
		return s, func() tea.Msg { return SettingsDriveSyncChangedMsg{Mode: next} }
	case "mail-admin":
		if s.actions.OpenMailAdmin != nil {
			return s, s.actions.OpenMailAdmin()
		}
		return s, nil
	case "sign-out":
		if s.confirming != "sign-out" {
			s.confirming = "sign-out"
			return s, nil
		}
		s.confirming = ""
		if s.actions.SignOut != nil {
			return s, s.actions.SignOut()
		}
		return s, nil
	case "open-data-dir":
		if s.actions.OpenPath != nil && s.state.DataDir != "" {
			return s, s.actions.OpenPath(s.state.DataDir)
		}
		return s, nil
	case "open-config-dir":
		if s.actions.OpenPath != nil && s.state.ConfigDir != "" {
			return s, s.actions.OpenPath(s.state.ConfigDir)
		}
		return s, nil
	}
	return s, nil
}

func (s Settings) updateThemeSelect(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch {
	case key.Matches(msg, s.keys.Enter):
		name, ok := s.selectedThemeName()
		if !ok {
			return s, nil
		}
		s.mode = settingsModeNormal
		s.preTheme = ""
		return s, func() tea.Msg { return SettingsThemeChangedMsg{Name: name} }
	case key.Matches(msg, s.keys.Back):
		original := s.preTheme
		s.mode = settingsModeNormal
		s.preTheme = ""
		return s, func() tea.Msg { return SettingsThemeCancelMsg{Name: original} }
	}
	previous, _ := s.selectedThemeName()
	updated, cmd := s.themeList.Update(msg)
	s.themeList = updated
	current, _ := s.selectedThemeName()
	if current != "" && current != previous {
		return s, tea.Batch(cmd, s.previewThemeCmd())
	}
	return s, cmd
}

func (s Settings) previewThemeCmd() tea.Cmd {
	name, ok := s.selectedThemeName()
	if !ok {
		return nil
	}
	return func() tea.Msg { return SettingsThemePreviewMsg{Name: name} }
}

func (s Settings) selectedThemeName() (string, bool) {
	item, ok := s.themeList.SelectedItem().(settingsThemeItem)
	if !ok {
		return "", false
	}
	return item.name, true
}

func nextDriveSyncMode(current string) string {
	if current == driveSyncMetadataOnly {
		return driveSyncFull
	}
	return driveSyncMetadataOnly
}

func (s Settings) focusedRow() settingsRowDef {
	if s.cursor < 0 || s.cursor >= len(focusableSettingsRowIdx) {
		return settingsRowDef{}
	}
	return settingsRows[focusableSettingsRowIdx[s.cursor]]
}
