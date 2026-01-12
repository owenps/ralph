package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenps/ralph/internal"
)

type settingField int

const (
	fieldDefaultMaxIterations settingField = iota
	fieldModel
	fieldAllowedTools
	fieldMaxTurns
	fieldTimeout
	fieldSystemPrompt
	fieldCount
)

type settingsTab int

const (
	tabLoop settingsTab = iota
	tabClaude
	tabCount
)

type settingsModel struct {
	config      *internal.GlobalConfig
	activeTab   settingsTab
	cursor      settingField
	editing     bool
	textInput   textinput.Model
	textArea    textarea.Model
	toolsCursor int
	toolsSelect map[string]bool
	width       int
	height      int
}

type settingsSavedMsg struct{}
type backFromSettingsMsg struct{}

var availableTools = []string{
	"Read", "Write", "Edit", "Bash", "Glob", "Grep",
	"WebFetch", "WebSearch", "Task", "TodoWrite",
}

func newSettingsView(cfg *internal.GlobalConfig) settingsModel {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.Width = 30

	ta := textarea.New()
	ta.SetWidth(60)
	ta.SetHeight(5)
	ta.CharLimit = 1000
	ta.Placeholder = "Enter system prompt..."

	toolsSelect := make(map[string]bool)
	for _, tool := range cfg.Claude.AllowedTools {
		toolsSelect[tool] = true
	}

	return settingsModel{
		config:      cfg,
		cursor:      fieldDefaultMaxIterations,
		textInput:   ti,
		textArea:    ta,
		toolsSelect: toolsSelect,
	}
}

func (m settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.editing {
			return m.updateEditing(msg)
		}
		return m.updateNavigation(msg)
	}

	return m, nil
}

func (m settingsModel) fieldsForTab() (first, last settingField) {
	switch m.activeTab {
	case tabLoop:
		return fieldDefaultMaxIterations, fieldDefaultMaxIterations
	case tabClaude:
		return fieldModel, fieldSystemPrompt
	}
	return fieldDefaultMaxIterations, fieldSystemPrompt
}

func (m settingsModel) updateNavigation(msg tea.KeyMsg) (settingsModel, tea.Cmd) {
	first, last := m.fieldsForTab()

	switch {
	case key.Matches(msg, settingsKeys.Tab):
		m.activeTab = (m.activeTab + 1) % tabCount
		newFirst, _ := m.fieldsForTab()
		m.cursor = newFirst
		return m, nil
	case key.Matches(msg, settingsKeys.Up):
		if m.cursor > first {
			m.cursor--
		}
	case key.Matches(msg, settingsKeys.Down):
		if m.cursor < last {
			m.cursor++
		}
	case key.Matches(msg, settingsKeys.Enter):
		m.editing = true
		m.textInput.Reset()

		switch m.cursor {
		case fieldDefaultMaxIterations:
			m.textInput.SetValue(strconv.Itoa(m.config.Loop.DefaultMaxIterations))
			m.textInput.Focus()
			return m, textinput.Blink
		case fieldModel:
			m.textInput.SetValue(m.config.Claude.Model)
			m.textInput.Focus()
			return m, textinput.Blink
		case fieldAllowedTools:
			m.toolsCursor = 0
			return m, nil
		case fieldMaxTurns:
			m.textInput.SetValue(strconv.Itoa(m.config.Claude.MaxTurns))
			m.textInput.Focus()
			return m, textinput.Blink
		case fieldTimeout:
			m.textInput.SetValue(strconv.Itoa(m.config.Claude.TimeoutSeconds))
			m.textInput.Focus()
			return m, textinput.Blink
		case fieldSystemPrompt:
			m.textArea.SetValue(m.config.Claude.SystemPrompt)
			m.textArea.Focus()
			return m, textarea.Blink
		}
	case key.Matches(msg, settingsKeys.Back):
		return m, func() tea.Msg { return backFromSettingsMsg{} }
	}
	return m, nil
}

func (m settingsModel) updateEditing(msg tea.KeyMsg) (settingsModel, tea.Cmd) {
	if m.cursor == fieldAllowedTools {
		return m.updateToolsEditing(msg)
	}

	if m.cursor == fieldSystemPrompt {
		return m.updateTextAreaEditing(msg)
	}

	switch {
	case key.Matches(msg, settingsKeys.Escape):
		m.editing = false
		m.textInput.Blur()
		return m, nil
	case key.Matches(msg, settingsKeys.Enter):
		m.editing = false
		m.textInput.Blur()
		m.applyTextValue()
		return m, m.saveConfig
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m settingsModel) updateTextAreaEditing(msg tea.KeyMsg) (settingsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, settingsKeys.Escape):
		m.editing = false
		m.textArea.Blur()
		m.config.Claude.SystemPrompt = m.textArea.Value()
		return m, m.saveConfig
	default:
		var cmd tea.Cmd
		m.textArea, cmd = m.textArea.Update(msg)
		return m, cmd
	}
}

func (m settingsModel) updateToolsEditing(msg tea.KeyMsg) (settingsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, settingsKeys.Escape), key.Matches(msg, settingsKeys.Enter):
		m.editing = false
		m.applyToolsSelection()
		return m, m.saveConfig
	case key.Matches(msg, settingsKeys.Up):
		if m.toolsCursor > 0 {
			m.toolsCursor--
		}
	case key.Matches(msg, settingsKeys.Down):
		if m.toolsCursor < len(availableTools)-1 {
			m.toolsCursor++
		}
	case key.Matches(msg, settingsKeys.Space):
		tool := availableTools[m.toolsCursor]
		m.toolsSelect[tool] = !m.toolsSelect[tool]
	}
	return m, nil
}

func (m *settingsModel) applyTextValue() {
	value := strings.TrimSpace(m.textInput.Value())

	switch m.cursor {
	case fieldDefaultMaxIterations:
		if v, err := strconv.Atoi(value); err == nil && v > 0 {
			m.config.Loop.DefaultMaxIterations = v
		}
	case fieldModel:
		m.config.Claude.Model = value
	case fieldMaxTurns:
		if v, err := strconv.Atoi(value); err == nil && v >= 0 {
			m.config.Claude.MaxTurns = v
		}
	case fieldTimeout:
		if v, err := strconv.Atoi(value); err == nil && v > 0 {
			m.config.Claude.TimeoutSeconds = v
		}
	case fieldSystemPrompt:
		m.config.Claude.SystemPrompt = value
	}
}

func (m *settingsModel) applyToolsSelection() {
	var tools []string
	for _, tool := range availableTools {
		if m.toolsSelect[tool] {
			tools = append(tools, tool)
		}
	}
	m.config.Claude.AllowedTools = tools
}

func (m settingsModel) saveConfig() tea.Msg {
	if err := internal.SaveGlobalConfig(m.config); err != nil {
		return errMsg{err: err}
	}
	return settingsSavedMsg{}
}

func (m settingsModel) renderSettingsTabs() string {
	tabs := []struct {
		label string
		tab   settingsTab
	}{
		{"loop", tabLoop},
		{"claude", tabClaude},
	}

	var tabViews []string
	for _, tab := range tabs {
		style := inactiveTabStyle
		if m.activeTab == tab.tab {
			style = activeTabStyle
		}
		tabViews = append(tabViews, style.Render(tab.label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)
}

func (m settingsModel) View() string {
	s := m.renderSettingsTabs() + "\n\n"

	switch m.activeTab {
	case tabLoop:
		s += m.renderSection("", []settingRow{
			{field: fieldDefaultMaxIterations, label: "Default MAX Iterations", value: strconv.Itoa(m.config.Loop.DefaultMaxIterations)},
		})
	case tabClaude:
		s += m.renderSection("", []settingRow{
			{field: fieldModel, label: "Model", value: m.displayModel()},
			{field: fieldAllowedTools, label: "Allowed Tools", value: m.displayTools()},
			{field: fieldMaxTurns, label: "Max Turns", value: m.displayMaxTurns()},
			{field: fieldTimeout, label: "Timeout (seconds)", value: strconv.Itoa(m.config.Claude.TimeoutSeconds)},
			{field: fieldSystemPrompt, label: "System Prompt", value: m.displaySystemPrompt()},
		})

		if m.editing && m.cursor == fieldAllowedTools {
			s += "\n" + m.renderToolsEditor()
		}
		if m.editing && m.cursor == fieldSystemPrompt {
			s += "\n" + m.renderTextAreaEditor()
		}
	}

	s += "\n"
	if m.editing {
		if m.cursor == fieldAllowedTools {
			s += renderHelp(toolsEditKeys)
		} else if m.cursor == fieldSystemPrompt {
			s += renderHelp(textAreaEditKeys)
		} else {
			s += renderHelp(inputKeys)
		}
	} else {
		s += renderHelp(settingsHelpKeys)
	}

	return s
}

type settingRow struct {
	field settingField
	label string
	value string
}

func (m settingsModel) renderSection(title string, rows []settingRow) string {
	var s string
	if title != "" {
		s = detailLabelStyle.Render(title) + "\n"
	}

	for _, row := range rows {
		cursor := "  "
		if m.cursor == row.field {
			cursor = cursorStyle.Render("> ")
		}

		label := helpDescStyle.Render(row.label + ": ")
		var value string

		if m.editing && m.cursor == row.field && row.field != fieldAllowedTools {
			value = m.textInput.View()
		} else {
			value = detailValueStyle.Render(row.value)
		}

		s += cursor + label + value + "\n"
	}

	return s
}

func (m settingsModel) renderToolsEditor() string {
	s := detailLabelStyle.Render("  Select tools (space to toggle):") + "\n"

	for i, tool := range availableTools {
		cursor := "    "
		if m.toolsCursor == i {
			cursor = cursorStyle.Render("  > ")
		}

		checkbox := "[ ]"
		if m.toolsSelect[tool] {
			checkbox = "[x]"
		}

		s += cursor + checkbox + " " + tool + "\n"
	}

	return s
}

func (m settingsModel) renderTextAreaEditor() string {
	lines := strings.Split(m.textArea.View(), "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func (m settingsModel) displayModel() string {
	if m.config.Claude.Model == "" {
		return "(default)"
	}
	return m.config.Claude.Model
}

func (m settingsModel) displayTools() string {
	if len(m.config.Claude.AllowedTools) == 0 {
		return "(default)"
	}
	return fmt.Sprintf("%d tools", len(m.config.Claude.AllowedTools))
}

func (m settingsModel) displayMaxTurns() string {
	if m.config.Claude.MaxTurns == 0 {
		return "(default)"
	}
	return strconv.Itoa(m.config.Claude.MaxTurns)
}

func (m settingsModel) displaySystemPrompt() string {
	if m.config.Claude.SystemPrompt == "" {
		return "(none)"
	}
	// Truncate for display
	s := m.config.Claude.SystemPrompt
	if len(s) > 40 {
		s = s[:37] + "..."
	}
	return s
}

type settingsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Escape key.Binding
	Back   key.Binding
	Space  key.Binding
	Tab    key.Binding
}

var settingsKeys = settingsKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
	Back: key.NewBinding(
		key.WithKeys("q"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
	),
}

var settingsHelpKeys = []keyBinding{
	{Key: "tab", Desc: "switch tab"},
	{Key: "j/k", Desc: "navigate"},
	{Key: "enter", Desc: "edit"},
	{Key: "q", Desc: "menu"},
}

var toolsEditKeys = []keyBinding{
	{Key: "j/k", Desc: "navigate"},
	{Key: "space", Desc: "toggle"},
	{Key: "enter", Desc: "done"},
}

var textAreaEditKeys = []keyBinding{
	{Key: "esc", Desc: "save & exit"},
}
