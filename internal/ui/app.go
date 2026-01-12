package ui

import (
	"context"
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
	stateSettings
	stateLoopConfig
	stateLoopRunning
)

type App struct {
	state         appState
	forceInit     bool
	store         *internal.TaskStore
	globalConfig  *internal.GlobalConfig
	setupWizard   setupModel
	menuView      menuModel
	createWizard  createModel
	viewTasks     viewModel
	settingsView  settingsModel
	loopConfig    loopConfigModel
	loopView      loopViewModel
	loopRunner    *internal.LoopRunner
	lastQuitPress time.Time
	err           error
	warnErr       error
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
	return App{
		state:     stateInit,
		forceInit: forceInit,
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
type backToMenuMsg struct{}

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
		// Handle dismissable warning - any key returns to manage tasks
		if m.warnErr != nil {
			m.warnErr = nil
			m.viewTasks.SetSize(m.width, m.height)
			m.state = stateView
			return m, nil
		}

		// Only menu and noProject states quit on 'q'
		// All other states handle 'q' themselves (typically as "back")
		if m.state == stateMenu || m.state == stateNoProject {
			if key.Matches(msg, appKeys.Quit) {
				return m, tea.Quit
			}
		}

		// ctrl+c requires double-press confirmation in input-heavy states
		if key.Matches(msg, appKeys.QuitCtrlC) {
			if m.state == stateSetup || m.state == stateCreate || m.state == stateSettings || m.state == stateLoopConfig || m.state == stateLoopRunning {
				now := time.Now()
				if !m.lastQuitPress.IsZero() && now.Sub(m.lastQuitPress) < quitConfirmWindow {
					return m, tea.Quit
				}
				m.lastQuitPress = now
				return m, tea.Tick(quitConfirmWindow, func(time.Time) tea.Msg {
					return clearQuitPromptMsg{}
				})
			}
			// Other states quit immediately on ctrl+c
			return m, tea.Quit
		}

		// Reset quit confirmation on any other key in input-heavy states
		if m.state == stateSetup || m.state == stateCreate || m.state == stateSettings || m.state == stateLoopConfig || m.state == stateLoopRunning {
			m.lastQuitPress = time.Time{}
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
		m.viewTasks.SetSize(m.width, m.height)
		// Load global config
		cfg, err := internal.LoadGlobalConfig()
		if err != nil {
			cfg = internal.DefaultGlobalConfig()
		}
		m.globalConfig = cfg
		m.menuView = newMenuModel()
		m.state = stateMenu
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case openTasksFromMenuMsg:
		m.viewTasks.SetSize(m.width, m.height)
		m.state = stateView
		return m, nil

	case openSettingsFromMenuMsg:
		m.settingsView = newSettingsView(m.globalConfig)
		m.state = stateSettings
		return m, nil

	case backToMenuMsg:
		m.state = stateMenu
		return m, nil

	case openCreateMsg:
		m.createWizard = newCreateWizard()
		m.state = stateCreate
		return m, nil

	case openSettingsMsg:
		m.settingsView = newSettingsView(m.globalConfig)
		m.state = stateSettings
		return m, nil

	case saveTaskMsg:
		if msg.EditingID != "" {
			m.store.Update(msg.EditingID, msg.Task)
		} else {
			m.store.Add(msg.Task)
		}
		if err := m.store.Save(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.viewTasks.SetTasks(m.store.All())
		m.viewTasks.SetSize(m.width, m.height)
		m.state = stateView
		return m, nil

	case cancelCreateMsg:
		m.viewTasks.SetSize(m.width, m.height)
		m.state = stateView
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

	case editTaskMsg:
		m.createWizard = newCreateWizardWithTask(msg.Task)
		m.state = stateCreate
		return m, nil

	case backFromSettingsMsg:
		m.state = stateMenu
		return m, nil

	case settingsSavedMsg:
		// Settings saved, stay in settings view
		return m, nil

	case runSelectedTasksMsg:
		// Check for uncommitted changes
		if err := internal.CheckWorkingDirectoryClean(); err != nil {
			m.warnErr = err
			return m, nil
		}
		// Go to loop config
		m.loopConfig = newLoopConfig(msg.TaskIDs, m.store.All(), m.globalConfig.Loop.DefaultMaxIterations)
		m.state = stateLoopConfig
		return m, nil

	case cancelLoopConfigMsg:
		m.viewTasks.SetSize(m.width, m.height)
		m.state = stateView
		return m, nil

	case startLoopMsg:
		runner, err := internal.NewLoopRunner(m.globalConfig, m.store.All(), msg.TaskIDs, m.store, msg.MaxIterations)
		if err != nil {
			m.warnErr = err
			return m, nil
		}
		m.loopRunner = runner
		m.loopView = newLoopView(runner)
		m.state = stateLoopRunning

		// Start the loop in background and init the loop view components
		return m, tea.Batch(m.startLoop, m.loopView.Init())

	case loopOutputMsg:
		m.loopView, _ = m.loopView.Update(msg)
		return m, nil

	case loopCompleteMsg:
		// Refresh tasks
		m.store.Load()
		m.viewTasks.SetTasks(m.store.All())
		// Clear selections
		m.viewTasks.selected = make(map[string]bool)
		return m, nil

	case hideLoopViewMsg:
		m.viewTasks.SetSize(m.width, m.height)
		m.state = stateView
		return m, nil

	case showLoopMsg:
		if m.loopRunner != nil {
			state := m.loopRunner.GetState()
			if state.Status == internal.RunStatusRunning {
				m.state = stateLoopRunning
			}
		}
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
	case stateSettings:
		return m.updateSettings(msg)
	case stateLoopConfig:
		return m.updateLoopConfig(msg)
	case stateLoopRunning:
		return m.updateLoopView(msg)
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
	m.menuView, cmd = m.menuView.Update(msg)
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

func (m App) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.settingsView, cmd = m.settingsView.Update(msg)
	return m, cmd
}

func (m App) updateLoopConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.loopConfig, cmd = m.loopConfig.Update(msg)
	return m, cmd
}

func (m App) updateLoopView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.loopView, cmd = m.loopView.Update(msg)
	return m, cmd
}

func (m App) startLoop() tea.Msg {
	outputChan := make(chan string, 100)

	go func() {
		ctx := context.Background()
		err := m.loopRunner.Run(ctx, outputChan)
		close(outputChan)
		if err != nil {
			// Error is already logged in progress
		}
	}()

	// Start reading output in another goroutine
	go func() {
		for output := range outputChan {
			// We can't send messages from here directly
			// The loop view polls state instead
			_ = output
		}
	}()

	return loopCompleteMsg{}
}

func (m App) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPlease fix the issue and restart Ralph.", m.err))
	}

	if m.warnErr != nil {
		content := errorStyle.Render(fmt.Sprintf("Error: %v", m.warnErr)) + "\n\n" + helpDescStyle.Render("Press any key to continue")
		return warnBorderStyle.Render(content)
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
		return m.menuView.View()
	case stateCreate:
		s := m.createWizard.View()
		if !m.lastQuitPress.IsZero() {
			s += "\n" + warnStyle.Render("Press ctrl+c again to quit")
		}
		return s
	case stateView:
		return m.viewTasks.View()
	case stateSettings:
		return m.settingsView.View()
	case stateLoopConfig:
		return m.loopConfig.View()
	case stateLoopRunning:
		return m.loopView.View()
	}

	return ""
}

func (m App) viewNoProject() string {
	s := appTitle() + "\n\n"
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
