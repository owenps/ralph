package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/ralph/internal"
)

type loopViewModel struct {
	runner    *internal.LoopRunner
	state     *internal.LoopState
	output    []string
	startTime time.Time
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
	return loopViewModel{
		runner:    runner,
		state:     runner.GetState(),
		output:    []string{},
		startTime: time.Now(),
	}
}

func (m loopViewModel) Update(msg tea.Msg) (loopViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case loopOutputMsg:
		m.output = append(m.output, msg.Output)
		// Keep only last 20 lines
		if len(m.output) > 20 {
			m.output = m.output[len(m.output)-20:]
		}
		m.state = m.runner.GetState()
		return m, nil

	case loopTickMsg:
		m.state = m.runner.GetState()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, loopKeys.GracefulStop):
			m.runner.RequestGracefulStop()
			m.output = append(m.output, "Graceful stop requested - finishing current task...")
		case key.Matches(msg, loopKeys.ImmediateAbort):
			m.runner.RequestImmediateAbort()
			m.output = append(m.output, "Immediate abort requested...")
		case key.Matches(msg, loopKeys.Escape):
			return m, func() tea.Msg { return hideLoopViewMsg{} }
		}
	}

	return m, nil
}

func (m loopViewModel) View() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Ralph Loop"))

	switch m.state.Status {
	case internal.RunStatusRunning:
		s.WriteString(successStyle.Render(" Running"))
	case internal.RunStatusCompleted:
		s.WriteString(successStyle.Render(" Completed"))
	case internal.RunStatusFailed:
		s.WriteString(errorStyle.Render(" Failed"))
	case internal.RunStatusAborted:
		s.WriteString(warnStyle.Render(" Aborted"))
	}

	s.WriteString("\n\n")

	// Task progress
	s.WriteString(fmt.Sprintf("Task %d/%d", m.state.CurrentTaskIdx+1, m.state.TotalTasks))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("Iteration: %d/%d", m.state.Iteration, m.state.MaxIterations))
	s.WriteString("\n")

	elapsed := time.Since(m.startTime).Round(time.Second)
	s.WriteString(fmt.Sprintf("Elapsed: %s", elapsed))
	s.WriteString("\n\n")

	// Live output
	s.WriteString(detailLabelStyle.Render("Live Output"))
	s.WriteString("\n")
	s.WriteString("─────────────────────────────────\n")

	outputHeight := 10
	outputLines := m.output
	if len(outputLines) > outputHeight {
		outputLines = outputLines[len(outputLines)-outputHeight:]
	}

	for _, line := range outputLines {
		// Truncate long lines
		if len(line) > 60 {
			line = line[:57] + "..."
		}
		s.WriteString(helpDescStyle.Render(line))
		s.WriteString("\n")
	}

	// Pad to fixed height
	for i := len(outputLines); i < outputHeight; i++ {
		s.WriteString("\n")
	}

	s.WriteString("─────────────────────────────────\n\n")

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
	Escape         key.Binding
}

var loopKeys = loopKeyMap{
	GracefulStop: key.NewBinding(
		key.WithKeys("c"),
	),
	ImmediateAbort: key.NewBinding(
		key.WithKeys("C"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
}

var loopHelpKeys = []keyBinding{
	{Key: "c", Desc: "graceful stop"},
	{Key: "C", Desc: "abort"},
	{Key: "esc", Desc: "hide"},
}

var loopDoneHelpKeys = []keyBinding{
	{Key: "esc", Desc: "back"},
}
