package screens

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func defaultSettingsKeyMap() settingsKeyMap {
	return settingsKeyMap{
		Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "previous row")),
		Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "next row")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "activate")),
		Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	}
}

func NewSettings(state SettingsState, th theme.Theme, themes []theme.Theme, actions SettingsActions) Settings {
	return Settings{
		state:     state,
		th:        th,
		themes:    themes,
		actions:   actions,
		themeList: newSettingsThemeList(themes, state.ThemeName, state.ThemeName, th, 0, 0),
		keys:      defaultSettingsKeyMap(),
	}
}

func (s Settings) Reconfigure(state SettingsState, th theme.Theme, themes []theme.Theme, actions SettingsActions) Settings {
	s.state = state
	s.th = th
	s.themes = themes
	s.actions = actions
	if s.cursor < 0 {
		s.cursor = 0
	}
	if s.cursor >= len(focusableSettingsRowIdx) {
		s.cursor = len(focusableSettingsRowIdx) - 1
	}
	selected := state.ThemeName
	if s.mode == settingsModeThemes {
		if name, ok := s.selectedThemeName(); ok {
			selected = name
		}
	}
	s.themeList = newSettingsThemeList(s.themes, selected, s.preTheme, s.th, s.themeList.Width(), s.themeList.Height())
	return s
}

func (s Settings) Init() tea.Cmd { return nil }

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

func (s Settings) View(width, height int) string {
	if s.mode == settingsModeThemes {
		return s.themeSelectView(width, height)
	}
	var b strings.Builder
	focusedRowIdx := -1
	if s.cursor >= 0 && s.cursor < len(focusableSettingsRowIdx) {
		focusedRowIdx = focusableSettingsRowIdx[s.cursor]
	}
	for i, row := range settingsRows {
		switch row.kind {
		case rowSection:
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(s.th.Title.Render(row.label))
			b.WriteString("\n")
		default:
			line := s.formatRow(row, width)
			if i == focusedRowIdx {
				line = s.th.Selected.Render(line)
			} else {
				line = s.th.Text.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(s.th.Muted.Render("↑/↓ move · enter activate · esc back"))
	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func (s Settings) formatRow(row settingsRowDef, width int) string {
	const indent = "  "
	const labelCol = 20

	var line string
	switch row.kind {
	case rowAction:
		text := "› " + row.label
		if s.confirming == row.id {
			text += "   press enter again to confirm"
		}
		line = indent + text
	default:
		label := padRight(row.label, labelCol)
		value := s.rowValue(row)
		switch row.kind {
		case rowSelect:
			value = padRight(value, 16) + " ›"
		case rowToggle:
			if s.toggleValue(row.id) {
				value = padRight(value, 16) + " ●"
			} else {
				value = padRight(value, 16) + " ○"
			}
		}
		line = indent + label + "  " + value
	}
	return padRight(line, width)
}

func (s Settings) rowValue(row settingsRowDef) string {
	switch row.id {
	case "theme":
		return valueOrDash(s.state.ThemeName)
	case "sidebar-visible":
		if s.state.SidebarVisible {
			return "on"
		}
		return "off"
	case "instance":
		return valueOrDash(s.state.Instance)
	case "auth-status":
		return valueOrDash(s.state.AuthStatus)
	case "mail-admin":
		return "Manage domains and inboxes"
	case "data-dir":
		return valueOrDash(s.state.DataDir)
	case "cache-size":
		if s.state.CacheSize <= 0 {
			return "0 B"
		}
		return formatBytes(s.state.CacheSize)
	case "drive-sync":
		return valueOrDash(s.state.DriveSyncMode)
	case "version":
		return valueOrDash(s.state.Version)
	case "commit":
		return valueOrDash(s.state.Commit)
	case "date":
		return valueOrDash(s.state.Date)
	}
	return ""
}

func (s Settings) toggleValue(id string) bool {
	switch id {
	case "sidebar-visible":
		return s.state.SidebarVisible
	}
	return false
}

func (s Settings) themeSelectView(width, height int) string {
	s.themeList.SetSize(width, max(1, height-4))
	var b strings.Builder
	b.WriteString(s.th.Title.Render("Theme"))
	b.WriteString("\n")
	b.WriteString(s.th.Muted.Render("Move to preview · enter selects · esc reverts"))
	b.WriteString("\n\n")
	b.WriteString(s.themeList.View())
	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func newSettingsThemeList(themes []theme.Theme, selected, original string, th theme.Theme, width, height int) list.Model {
	items := make([]list.Item, 0, len(themes))
	selectedIdx := 0
	for i, t := range themes {
		items = append(items, settingsThemeItem{name: t.Name, was: t.Name == original})
		if t.Name == selected {
			selectedIdx = i
		}
	}

	m := list.New(items, settingsThemeDelegate{th: th}, width, height)
	m.SetShowTitle(false)
	m.SetShowFilter(false)
	m.SetFilteringEnabled(false)
	m.SetShowStatusBar(false)
	m.SetShowHelp(false)
	m.DisableQuitKeybindings()
	m.Select(selectedIdx)
	return m
}

type settingsThemeDelegate struct{ th theme.Theme }

func (d settingsThemeDelegate) Height() int  { return 1 }
func (d settingsThemeDelegate) Spacing() int { return 0 }
func (d settingsThemeDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d settingsThemeDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	themeItem, ok := item.(settingsThemeItem)
	if !ok {
		return
	}
	marker := "  "
	label := themeItem.name
	if themeItem.was {
		label += "  (was)"
	}
	line := padRight(marker+label, m.Width())
	if index == m.Index() {
		line = padRight("▸ "+label, m.Width())
		_, _ = io.WriteString(w, d.th.Selected.Render(line))
		return
	}
	_, _ = io.WriteString(w, d.th.Text.Render(line))
}

func valueOrDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (s Settings) Title() string { return "Settings" }

func (s Settings) KeyBindings() []key.Binding {
	return []key.Binding{s.keys.Up, s.keys.Down, s.keys.Enter, s.keys.Back}
}
