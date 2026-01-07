package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/ralph/internal"
)

const quitConfirmWindow = 2 * time.Second

type clearQuitPromptMsg struct{}

type appState int

const (
	stateInit appState = iota
	stateSetup
	stateNoProject
	stateMenu
	stateCreate
	stateView
)

type App struct {
	state         appState
	forceInit     bool
	store         *internal.TaskStore
	setupWizard   setupModel
	menu          menuModel
	createWizard  createModel
	viewTasks     viewModel
	lastQuitPress time.Time
	err           error
	width         int
	height        int
}

// NewApp creates a new app for normal launch
func NewApp() App {
	return newAppWithOptions(false)
}

// NewAppForInit creates a new app for `ralph init` command
func NewAppForInit() App {
	return newAppWithOptions(true)
}

func newAppWithOptions(forceInit bool) App {
	menuItems := []menuItem{
		{Name: "Create Task", Description: "Create a new backlog task"},
		{Name: "View Tasks", Description: "View and filter existing tasks"},
	}

	return App{
		state:     stateInit,
		forceInit: forceInit,
		menu:      newMenu(menuItems),
	}
}

func (m App) Init() tea.Cmd {
	return m.checkGlobalConfig
}

// Message types
type globalConfigFoundMsg struct{}
type globalConfigNotFoundMsg struct{}
type projectFoundMsg struct{}
type projectNotFoundMsg struct{}
type projectInitializedMsg struct{}
type tasksLoadedMsg struct{ tasks []internal.Task }
type errMsg struct{ err error }

func (m App) checkGlobalConfig() tea.Msg {
	if internal.GlobalConfigExists() {
		return globalConfigFoundMsg{}
	}
	return globalConfigNotFoundMsg{}
}

func (m App) checkProject() tea.Msg {
	if internal.ProjectExists() {
		return projectFoundMsg{}
	}
	return projectNotFoundMsg{}
}

func (m App) initProject() tea.Msg {
	if err := internal.InitProject(); err != nil {
		return errMsg{err}
	}
	return projectInitializedMsg{}
}

func (m App) loadTasks() tea.Msg {
	tasksPath, err := internal.TasksFilePath()
	if err != nil {
		return errMsg{err}
	}

	store := internal.NewTaskStore(tasksPath)
	if err := store.Load(); err != nil {
		return errMsg{fmt.Errorf("failed to parse tasks file: %w", err)}
	}

	return tasksLoadedMsg{tasks: store.All()}
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case clearQuitPromptMsg:
		m.lastQuitPress = time.Time{}
		return m, nil

	case tea.KeyMsg:
		// States with text input only respond to ctrl+c (not 'q')
		if m.state == stateSetup || m.state == stateCreate {
			if key.Matches(msg, appKeys.QuitCtrlC) {
				now := time.Now()
				if !m.lastQuitPress.IsZero() && now.Sub(m.lastQuitPress) < quitConfirmWindow {
					return m, tea.Quit
				}
				m.lastQuitPress = now
				return m, tea.Tick(quitConfirmWindow, func(time.Time) tea.Msg {
					return clearQuitPromptMsg{}
				})
			}
			// Reset quit confirmation on any other key
			m.lastQuitPress = time.Time{}
		} else if key.Matches(msg, appKeys.Quit) {
			return m, tea.Quit
		}

	case globalConfigFoundMsg:
		// Global config exists
		if m.forceInit {
			// ralph init: initialize project
			return m, m.initProject
		}
		// Normal launch: check for project
		return m, m.checkProject

	case globalConfigNotFoundMsg:
		// No global config - show setup wizard
		m.setupWizard = newSetupWizard()
		m.state = stateSetup
		return m, nil

	case setupCompleteMsg:
		// Setup finished
		if m.forceInit {
			// ralph init: initialize project
			return m, m.initProject
		}
		// Normal launch after first-time setup: exit with instructions
		return m, tea.Quit

	case projectFoundMsg:
		// Project exists - load tasks
		return m, m.loadTasks

	case projectNotFoundMsg:
		// No project in current directory
		m.state = stateNoProject
		return m, nil

	case projectInitializedMsg:
		// Project just initialized - load tasks
		return m, m.loadTasks

	case tasksLoadedMsg:
		tasksPath, _ := internal.TasksFilePath()
		m.store = internal.NewTaskStore(tasksPath)
		m.store.Load()
		m.viewTasks = newViewTasks(msg.tasks)
		m.state = stateMenu
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case menuSelectMsg:
		switch msg.Index {
		case 0:
			m.createWizard = newCreateWizard()
			m.state = stateCreate
			return m, nil
		case 1:
			m.viewTasks.SetTasks(m.store.All())
			m.state = stateView
			return m, nil
		}

	case saveTaskMsg:
		m.store.Add(msg.Task)
		if err := m.store.Save(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.state = stateMenu
		return m, nil

	case cancelCreateMsg:
		m.state = stateMenu
		return m, nil

	case backToMenuMsg:
		m.state = stateMenu
		return m, nil

	case deleteTaskMsg:
		m.store.Delete(msg.ID)
		if err := m.store.Save(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.viewTasks.SetTasks(m.store.All())
		return m, nil

	case toggleTaskMsg:
		m.store.ToggleDone(msg.ID)
		if err := m.store.Save(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.viewTasks.SetTasks(m.store.All())
		return m, nil
	}

	switch m.state {
	case stateSetup:
		return m.updateSetup(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateCreate:
		return m.updateCreate(msg)
	case stateView:
		return m.updateView(msg)
	}

	return m, nil
}

func (m App) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.setupWizard, cmd = m.setupWizard.Update(msg)
	return m, cmd
}

func (m App) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m App) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.createWizard, cmd = m.createWizard.Update(msg)
	return m, cmd
}

func (m App) updateView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewTasks, cmd = m.viewTasks.Update(msg)
	return m, cmd
}

func (m App) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPlease fix the issue and restart Ralph.", m.err))
	}

	switch m.state {
	case stateInit:
		return "Loading..."
	case stateSetup:
		s := m.setupWizard.View()
		if !m.lastQuitPress.IsZero() {
			s += "\n" + warnStyle.Render("Press ctrl+c again to quit")
		}
		return s
	case stateNoProject:
		return m.viewNoProject()
	case stateMenu:
		return m.menu.View()
	case stateCreate:
		s := m.createWizard.View()
		if !m.lastQuitPress.IsZero() {
			s += "\n" + warnStyle.Render("Press ctrl+c again to quit")
		}
		return s
	case stateView:
		return m.viewTasks.View()
	}

	return ""
}

func (m App) viewNoProject() string {
	s := banner() + "\n\n"
	s += warnStyle.Render("No Ralph project found in this directory.") + "\n\n"
	s += helpDescStyle.Render("Run ") + codeStyle.Render("ralph init") + helpDescStyle.Render(" to start tracking tasks here.") + "\n\n"
	s += renderHelp([]keyBinding{
		{Key: "q", Desc: "quit"},
	})
	return s
}

type appKeyMap struct {
	Quit      key.Binding
	QuitCtrlC key.Binding
	Enter     key.Binding
	Left      key.Binding
	Right     key.Binding
}

var appKeys = appKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
	QuitCtrlC: key.NewBinding(
		key.WithKeys("ctrl+c"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
	),
}
