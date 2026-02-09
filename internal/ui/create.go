package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/jolteon/internal"
)

type createStep int

const (
	stepCategory createStep = iota
	stepDescription
	stepSteps
	stepPreview
	stepConfirmCancel
)

// categoryItem wraps internal.Category to implement list.Item
type categoryItem struct {
	category internal.Category
}

func (i categoryItem) Title() string       { return string(i.category) }
func (i categoryItem) Description() string { return "" }
func (i categoryItem) FilterValue() string { return string(i.category) }

// categoryDelegate renders category items in the list
type categoryDelegate struct{}

func (d categoryDelegate) Height() int                             { return 1 }
func (d categoryDelegate) Spacing() int                            { return 0 }
func (d categoryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d categoryDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(categoryItem)
	if !ok {
		return
	}

	cursor := "  "
	if index == m.Index() {
		cursor = cursorStyle.Render("> ")
	}

	badge := categoryBadge(string(ci.category))
	fmt.Fprint(w, cursor+badge.Render(string(ci.category)))
}

type createModel struct {
	step             createStep
	prevStep         createStep
	categoryList     list.Model
	descriptionInput textinput.Model
	stepsInput       textarea.Model
	confirmDialog    *confirmDialog

	// Task data being built
	editingID   string // non-empty if editing existing task
	category    internal.Category
	description string
	steps       []string

	width  int
	height int
}

type saveTaskMsg struct {
	Task      *internal.Task
	EditingID string // non-empty if updating existing task
}

type cancelCreateMsg struct{}

func newCategoryList(selectedIndex int) list.Model {
	items := make([]list.Item, len(internal.Categories))
	for i, cat := range internal.Categories {
		items[i] = categoryItem{category: cat}
	}

	l := list.New(items, categoryDelegate{}, 30, len(internal.Categories)+2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.Select(selectedIndex)

	return l
}

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
		categoryList:     newCategoryList(0),
		descriptionInput: ti,
		stepsInput:       ta,
	}
}

func newCreateWizardWithTask(task internal.Task) createModel {
	ti := textinput.New()
	ti.Placeholder = "A brief overview..."
	ti.CharLimit = 200
	ti.Width = 50
	ti.SetValue(task.Description)

	ta := textarea.New()
	ta.Placeholder = "How to complete the task..."
	ta.SetWidth(50)
	ta.SetHeight(6)
	ta.SetValue(strings.Join(task.Steps, "\n"))

	// Find category index
	categoryIndex := 0
	for i, cat := range internal.Categories {
		if cat == task.Category {
			categoryIndex = i
			break
		}
	}

	return createModel{
		step:             stepCategory,
		categoryList:     newCategoryList(categoryIndex),
		descriptionInput: ti,
		stepsInput:       ta,
		editingID:        task.ID,
		category:         task.Category,
		description:      task.Description,
		steps:            task.Steps,
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
	case key.Matches(msg, createKeys.Enter):
		if item := m.categoryList.SelectedItem(); item != nil {
			if ci, ok := item.(categoryItem); ok {
				m.category = ci.category
				m.step = stepDescription
				m.descriptionInput.Focus()
				return m, textinput.Blink
			}
		}
	case key.Matches(msg, createKeys.Escape):
		return m, func() tea.Msg { return cancelCreateMsg{} }
	}

	// Let the list handle navigation
	var cmd tea.Cmd
	m.categoryList, cmd = m.categoryList.Update(msg)
	return m, cmd
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
		return m, func() tea.Msg { return saveTaskMsg{Task: t, EditingID: m.editingID} }
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
	s += m.categoryList.View()
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

// Help key binding presets for wizard views
var (
	wizardKeys = []keyBinding{
		{Key: "j/k", Desc: "navigate"},
		{Key: "enter", Desc: "confirm"},
		{Key: "esc", Desc: "back"},
		{Key: "ctrl+c", Desc: "quit"},
	}

	inputKeys = []keyBinding{
		{Key: "enter", Desc: "confirm"},
		{Key: "esc", Desc: "back"},
	}

	textAreaKeys = []keyBinding{
		{Key: "ctrl+d", Desc: "done"},
		{Key: "esc", Desc: "back"},
	}

	previewKeys = []keyBinding{
		{Key: "enter", Desc: "save"},
		{Key: "e", Desc: "edit"},
		{Key: "esc", Desc: "cancel"},
	}

	confirmKeys = []keyBinding{
		{Key: "←/→", Desc: "select"},
		{Key: "enter", Desc: "confirm"},
		{Key: "esc", Desc: "back"},
	}
)

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
