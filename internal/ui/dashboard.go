package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenps/jolteon/internal"
)

const deleteConfirmWindow = 2 * time.Second

type statusFilter int

const (
	filterAll statusFilter = iota
	filterPending
	filterActive
	filterDone
)

// taskItem wraps internal.Task to implement list.Item
type taskItem struct {
	task internal.Task
}

func (i taskItem) Title() string       { return i.task.Description }
func (i taskItem) Description() string { return string(i.task.Category) }
func (i taskItem) FilterValue() string { return i.task.Description }

// taskDelegate renders tasks with status badges and category badges.
type taskDelegate struct{}

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

	sBadge := statusBadge(t.Status)
	statusStr := string(t.Status)
	if len(statusStr) > 3 {
		statusStr = statusStr[:3]
	}

	cBadge := categoryBadge(string(t.Category))
	catStr := string(t.Category)
	if len(catStr) > 3 {
		catStr = catStr[:3]
	}

	width := m.Width() - 4
	if width < 20 {
		width = 20
	}

	// Show GitHub issue number if from GitHub
	ghTag := ""
	if t.Source == internal.TaskSourceGitHub && t.IssueNumber > 0 {
		ghTag = helpDescStyle.Render(fmt.Sprintf(" #%d", t.IssueNumber))
	}

	desc := truncate(t.Description, width-20)
	line := cursor + sBadge.Render(statusStr) + " " + cBadge.Render(catStr) + " " + style.Render(desc) + ghTag
	fmt.Fprint(w, line)
}

type dashboardModel struct {
	tasks           []internal.Task
	filter          statusFilter
	width           int
	height          int
	lastDeletePress time.Time
	deleteConfirmID string
	help            help.Model
	keys            dashKeyMap
	list            list.Model
	sprite          SpriteModel
	spinner         spinner.Model
	syncing         bool
	statusMsg       string
	statusTimer     time.Time
}

type deleteTaskMsg struct{ ID string }
type clearDeletePromptMsg struct{}
type clearStatusMsg struct{}
type openCreateMsg struct{}
type syncIssuesMsg struct{}
type syncCompleteMsg struct {
	Added int
	Err   error
}
type startSessionMsg struct{ Task internal.Task }
type sessionDoneMsg struct{ TaskID string }
type createPRMsg struct{ Task internal.Task }
type prCreatedMsg struct {
	TaskID   string
	PRNumber int
	Err      error
}
type cleanWorktreeMsg struct{ Task internal.Task }
type worktreeCleanedMsg struct {
	TaskID string
	Err    error
}

func newDashboard(tasks []internal.Task) dashboardModel {
	h := help.New()
	h.ShowAll = true
	h.Styles.ShortKey = helpKeyStyle
	h.Styles.ShortDesc = helpDescStyle
	h.Styles.ShortSeparator = helpDescStyle
	h.Styles.FullKey = helpKeyStyle
	h.Styles.FullDesc = helpDescStyle
	h.Styles.FullSeparator = helpDescStyle

	delegate := taskDelegate{}
	items := tasksToItems(tasks)
	l := list.New(items, delegate, 40, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorYellow)

	m := dashboardModel{
		tasks:   tasks,
		filter:  filterAll,
		help:    h,
		keys:    dKeys,
		list:    l,
		sprite:  NewSprite(),
		spinner: s,
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

func (m *dashboardModel) SetTasks(tasks []internal.Task) {
	m.tasks = tasks
	m.applyFilter()
}

func (m *dashboardModel) SetSize(width, height int) {
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

func (m *dashboardModel) applyFilter() {
	var filtered []internal.Task
	switch m.filter {
	case filterAll:
		filtered = m.tasks
	case filterPending:
		filtered = filterByStatus(m.tasks, internal.TaskStatusPending)
	case filterActive:
		filtered = filterByStatus(m.tasks, internal.TaskStatusActive)
	case filterDone:
		filtered = filterByStatus(m.tasks, internal.TaskStatusDone)
	}
	m.list.SetItems(tasksToItems(filtered))
}

func (m *dashboardModel) currentTask() *internal.Task {
	if item := m.list.SelectedItem(); item != nil {
		if ti, ok := item.(taskItem); ok {
			return &ti.task
		}
	}
	return nil
}

func filterByStatus(tasks []internal.Task, status internal.TaskStatus) []internal.Task {
	var result []internal.Task
	for _, t := range tasks {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

func (m dashboardModel) Update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case clearDeletePromptMsg:
		m.lastDeletePress = time.Time{}
		m.deleteConfirmID = ""
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case spriteTickMsg:
		var cmd tea.Cmd
		m.sprite, cmd = m.sprite.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		if m.syncing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, dKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, dKeys.Escape):
			if !m.lastDeletePress.IsZero() {
				m.lastDeletePress = time.Time{}
				m.deleteConfirmID = ""
				return m, nil
			}
			return m, nil
		case key.Matches(msg, dKeys.New):
			return m, func() tea.Msg { return openCreateMsg{} }
		case key.Matches(msg, dKeys.Tab):
			m.filter = (m.filter + 1) % 4
			m.applyFilter()
			m.lastDeletePress = time.Time{}
			m.deleteConfirmID = ""
		case key.Matches(msg, dKeys.Enter):
			if task := m.currentTask(); task != nil {
				t := *task
				return m, func() tea.Msg { return startSessionMsg{Task: t} }
			}
		case key.Matches(msg, dKeys.CreatePR):
			if task := m.currentTask(); task != nil {
				t := *task
				return m, func() tea.Msg { return createPRMsg{Task: t} }
			}
		case key.Matches(msg, dKeys.CleanWorktree):
			if task := m.currentTask(); task != nil {
				t := *task
				return m, func() tea.Msg { return cleanWorktreeMsg{Task: t} }
			}
		case key.Matches(msg, dKeys.Sync):
			if !m.syncing {
				return m, func() tea.Msg { return syncIssuesMsg{} }
			}
		case key.Matches(msg, dKeys.Delete):
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
		case key.Matches(msg, dKeys.Edit):
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
		if key.Matches(msg.(tea.KeyMsg), dKeys.Up) || key.Matches(msg.(tea.KeyMsg), dKeys.Down) {
			m.lastDeletePress = time.Time{}
			m.deleteConfirmID = ""
		}
	}

	return m, cmd
}

func (m dashboardModel) renderHeader() string {
	titleBox := appTitle()
	spriteView := m.sprite.View()
	if m.width >= 80 && spriteView != "" {
		return lipgloss.JoinHorizontal(lipgloss.Bottom, spriteView, "  ", titleBox)
	}
	return titleBox
}

func (m dashboardModel) View() string {
	if len(m.list.Items()) == 0 && !m.syncing {
		s := m.renderHeader() + "\n\n"
		s += m.renderTabs() + "\n\n"
		s += helpDescStyle.Render("No tasks yet.") + "\n\n"
		s += renderHelp([]keyBinding{
			{Key: "n", Desc: "new task"},
			{Key: "S", Desc: "sync issues"},
			{Key: "q", Desc: "quit"},
		})
		return s
	}

	s := m.renderHeader() + "\n\n"
	s += m.renderTabs()

	// Inline warnings
	if !m.lastDeletePress.IsZero() {
		s += "  " + warnStyle.Render("Press d again to delete")
	}
	if m.syncing {
		s += "  " + m.spinner.View() + " " + helpDescStyle.Render("Syncing issues...")
	}
	if m.statusMsg != "" {
		s += "  " + successStyle.Render(m.statusMsg)
	}

	s += "\n\n"
	s += m.renderSplitView()

	s += "\n"
	s += m.help.View(m.keys)

	return s
}

func (m dashboardModel) renderTabs() string {
	tabs := []struct {
		label  string
		filter statusFilter
	}{
		{"all", filterAll},
		{"pending", filterPending},
		{"active", filterActive},
		{"done", filterDone},
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

func (m dashboardModel) renderSplitView() string {
	totalWidth := m.width
	if totalWidth < 60 {
		totalWidth = 80
	}
	listWidth := totalWidth * 2 / 5
	detailWidth := totalWidth - listWidth - 5

	listPanel := borderStyle.
		Width(listWidth).
		Render(m.list.View())

	detailContent := m.renderTaskDetail(detailWidth)
	detailPanel := borderStyle.
		Width(detailWidth).
		Render(detailContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)
}

func (m dashboardModel) renderTaskDetail(width int) string {
	task := m.currentTask()
	if task == nil {
		return helpDescStyle.Render("No task selected")
	}

	t := *task
	var s strings.Builder

	// Status badge
	sBadge := statusBadge(t.Status)
	s.WriteString(detailLabelStyle.Render("Status: "))
	s.WriteString(sBadge.Render(string(t.Status)))
	s.WriteString("\n\n")

	// Category badge
	badge := categoryBadge(string(t.Category))
	s.WriteString(detailLabelStyle.Render("Category: "))
	s.WriteString(badge.Render(string(t.Category)))
	s.WriteString("\n\n")

	// Description
	s.WriteString(detailLabelStyle.Render("Description:"))
	s.WriteString("\n")
	s.WriteString(detailValueStyle.Render(wrapText(t.Description, width-4)))
	s.WriteString("\n\n")

	// Steps
	if len(t.Steps) > 0 {
		s.WriteString(detailLabelStyle.Render("Steps:"))
		s.WriteString("\n")
		for i, step := range t.Steps {
			s.WriteString(detailValueStyle.Render(fmt.Sprintf("  %d. %s", i+1, step)))
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Source info
	if t.Source == internal.TaskSourceGitHub && t.IssueNumber > 0 {
		s.WriteString(detailLabelStyle.Render("Issue: "))
		s.WriteString(detailValueStyle.Render(fmt.Sprintf("#%d", t.IssueNumber)))
		s.WriteString("\n")
	}

	// Branch info
	if t.Branch != "" {
		s.WriteString(detailLabelStyle.Render("Branch: "))
		s.WriteString(detailValueStyle.Render(t.Branch))
		s.WriteString("\n")
	}

	// Worktree path
	if t.WorktreePath != "" {
		s.WriteString(detailLabelStyle.Render("Worktree: "))
		s.WriteString(helpDescStyle.Render(t.WorktreePath))
		s.WriteString("\n")
	}

	// PR number
	if t.PRNumber > 0 {
		s.WriteString(detailLabelStyle.Render("PR: "))
		s.WriteString(detailValueStyle.Render(fmt.Sprintf("#%d", t.PRNumber)))
		s.WriteString("\n")
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

// Key map for dashboard
type dashKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Tab           key.Binding
	Escape        key.Binding
	Quit          key.Binding
	Enter         key.Binding
	Delete        key.Binding
	Edit          key.Binding
	New           key.Binding
	CreatePR      key.Binding
	CleanWorktree key.Binding
	Sync          key.Binding
}

func (k dashKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Enter, k.New, k.CreatePR, k.CleanWorktree, k.Sync, k.Edit, k.Delete, k.Tab, k.Quit}
}

func (k dashKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Enter, k.New, k.CreatePR, k.CleanWorktree},
		{k.Sync, k.Edit, k.Delete, k.Tab, k.Quit},
	}
}

var dKeys = dashKeyMap{
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
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "start session"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new task"),
	),
	CreatePR: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "create PR"),
	),
	CleanWorktree: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "clean worktree"),
	),
	Sync: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "sync issues"),
	),
}

type editTaskMsg struct{ Task internal.Task }
