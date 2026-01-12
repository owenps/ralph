package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenps/ralph/internal"
)

type loopViewModel struct {
	runner    *internal.LoopRunner
	state     *internal.LoopState
	spinner   spinner.Model
	progress  progress.Model
	viewport  viewport.Model
	stopwatch stopwatch.Model
	output    []string
	width     int
	height    int
}

type loopOutputMsg struct {
	Output string
}

type loopCompleteMsg struct {
	Status internal.RunStatus
	Error  error
}

type loopTickMsg struct{}

type hideLoopViewMsg struct{}

func newLoopView(runner *internal.LoopRunner) loopViewModel {
	// Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorYellow)

	// Progress bar
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	// Viewport for output
	vp := viewport.New(60, 10)
	vp.Style = lipgloss.NewStyle().Foreground(colorGray)

	// Stopwatch
	sw := stopwatch.New()

	return loopViewModel{
		runner:    runner,
		state:     runner.GetState(),
		spinner:   s,
		progress:  p,
		viewport:  vp,
		stopwatch: sw,
		output:    []string{},
	}
}

func (m loopViewModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.stopwatch.Start())
}

func (m loopViewModel) Update(msg tea.Msg) (loopViewModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.progress.Width = msg.Width - 20
		return m, nil

	case loopOutputMsg:
		m.output = append(m.output, msg.Output)
		m.viewport.SetContent(strings.Join(m.output, "\n"))
		m.viewport.GotoBottom()
		m.state = m.runner.GetState()
		return m, nil

	case loopTickMsg:
		m.state = m.runner.GetState()
		return m, nil

	case loopCompleteMsg:
		// Stop the stopwatch when complete
		return m, m.stopwatch.Stop()

	case tea.KeyMsg:
		if m.state.Status == internal.RunStatusRunning {
			switch {
			case key.Matches(msg, loopKeys.GracefulStop):
				m.runner.RequestGracefulStop()
				m.output = append(m.output, "Stopping after current iteration...")
				m.viewport.SetContent(strings.Join(m.output, "\n"))
				m.viewport.GotoBottom()
			case key.Matches(msg, loopKeys.ImmediateAbort):
				m.runner.RequestImmediateAbort()
				m.output = append(m.output, "Stopping now...")
				m.viewport.SetContent(strings.Join(m.output, "\n"))
				m.viewport.GotoBottom()
			case key.Matches(msg, loopKeys.Hide):
				return m, func() tea.Msg { return hideLoopViewMsg{} }
			}
		} else {
			// Loop is done - use q to go back
			if key.Matches(msg, loopKeys.Back) {
				return m, func() tea.Msg { return hideLoopViewMsg{} }
			}
		}
	}

	// Update spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	// Update stopwatch
	m.stopwatch, cmd = m.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport (for scrolling)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m loopViewModel) View() string {
	var s strings.Builder

	// Header with spinner or status
	s.WriteString(headerStyle.Render("Ralph Loop"))
	s.WriteString(" ")

	switch m.state.Status {
	case internal.RunStatusRunning:
		s.WriteString(m.spinner.View())
		s.WriteString(" ")
		s.WriteString(successStyle.Render("Running"))
	case internal.RunStatusCompleted:
		s.WriteString(successStyle.Render("Completed"))
	case internal.RunStatusFailed:
		s.WriteString(errorStyle.Render("Failed"))
	case internal.RunStatusAborted:
		s.WriteString(warnStyle.Render("Stopped"))
	}

	s.WriteString("\n\n")

	// Elapsed time using stopwatch
	s.WriteString(detailLabelStyle.Render("Elapsed: "))
	s.WriteString(m.stopwatch.View())
	s.WriteString("\n\n")

	// Task progress with progress bar
	taskProgress := float64(m.state.CurrentTaskIdx) / float64(m.state.TotalTasks)
	s.WriteString(detailLabelStyle.Render("Tasks: "))
	s.WriteString(fmt.Sprintf("%d/%d ", m.state.CurrentTaskIdx+1, m.state.TotalTasks))
	s.WriteString(m.progress.ViewAs(taskProgress))
	s.WriteString("\n")

	// Iteration progress with progress bar
	iterProgress := float64(m.state.Iteration) / float64(m.state.MaxIterations)
	s.WriteString(detailLabelStyle.Render("Iteration: "))
	s.WriteString(fmt.Sprintf("%d/%d ", m.state.Iteration, m.state.MaxIterations))
	s.WriteString(m.progress.ViewAs(iterProgress))
	s.WriteString("\n\n")

	// Live output using viewport
	s.WriteString(detailLabelStyle.Render("Live Output"))
	s.WriteString("\n")
	s.WriteString(borderStyle.Render(m.viewport.View()))
	s.WriteString("\n\n")

	// Help
	if m.state.Status == internal.RunStatusRunning {
		s.WriteString(renderHelp(loopHelpKeys))
	} else {
		s.WriteString(renderHelp(loopDoneHelpKeys))
	}

	return s.String()
}

type loopKeyMap struct {
	GracefulStop   key.Binding
	ImmediateAbort key.Binding
	Hide           key.Binding
	Back           key.Binding
}

var loopKeys = loopKeyMap{
	GracefulStop: key.NewBinding(
		key.WithKeys("c"),
	),
	ImmediateAbort: key.NewBinding(
		key.WithKeys("C"),
	),
	Hide: key.NewBinding(
		key.WithKeys("esc"),
	),
	Back: key.NewBinding(
		key.WithKeys("q"),
	),
}

var loopHelpKeys = []keyBinding{
	{Key: "c", Desc: "stop after iteration"},
	{Key: "C", Desc: "stop now"},
	{Key: "esc", Desc: "hide"},
}

var loopDoneHelpKeys = []keyBinding{
	{Key: "q", Desc: "tasks"},
}
