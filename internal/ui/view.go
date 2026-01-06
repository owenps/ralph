package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenps/ralph/internal"
)

type filterType int

const (
	filterAll filterType = iota
	filterBug
	filterFeature
	filterRefactor
)

type viewModel struct {
	tasks    []internal.Task
	filtered []internal.Task
	cursor   int
	filter   filterType
	width    int
	height   int
}

type backToMenuMsg struct{}

func newViewTasks(tasks []internal.Task) viewModel {
	m := viewModel{
		tasks:  tasks,
		cursor: 0,
		filter: filterAll,
	}
	m.applyFilter()
	return m
}

func (m *viewModel) SetTasks(tasks []internal.Task) {
	m.tasks = tasks
	m.applyFilter()
}

func (m *viewModel) applyFilter() {
	switch m.filter {
	case filterAll:
		m.filtered = m.tasks
	case filterBug:
		m.filtered = filterByCategory(m.tasks, internal.CategoryBug)
	case filterFeature:
		m.filtered = filterByCategory(m.tasks, internal.CategoryFeature)
	case filterRefactor:
		m.filtered = filterByCategory(m.tasks, internal.CategoryRefactor)
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func filterByCategory(tasks []internal.Task, cat internal.Category) []internal.Task {
	var result []internal.Task
	for _, t := range tasks {
		if t.Category == cat {
			result = append(result, t)
		}
	}
	return result
}

func (m viewModel) Update(msg tea.Msg) (viewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, vKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, vKeys.Escape):
			return m, func() tea.Msg { return backToMenuMsg{} }
		case key.Matches(msg, vKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, vKeys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case key.Matches(msg, vKeys.Tab):
			m.filter = (m.filter + 1) % 4
			m.applyFilter()
		}
	}

	return m, nil
}

func (m viewModel) View() string {
	s := m.renderTabs() + "\n\n"

	if len(m.filtered) == 0 {
		s += m.renderEmpty()
	} else {
		s += m.renderSplitView()
	}

	s += "\n"
	s += renderHelp(viewHelpKeys)

	return s
}

func (m viewModel) renderTabs() string {
	tabs := []struct {
		label  string
		filter filterType
	}{
		{"all", filterAll},
		{"bug", filterBug},
		{"feature", filterFeature},
		{"refactor", filterRefactor},
	}

	var tabViews []string
	for _, tab := range tabs {
		style := inactiveTabStyle
		if m.filter == tab.filter {
			style = activeTabStyle
		}
		tabViews = append(tabViews, style.Render(tab.label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)
}

func (m viewModel) renderEmpty() string {
	return helpDescStyle.Render("No tasks yet. Enjoy the calm.")
}

func (m viewModel) renderSplitView() string {
	totalWidth := m.width
	if totalWidth < 60 {
		totalWidth = 80
	}
	listWidth := totalWidth / 3
	detailWidth := totalWidth - listWidth - 5

	listContent := m.renderTaskList(listWidth)
	listPanel := borderStyle.
		Width(listWidth).
		Render(listContent)

	detailContent := m.renderTaskDetail(detailWidth)
	detailPanel := borderStyle.
		Width(detailWidth).
		Render(detailContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)
}

func (m viewModel) renderTaskList(width int) string {
	var lines []string

	for i, t := range m.filtered {
		cursor := "  "
		style := menuItemStyle

		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
			style = selectedMenuItemStyle
		}

		badge := categoryBadge(string(t.Category))
		desc := truncate(t.Description, width-15)

		doneMarker := ""
		if t.Done {
			doneMarker = successStyle.Render(" ✔︎")
		}

		line := cursor + badge.Render(string(t.Category)[:3]) + " " + style.Render(desc) + doneMarker
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m viewModel) renderTaskDetail(width int) string {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return helpDescStyle.Render("No task selected")
	}

	t := m.filtered[m.cursor]
	var s strings.Builder

	badge := categoryBadge(string(t.Category))
	s.WriteString(detailLabelStyle.Render("Category: "))
	s.WriteString(badge.Render(string(t.Category)))
	s.WriteString("\n\n")

	s.WriteString(detailLabelStyle.Render("Description:"))
	s.WriteString("\n")
	s.WriteString(detailValueStyle.Render(wrapText(t.Description, width-4)))
	s.WriteString("\n\n")

	if len(t.Steps) > 0 {
		s.WriteString(detailLabelStyle.Render("Steps:"))
		s.WriteString("\n")
		for i, step := range t.Steps {
			s.WriteString(detailValueStyle.Render(fmt.Sprintf("  %d. %s", i+1, step)))
			s.WriteString("\n")
		}
	} else {
		s.WriteString(detailLabelStyle.Render("Steps: "))
		s.WriteString(helpDescStyle.Render("(none)"))
		s.WriteString("\n")
	}

	s.WriteString("\n")

	s.WriteString(detailLabelStyle.Render("Status: "))
	if t.Done {
		s.WriteString(successStyle.Render("Complete"))
	} else {
		s.WriteString(helpDescStyle.Render("Incomplete"))
	}

	return s.String()
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 20
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(s string, width int) string {
	if width <= 0 {
		width = 40
	}
	if len(s) <= width {
		return s
	}

	var lines []string
	for len(s) > width {
		idx := strings.LastIndex(s[:width], " ")
		if idx == -1 {
			idx = width
		}
		lines = append(lines, s[:idx])
		s = strings.TrimSpace(s[idx:])
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}

type viewKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Tab    key.Binding
	Escape key.Binding
	Quit   key.Binding
}

var vKeys = viewKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}
