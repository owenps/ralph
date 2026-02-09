package ui

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/jolteon/internal"
)

const quitConfirmWindow = 2 * time.Second

type clearQuitPromptMsg struct{}

type appState int

const (
	stateInit appState = iota
	stateSetup
	stateNoProject
	stateDashboard
	stateCreate
)

type App struct {
	state         appState
	forceInit     bool
	store         *internal.TaskStore
	setupWizard   setupModel
	dashboard     dashboardModel
	createWizard  createModel
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

// NewAppForInit creates a new app for `jolteon init` command
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
		// Handle dismissable warning - any key returns to dashboard
		if m.warnErr != nil {
			m.warnErr = nil
			m.dashboard.SetSize(m.width, m.height)
			m.state = stateDashboard
			return m, nil
		}

		// noProject state quits on 'q'
		if m.state == stateNoProject {
			if key.Matches(msg, appKeys.Quit) {
				return m, tea.Quit
			}
		}

		// ctrl+c requires double-press confirmation in input-heavy states
		if key.Matches(msg, appKeys.QuitCtrlC) {
			if m.state == stateSetup || m.state == stateCreate {
				now := time.Now()
				if !m.lastQuitPress.IsZero() && now.Sub(m.lastQuitPress) < quitConfirmWindow {
					return m, tea.Quit
				}
				m.lastQuitPress = now
				return m, tea.Tick(quitConfirmWindow, func(time.Time) tea.Msg {
					return clearQuitPromptMsg{}
				})
			}
		}

		// Reset quit confirmation on any other key in input-heavy states
		if m.state == stateSetup || m.state == stateCreate {
			m.lastQuitPress = time.Time{}
		}

	case globalConfigFoundMsg:
		if m.forceInit {
			return m, m.initProject
		}
		return m, m.checkProject

	case globalConfigNotFoundMsg:
		if m.forceInit {
			// Auto-create global config and skip setup wizard
			cfg := internal.DefaultGlobalConfig()
			if err := internal.SaveGlobalConfig(cfg); err != nil {
				return m, func() tea.Msg { return errMsg{err} }
			}
			return m, m.initProject
		}
		m.setupWizard = newSetupWizard()
		m.state = stateSetup
		return m, nil

	case setupCompleteMsg:
		if m.forceInit {
			return m, m.initProject
		}
		return m, tea.Quit

	case projectFoundMsg:
		return m, m.loadTasks

	case projectNotFoundMsg:
		m.state = stateNoProject
		return m, nil

	case projectInitializedMsg:
		return m, m.loadTasks

	case tasksLoadedMsg:
		tasksPath, _ := internal.TasksFilePath()
		m.store = internal.NewTaskStore(tasksPath)
		m.store.Load()
		m.dashboard = newDashboard(msg.tasks)
		m.dashboard.SetSize(m.width, m.height)
		m.state = stateDashboard
		return m, m.dashboard.sprite.Tick()

	case errMsg:
		m.err = msg.err
		return m, tea.Quit

	case openCreateMsg:
		m.createWizard = newCreateWizard()
		m.state = stateCreate
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
		m.dashboard.SetTasks(m.store.All())
		m.dashboard.SetSize(m.width, m.height)
		m.state = stateDashboard
		return m, nil

	case cancelCreateMsg:
		m.dashboard.SetSize(m.width, m.height)
		m.state = stateDashboard
		return m, nil

	case deleteTaskMsg:
		m.store.Delete(msg.ID)
		if err := m.store.Save(); err != nil {
			m.err = err
			return m, tea.Quit
		}
		m.dashboard.SetTasks(m.store.All())
		return m, nil

	case editTaskMsg:
		m.createWizard = newCreateWizardWithTask(msg.Task)
		m.state = stateCreate
		return m, nil

	case startSessionMsg:
		return m, m.launchSession(msg.Task)

	case execProcessMsg:
		// Hand terminal to Claude via tea.ExecProcess
		taskID := msg.taskID
		return m, tea.ExecProcess(msg.cmd, func(err error) tea.Msg {
			return sessionDoneMsg{TaskID: taskID}
		})

	case sessionDoneMsg:
		// Refresh tasks after Claude session
		m.store.Load()
		m.dashboard.SetTasks(m.store.All())
		m.dashboard.SetSize(m.width, m.height)
		m.state = stateDashboard
		return m, nil

	case syncIssuesMsg:
		m.dashboard.syncing = true
		m.dashboard.sprite.SetAnim(SpriteWalk)
		return m, tea.Batch(m.dashboard.spinner.Tick, m.doSync)

	case syncCompleteMsg:
		m.dashboard.syncing = false
		m.dashboard.sprite.SetAnim(SpriteIdle)
		if msg.Err != nil {
			m.warnErr = msg.Err
			return m, nil
		}
		m.store.Load()
		m.dashboard.SetTasks(m.store.All())
		m.dashboard.statusMsg = fmt.Sprintf("Synced %d issues", msg.Added)
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case createPRMsg:
		return m, m.doCreatePR(msg.Task)

	case prCreatedMsg:
		if msg.Err != nil {
			m.warnErr = msg.Err
			return m, nil
		}
		m.store.SetPRNumber(msg.TaskID, msg.PRNumber)
		m.store.Save()
		m.dashboard.SetTasks(m.store.All())
		m.dashboard.statusMsg = fmt.Sprintf("PR #%d created", msg.PRNumber)
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case cleanWorktreeMsg:
		return m, m.doCleanWorktree(msg.Task)

	case worktreeCleanedMsg:
		if msg.Err != nil {
			m.warnErr = msg.Err
			return m, nil
		}
		m.store.SetWorktreePath(msg.TaskID, "")
		m.store.SetBranch(msg.TaskID, "")
		m.store.Save()
		m.dashboard.SetTasks(m.store.All())
		m.dashboard.statusMsg = "Worktree cleaned"
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})
	}

	switch m.state {
	case stateSetup:
		return m.updateSetup(msg)
	case stateCreate:
		return m.updateCreate(msg)
	case stateDashboard:
		return m.updateDashboard(msg)
	}

	return m, nil
}

func (m App) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.setupWizard, cmd = m.setupWizard.Update(msg)
	return m, cmd
}

func (m App) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.createWizard, cmd = m.createWizard.Update(msg)
	return m, cmd
}

func (m App) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.dashboard, cmd = m.dashboard.Update(msg)
	return m, cmd
}

func (m App) launchSession(task internal.Task) tea.Cmd {
	return func() tea.Msg {
		// Create worktree if needed
		wtPath, branch, err := internal.WorktreeCreate(&task)
		if err != nil {
			return errMsg{fmt.Errorf("failed to create worktree: %w", err)}
		}

		// Update task in store
		m.store.SetStatus(task.ID, internal.TaskStatusActive)
		m.store.SetBranch(task.ID, branch)
		m.store.SetWorktreePath(task.ID, wtPath)
		m.store.Save()

		// Build Claude command
		resume := task.Status == internal.TaskStatusActive
		cmd, err := internal.BuildClaudeCmd(wtPath, resume)
		if err != nil {
			return errMsg{err}
		}

		// Use tea.ExecProcess to hand terminal to Claude
		return execProcessMsg{cmd: cmd, taskID: task.ID}
	}
}

type execProcessMsg struct {
	cmd    *exec.Cmd
	taskID string
}

func (m App) doSync() tea.Msg {
	added, err := internal.SyncIssues(m.store)
	return syncCompleteMsg{Added: added, Err: err}
}

func (m App) doCreatePR(task internal.Task) tea.Cmd {
	return func() tea.Msg {
		if task.Branch == "" {
			return prCreatedMsg{Err: fmt.Errorf("no branch — start a session first")}
		}

		// Push branch first
		if err := internal.WorktreePush(task.Branch); err != nil {
			return prCreatedMsg{Err: err}
		}

		prNum, err := internal.WorktreeCreatePR(&task, task.Branch)
		return prCreatedMsg{TaskID: task.ID, PRNumber: prNum, Err: err}
	}
}

func (m App) doCleanWorktree(task internal.Task) tea.Cmd {
	return func() tea.Msg {
		if err := internal.WorktreeRemove(task.ID); err != nil {
			return worktreeCleanedMsg{TaskID: task.ID, Err: err}
		}
		return worktreeCleanedMsg{TaskID: task.ID}
	}
}

func (m App) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPlease fix the issue and restart Jolteon.", m.err))
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
	case stateDashboard:
		return m.dashboard.View()
	case stateCreate:
		s := m.createWizard.View()
		if !m.lastQuitPress.IsZero() {
			s += "\n" + warnStyle.Render("Press ctrl+c again to quit")
		}
		return s
	}

	return ""
}

func (m App) viewNoProject() string {
	s := appTitle() + "\n\n"
	s += warnStyle.Render("No Jolteon project found in this directory.") + "\n\n"
	s += helpDescStyle.Render("Run ") + codeStyle.Render("jolteon init") + helpDescStyle.Render(" to start tracking tasks here.") + "\n\n"
	s += renderHelp([]keyBinding{
		{Key: "q", Desc: "quit"},
	})
	return s
}

type appKeyMap struct {
	Quit      key.Binding
	QuitCtrlC key.Binding
}

var appKeys = appKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
	QuitCtrlC: key.NewBinding(
		key.WithKeys("ctrl+c"),
	),
}
