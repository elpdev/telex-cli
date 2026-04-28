package screens

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/theme"
)

type SettingsState struct {
	ThemeName      string
	SidebarVisible bool

	Instance   string
	AuthStatus string
	SignedIn   bool

	DataDir       string
	ConfigDir     string
	CacheSize     int64
	DriveSyncMode string

	Version string
	Commit  string
	Date    string
}

type SettingsActions struct {
	OpenPath      func(path string) tea.Cmd
	OpenURL       func(url string) tea.Cmd
	OpenMailAdmin func() tea.Cmd
	SignOut       func() tea.Cmd
}

type SettingsThemePreviewMsg struct{ Name string }

type SettingsThemeChangedMsg struct{ Name string }

type SettingsThemeCancelMsg struct{ Name string }

type SettingsSidebarChangedMsg struct{ Visible bool }

type SettingsDriveSyncChangedMsg struct{ Mode string }

const (
	driveSyncFull         = "full"
	driveSyncMetadataOnly = "metadata_only"
)

type settingsMode int

const (
	settingsModeNormal settingsMode = iota
	settingsModeThemes
)

type settingsRowKind int

const (
	rowSection settingsRowKind = iota
	rowReadonly
	rowToggle
	rowSelect
	rowAction
)

type settingsRowDef struct {
	kind  settingsRowKind
	id    string
	label string
}

var settingsRows = []settingsRowDef{
	{kind: rowSection, id: "section-appearance", label: "Appearance"},
	{kind: rowSelect, id: "theme", label: "Theme"},
	{kind: rowToggle, id: "sidebar-visible", label: "Sidebar at start"},

	{kind: rowSection, id: "section-account", label: "Account"},
	{kind: rowReadonly, id: "instance", label: "Instance"},
	{kind: rowReadonly, id: "auth-status", label: "Status"},
	{kind: rowAction, id: "mail-admin", label: "Mail Admin"},
	{kind: rowAction, id: "sign-out", label: "Sign out"},

	{kind: rowSection, id: "section-storage", label: "Storage"},
	{kind: rowReadonly, id: "data-dir", label: "Data dir"},
	{kind: rowReadonly, id: "cache-size", label: "Cache size"},
	{kind: rowSelect, id: "drive-sync", label: "Drive sync"},
	{kind: rowAction, id: "open-data-dir", label: "Open data dir"},
	{kind: rowAction, id: "open-config-dir", label: "Open config dir"},

	{kind: rowSection, id: "section-build", label: "Build"},
	{kind: rowReadonly, id: "version", label: "Version"},
	{kind: rowReadonly, id: "commit", label: "Commit"},
	{kind: rowReadonly, id: "date", label: "Date"},
}

var focusableSettingsRowIdx = func() []int {
	out := make([]int, 0, len(settingsRows))
	for i, r := range settingsRows {
		if r.kind != rowSection {
			out = append(out, i)
		}
	}
	return out
}()

type Settings struct {
	state   SettingsState
	th      theme.Theme
	themes  []theme.Theme
	actions SettingsActions

	cursor     int
	mode       settingsMode
	themeList  list.Model
	preTheme   string
	confirming string
	keys       settingsKeyMap
}

type settingsThemeItem struct {
	name string
	was  bool
}

func (i settingsThemeItem) FilterValue() string { return i.name }

type settingsKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Back  key.Binding
}
