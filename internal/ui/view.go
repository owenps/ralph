package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenps/ralph/internal"
)

const deleteConfirmWindow = 2 * time.Second

type filterType int

const (
	filterAll filterType = iota
	filterBug
	filterFeature
	filterRefactor
	filterResearch
	filterNotes
)

// taskItem wraps internal.Task to implement list.Item
type taskItem struct {
	task internal.Task
}

func (i taskItem) Title() string       { return i.task.Description }
func (i taskItem) Description() string { return string(i.task.Category) }
func (i taskItem) FilterValue() string { return i.task.Description }

// taskDelegate is a custom delegate for rendering tasks
type taskDelegate struct {
	selected *map[string]bool
}

func (d taskDelegate) Height() int                             { return 1 }
func (d taskDelegate) Spacing() int                            { return 0 }
func (d taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(taskItem)
	if !ok {
		return
	}

	t := ti.task
	cursor := "  "
	style := menuItemStyle

	if index == m.Index() {
		cursor = cursorStyle.Render("> ")
		style = selectedMenuItemStyle
	}

	// Selection indicator
	selectMark := "  "
	if d.selected != nil && (*d.selected)[t.ID] {
		selectMark = successStyle.Render("● ")
	}

	badge := categoryBadge(string(t.Category))
	width := m.Width() - 4
	if width < 20 {
		width = 20
	}
	desc := truncate(t.Description, width-12)

	doneMarker := ""
	if t.Done {
		doneMarker = successStyle.Render("✔︎")
	}

	line := cursor + selectMark + badge.Render(string(t.Category)[:3]) + " " + style.Render(desc) + doneMarker
	fmt.Fprint(w, line)
}

type viewModel struct {
	tasks           []internal.Task
	filter          filterType
	width           int
	height          int
	lastDeletePress time.Time
	deleteConfirmID string
	selected        map[string]bool
	showRunWarn     bool
	showSelectWarn  bool
	help            help.Model
	keys            viewKeyMap
	list            list.Model
}

type deleteTaskMsg struct{ ID string }
type toggleTaskMsg struct{ ID string }
type editTaskMsg struct{ Task internal.Task }
type clearDeletePromptMsg struct{}
type clearRunWarnMsg struct{}
type clearSelectWarnMsg struct{}
type runSelectedTasksMsg struct{ TaskIDs []string }
type showLoopMsg struct{}
type openCreateMsg struct{}
type openSettingsMsg struct{}

func newViewTasks(tasks []internal.Task) viewModel {
	h := help.New()
	h.ShowAll = true // Use full help view to avoid truncation/ellipsis
	h.Styles.ShortKey = helpKeyStyle
	h.Styles.ShortDesc = helpDescStyle
	h.Styles.ShortSeparator = helpDescStyle
	h.Styles.FullKey = helpKeyStyle
	h.Styles.FullDesc = helpDescStyle
	h.Styles.FullSeparator = helpDescStyle

	selected := make(map[string]bool)

	// Create list with custom delegate
	delegate := taskDelegate{selected: &selected}
	items := tasksToItems(tasks)
	l := list.New(items, delegate, 40, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	m := viewModel{
		tasks:    tasks,
		filter:   filterAll,
		selected: selected,
		help:     h,
		keys:     vKeys,
		list:     l,
	}
	m.applyFilter()
	return m
}

func tasksToItems(tasks []internal.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	return items
}

func (m *viewModel) SetTasks(tasks []internal.Task) {
	m.tasks = tasks
	m.applyFilter()
}

func (m *viewModel) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	m.width = width
	m.height = height
	m.help.Width = width
	m.list.SetWidth(width * 2 / 5)
	m.list.SetHeight(height - 10)
}

func (m *viewModel) applyFilter() {
	var filtered []internal.Task
	switch m.filter {
	case filterAll:
		filtered = m.tasks
	case filterBug:
		filtered = filterByCategory(m.tasks, internal.CategoryBug)
	case filterFeature:
		filtered = filterByCategory(m.tasks, internal.CategoryFeature)
	case filterRefactor:
		filtered = filterByCategory(m.tasks, internal.CategoryRefactor)
	case filterResearch:
		filtered = filterByCategory(m.tasks, internal.CategoryResearch)
	case filterNotes:
		filtered = filterByCategory(m.tasks, internal.CategoryNotes)
	}

	m.list.SetItems(tasksToItems(filtered))
}

func (m *viewModel) currentTask() *internal.Task {
	if item := m.list.SelectedItem(); item != nil {
		if ti, ok := item.(taskItem); ok {
			return &ti.task
		}
	}
	return nil
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
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case clearDeletePromptMsg:
		m.lastDeletePress = time.Time{}
		m.deleteConfirmID = ""
		return m, nil

	case clearRunWarnMsg:
		m.showRunWarn = false
		return m, nil

	case clearSelectWarnMsg:
		m.showSelectWarn = false
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, vKeys.Back):
			return m, func() tea.Msg { return backToMenuMsg{} }
		case key.Matches(msg, vKeys.Escape):
			// Clear delete prompt or selection
			if !m.lastDeletePress.IsZero() {
				m.lastDeletePress = time.Time{}
				m.deleteConfirmID = ""
				return m, nil
			}
			if len(m.selected) > 0 {
				m.selected = make(map[string]bool)
			}
			return m, nil
		case key.Matches(msg, vKeys.New):
			return m, func() tea.Msg { return openCreateMsg{} }
		case key.Matches(msg, vKeys.Tab):
			m.filter = (m.filter + 1) % 6
			m.applyFilter()
			m.lastDeletePress = time.Time{}
			m.deleteConfirmID = ""
			m.showRunWarn = false
			m.showSelectWarn = false
		case key.Matches(msg, vKeys.Toggle):
			if task := m.currentTask(); task != nil {
				return m, func() tea.Msg { return toggleTaskMsg{ID: task.ID} }
			}
		case key.Matches(msg, vKeys.Delete):
			if task := m.currentTask(); task != nil {
				now := time.Now()
				if !m.lastDeletePress.IsZero() && m.deleteConfirmID == task.ID && now.Sub(m.lastDeletePress) < deleteConfirmWindow {
					m.lastDeletePress = time.Time{}
					m.deleteConfirmID = ""
					return m, func() tea.Msg { return deleteTaskMsg{ID: task.ID} }
				}
				m.lastDeletePress = now
				m.deleteConfirmID = task.ID
				return m, tea.Tick(deleteConfirmWindow, func(time.Time) tea.Msg {
					return clearDeletePromptMsg{}
				})
			}
		case key.Matches(msg, vKeys.Select):
			if task := m.currentTask(); task != nil {
				if task.Done {
					m.showSelectWarn = true
					return m, tea.Tick(deleteConfirmWindow, func(time.Time) tea.Msg {
						return clearSelectWarnMsg{}
					})
				}
				if m.selected[task.ID] {
					delete(m.selected, task.ID)
				} else {
					m.selected[task.ID] = true
				}
			}
		case key.Matches(msg, vKeys.Run):
			if len(m.selected) > 0 {
				var taskIDs []string
				for id := range m.selected {
					taskIDs = append(taskIDs, id)
				}
				return m, func() tea.Msg { return runSelectedTasksMsg{TaskIDs: taskIDs} }
			}
			m.showRunWarn = true
			return m, tea.Tick(deleteConfirmWindow, func(time.Time) tea.Msg {
				return clearRunWarnMsg{}
			})
		case key.Matches(msg, vKeys.ShowLoop):
			return m, func() tea.Msg { return showLoopMsg{} }
		case key.Matches(msg, vKeys.Edit):
			if task := m.currentTask(); task != nil {
				t := *task
				return m, func() tea.Msg { return editTaskMsg{Task: t} }
			}
		}
	}

	// Update list for navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// Clear delete prompt on navigation
	if _, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(msg.(tea.KeyMsg), vKeys.Up) || key.Matches(msg.(tea.KeyMsg), vKeys.Down) {
			m.lastDeletePress = time.Time{}
			m.deleteConfirmID = ""
			m.showRunWarn = false
			m.showSelectWarn = false
		}
	}

	return m, cmd
}

func (m viewModel) View() string {
	if len(m.list.Items()) == 0 {
		s := m.renderTabs() + "\n\n"
		s += m.renderEmpty()
		return s
	}

	s := appTitle() + "\n\n"
	s += m.renderTabs()

	if len(m.selected) > 0 {
		s += "  " + successStyle.Render(fmt.Sprintf("%d selected", len(m.selected)))
	}

	// Warnings appear inline after tabs
	if !m.lastDeletePress.IsZero() {
		s += "  " + warnStyle.Render("Press d again to delete")
	}
	if m.showRunWarn {
		s += "  " + warnStyle.Render("No tasks selected")
	}
	if m.showSelectWarn {
		s += "  " + warnStyle.Render("Cannot select completed task")
	}

	s += "\n\n"
	s += m.renderSplitView()

	s += "\n"
	s += m.help.View(m.keys)

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
		{"research", filterResearch},
		{"notes", filterNotes},
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
	s := helpDescStyle.Render("No tasks yet. Enjoy the peace.") + "\n\n"
	s += renderHelp([]keyBinding{
		{Key: "n", Desc: "new task"},
		{Key: "q", Desc: "menu"},
	})
	return s
}

func (m viewModel) renderSplitView() string {
	totalWidth := m.width
	if totalWidth < 60 {
		totalWidth = 80
	}
	listWidth := totalWidth * 2 / 5
	detailWidth := totalWidth - listWidth - 5

	// Use list's built-in view
	listPanel := borderStyle.
		Width(listWidth).
		Render(m.list.View())

	detailContent := m.renderTaskDetail(detailWidth)
	detailPanel := borderStyle.
		Width(detailWidth).
		Render(detailContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)
}

func (m viewModel) renderTaskDetail(width int) string {
	task := m.currentTask()
	if task == nil {
		return helpDescStyle.Render("No task selected")
	}

	t := *task
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
	if maxLen <= 3 {
		return s[:maxLen]
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
	Up       key.Binding
	Down     key.Binding
	Tab      key.Binding
	Escape   key.Binding
	Back     key.Binding
	Delete   key.Binding
	Toggle   key.Binding
	Select   key.Binding
	Run      key.Binding
	ShowLoop key.Binding
	Edit     key.Binding
	New      key.Binding
}

func (k viewKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Toggle, k.Edit, k.New, k.Select, k.Run, k.ShowLoop, k.Delete, k.Tab, k.Escape, k.Back}
}

func (k viewKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Toggle, k.Edit, k.New, k.Select, k.Run},
		{k.ShowLoop, k.Delete, k.Tab, k.Escape, k.Back},
	}
}

var vKeys = viewKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("j/k", "navigate"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/k", "navigate"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "filter"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "clear"),
	),
	Back: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "menu"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Toggle: key.NewBinding(
		key.WithKeys(" ", "enter"),
		key.WithHelp("space", "toggle done"),
	),
	Select: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "select"),
	),
	Run: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "run"),
	),
	ShowLoop: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "show loop"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new task"),
	),
}
