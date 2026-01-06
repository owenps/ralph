package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/ralph/internal"
)

type createStep int

const (
	stepCategory createStep = iota
	stepDescription
	stepSteps
	stepPreview
	stepConfirmCancel
)

type createModel struct {
	step             createStep
	prevStep         createStep
	categoryCursor   int
	descriptionInput textinput.Model
	stepsInput       textarea.Model
	confirmDialog    *confirmDialog

	// Task data being built
	category    internal.Category
	description string
	steps       []string

	width  int
	height int
}

type saveTaskMsg struct {
	Task *internal.Task
}

type cancelCreateMsg struct{}

func newCreateWizard() createModel {
	ti := textinput.New()
	ti.Placeholder = "A brief overview..."
	ti.CharLimit = 200
	ti.Width = 50

	ta := textarea.New()
	ta.Placeholder = "How to complete the task..."
	ta.SetWidth(50)
	ta.SetHeight(6)

	return createModel{
		step:             stepCategory,
		categoryCursor:   0,
		descriptionInput: ti,
		stepsInput:       ta,
	}
}

func (m createModel) Update(msg tea.Msg) (createModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.descriptionInput.Width = min(50, msg.Width-10)
		m.stepsInput.SetWidth(min(50, msg.Width-10))
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case stepCategory:
			return m.updateCategory(msg)
		case stepDescription:
			return m.updateDescription(msg)
		case stepSteps:
			return m.updateSteps(msg)
		case stepPreview:
			return m.updatePreview(msg)
		case stepConfirmCancel:
			return m.updateConfirmCancel(msg)
		}
	}

	return m, nil
}

func (m createModel) updateCategory(msg tea.KeyMsg) (createModel, tea.Cmd) {
	switch {
	case key.Matches(msg, createKeys.Up):
		if m.categoryCursor > 0 {
			m.categoryCursor--
		}
	case key.Matches(msg, createKeys.Down):
		if m.categoryCursor < len(internal.Categories)-1 {
			m.categoryCursor++
		}
	case key.Matches(msg, createKeys.Enter):
		m.category = internal.Categories[m.categoryCursor]
		m.step = stepDescription
		m.descriptionInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, createKeys.Escape):
		return m, func() tea.Msg { return cancelCreateMsg{} }
	}
	return m, nil
}

func (m createModel) updateDescription(msg tea.KeyMsg) (createModel, tea.Cmd) {
	switch {
	case key.Matches(msg, createKeys.Escape):
		m.step = stepCategory
		m.descriptionInput.Blur()
		return m, nil
	case key.Matches(msg, createKeys.Enter):
		value := strings.TrimSpace(m.descriptionInput.Value())
		if value != "" {
			m.description = value
			m.step = stepSteps
			m.descriptionInput.Blur()
			m.stepsInput.Focus()
			return m, textarea.Blink
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.descriptionInput, cmd = m.descriptionInput.Update(msg)
	return m, cmd
}

func (m createModel) updateSteps(msg tea.KeyMsg) (createModel, tea.Cmd) {
	switch {
	case key.Matches(msg, createKeys.Escape):
		m.step = stepDescription
		m.stepsInput.Blur()
		m.descriptionInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, createKeys.CtrlD):
		m.steps = parseSteps(m.stepsInput.Value())
		m.step = stepPreview
		m.stepsInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.stepsInput, cmd = m.stepsInput.Update(msg)
	return m, cmd
}

func (m createModel) updatePreview(msg tea.KeyMsg) (createModel, tea.Cmd) {
	switch {
	case key.Matches(msg, createKeys.Enter):
		t := internal.NewTask(m.category, m.description, m.steps)
		return m, func() tea.Msg { return saveTaskMsg{Task: t} }
	case key.Matches(msg, createKeys.Edit):
		m.step = stepCategory
		return m, nil
	case key.Matches(msg, createKeys.Escape):
		m.confirmDialog = newConfirmDialog("Discard task?")
		m.prevStep = stepPreview
		m.step = stepConfirmCancel
		return m, nil
	}
	return m, nil
}

func (m createModel) updateConfirmCancel(msg tea.KeyMsg) (createModel, tea.Cmd) {
	switch {
	case key.Matches(msg, createKeys.Left), key.Matches(msg, createKeys.Right):
		m.confirmDialog.Toggle()
	case key.Matches(msg, createKeys.Enter):
		if m.confirmDialog.IsYes() {
			return m, func() tea.Msg { return cancelCreateMsg{} }
		}
		m.step = m.prevStep
		m.confirmDialog = nil
	case key.Matches(msg, createKeys.Escape):
		m.step = m.prevStep
		m.confirmDialog = nil
	}
	return m, nil
}

func (m createModel) View() string {
	var s string

	// s += headerStyle.Render("Create New Task") + "\n\n"

	switch m.step {
	case stepCategory:
		s += m.viewCategory()
	case stepDescription:
		s += m.viewDescription()
	case stepSteps:
		s += m.viewSteps()
	case stepPreview:
		s += m.viewPreview()
	case stepConfirmCancel:
		s += m.confirmDialog.View()
		s += "\n\n"
		s += renderHelp(confirmKeys)
		return s
	}

	return s
}

func (m createModel) viewCategory() string {
	s := inputLabelStyle.Render("Select Category:") + "\n\n"

	for i, cat := range internal.Categories {
		cursor := "  "
		badge := categoryBadge(string(cat))

		if m.categoryCursor == i {
			cursor = cursorStyle.Render("> ")
		}

		s += cursor + badge.Render(string(cat)) + "\n"
	}

	s += "\n"
	s += renderHelp(wizardKeys)
	return s
}

func (m createModel) viewDescription() string {
	s := inputLabelStyle.Render("Task Description:") + "\n\n"
	s += m.descriptionInput.View() + "\n\n"
	s += renderHelp(inputKeys)
	return s
}

func (m createModel) viewSteps() string {
	s := inputLabelStyle.Render("Steps (one per line):") + "\n\n"
	s += m.stepsInput.View() + "\n\n"
	s += renderHelp(textAreaKeys)
	return s
}

func (m createModel) viewPreview() string {
	s := inputLabelStyle.Render("Preview:") + "\n\n"

	badge := categoryBadge(string(m.category))
	s += detailLabelStyle.Render("Category: ") + badge.Render(string(m.category)) + "\n\n"
	s += detailLabelStyle.Render("Description:") + "\n"
	s += detailValueStyle.Render(m.description) + "\n\n"

	if len(m.steps) > 0 {
		s += detailLabelStyle.Render("Steps:") + "\n"
		for i, step := range m.steps {
			s += detailValueStyle.Render(fmt.Sprintf("  %d. %s", i+1, step)) + "\n"
		}
	} else {
		s += detailLabelStyle.Render("Steps: ") + helpDescStyle.Render("(none)") + "\n"
	}

	s += "\n" + helpDescStyle.Render("Looks good?") + "\n\n"

	s += renderHelp(previewKeys)
	return s
}

func parseSteps(input string) []string {
	var steps []string
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			steps = append(steps, trimmed)
		}
	}
	return steps
}

func catDescription(cat internal.Category) string {
	switch cat {
	case internal.CategoryBug:
		return "Bug fix"
	case internal.CategoryFeature:
		return "New feature"
	case internal.CategoryRefactor:
		return "Code refactoring"
	default:
		return ""
	}
}

type createKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Escape key.Binding
	Edit   key.Binding
	CtrlD  key.Binding
}

var createKeys = createKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
	),
	CtrlD: key.NewBinding(
		key.WithKeys("ctrl+d"),
	),
}
