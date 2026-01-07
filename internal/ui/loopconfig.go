package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/ralph/internal"
)

type loopConfigStep int

const (
	loopConfigIterations loopConfigStep = iota
	loopConfigPreview
)

type loopConfigModel struct {
	step          loopConfigStep
	taskIDs       []string
	tasks         []internal.Task
	defaultMax    int
	iterInput     textinput.Model
	maxIterations int
	width         int
	height        int
}

type startLoopMsg struct {
	TaskIDs       []string
	MaxIterations int
}

type cancelLoopConfigMsg struct{}

func newLoopConfig(taskIDs []string, tasks []internal.Task, defaultMax int) loopConfigModel {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("default: %d", defaultMax)
	ti.CharLimit = 5
	ti.Width = 20
	ti.Focus()

	// Build task list for preview
	taskMap := make(map[string]internal.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	var selectedTasks []internal.Task
	for _, id := range taskIDs {
		if t, ok := taskMap[id]; ok {
			selectedTasks = append(selectedTasks, t)
		}
	}

	return loopConfigModel{
		step:       loopConfigIterations,
		taskIDs:    taskIDs,
		tasks:      selectedTasks,
		defaultMax: defaultMax,
		iterInput:  ti,
	}
}

func (m loopConfigModel) Update(msg tea.Msg) (loopConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case loopConfigIterations:
			return m.updateIterations(msg)
		case loopConfigPreview:
			return m.updatePreview(msg)
		}
	}

	return m, nil
}

func (m loopConfigModel) updateIterations(msg tea.KeyMsg) (loopConfigModel, tea.Cmd) {
	switch {
	case key.Matches(msg, loopConfigKeys.Escape):
		return m, func() tea.Msg { return cancelLoopConfigMsg{} }
	case key.Matches(msg, loopConfigKeys.Enter):
		value := strings.TrimSpace(m.iterInput.Value())
		if value == "" {
			m.maxIterations = m.defaultMax
		} else if v, err := strconv.Atoi(value); err == nil && v > 0 {
			m.maxIterations = v
		} else {
			// Invalid input, use default
			m.maxIterations = m.defaultMax
		}
		m.step = loopConfigPreview
		m.iterInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.iterInput, cmd = m.iterInput.Update(msg)
	return m, cmd
}

func (m loopConfigModel) updatePreview(msg tea.KeyMsg) (loopConfigModel, tea.Cmd) {
	switch {
	case key.Matches(msg, loopConfigKeys.Escape):
		m.step = loopConfigIterations
		m.iterInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, loopConfigKeys.Enter):
		return m, func() tea.Msg {
			return startLoopMsg{
				TaskIDs:       m.taskIDs,
				MaxIterations: m.maxIterations,
			}
		}
	}

	return m, nil
}

func (m loopConfigModel) View() string {
	switch m.step {
	case loopConfigIterations:
		return m.viewIterations()
	case loopConfigPreview:
		return m.viewPreview()
	}
	return ""
}

func (m loopConfigModel) viewIterations() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Run Loop"))
	s.WriteString("\n\n")

	s.WriteString(fmt.Sprintf("Selected %d tasks\n\n", len(m.tasks)))

	s.WriteString(inputLabelStyle.Render("Max Iterations:"))
	s.WriteString("\n")
	s.WriteString(m.iterInput.View())
	s.WriteString("\n\n")

	s.WriteString(helpDescStyle.Render("Leave blank for default"))
	s.WriteString("\n\n")

	s.WriteString(renderHelp(inputKeys))

	return s.String()
}

func (m loopConfigModel) viewPreview() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("Confirm Loop"))
	s.WriteString("\n\n")

	s.WriteString(detailLabelStyle.Render(fmt.Sprintf("Selected Tasks (%d):", len(m.tasks))))
	s.WriteString("\n")

	for i, t := range m.tasks {
		badge := categoryBadge(string(t.Category))
		s.WriteString(fmt.Sprintf("  %d. ", i+1))
		s.WriteString(badge.Render(string(t.Category)))
		s.WriteString(" ")
		s.WriteString(truncate(t.Description, 40))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(detailLabelStyle.Render("Max Iterations: "))
	s.WriteString(fmt.Sprintf("%d", m.maxIterations))
	s.WriteString("\n\n")

	s.WriteString(helpDescStyle.Render("Press Enter to start, Esc to go back"))
	s.WriteString("\n\n")

	s.WriteString(renderHelp(previewKeys))

	return s.String()
}

type loopConfigKeyMap struct {
	Enter  key.Binding
	Escape key.Binding
}

var loopConfigKeys = loopConfigKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
}
