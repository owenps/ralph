package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/jolteon/internal"
)

type setupStep int

const (
	setupWelcome setupStep = iota
	setupDone
)

type setupModel struct {
	step   setupStep
	width  int
	height int
}

type setupCompleteMsg struct{}

func newSetupWizard() setupModel {
	return setupModel{
		step: setupWelcome,
	}
}

func (m setupModel) Init() tea.Cmd {
	return nil
}

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case setupWelcome:
			return m.updateWelcome(msg)
		case setupDone:
			return m.updateDone(msg)
		}
	}

	return m, nil
}

func (m setupModel) updateWelcome(msg tea.KeyMsg) (setupModel, tea.Cmd) {
	if key.Matches(msg, setupKeys.Enter) {
		cfg := &internal.GlobalConfig{Initialized: true}
		if err := internal.SaveGlobalConfig(cfg); err != nil {
			return m, nil
		}
		m.step = setupDone
		return m, nil
	}
	return m, nil
}

func (m setupModel) updateDone(msg tea.KeyMsg) (setupModel, tea.Cmd) {
	if key.Matches(msg, setupKeys.Enter) {
		return m, func() tea.Msg { return setupCompleteMsg{} }
	}
	return m, nil
}

func (m setupModel) View() string {
	var s string

	s += appTitle() + "\n\n"

	switch m.step {
	case setupWelcome:
		s += m.viewWelcome()
	case setupDone:
		s += m.viewDone()
	}

	return s
}

func (m setupModel) viewWelcome() string {
	var s string

	s += inputLabelStyle.Render("Welcome to Jolteon") + "\n\n"
	s += helpDescStyle.Render("A thin coordination layer above Claude Code.") + "\n\n"
	s += detailLabelStyle.Render("How it works:") + "\n"
	s += helpDescStyle.Render("  1. Run ") + codeStyle.Render("jolteon init") + helpDescStyle.Render(" in any project to start tracking tasks") + "\n"
	s += helpDescStyle.Render("  2. Data is stored in ") + codeStyle.Render(".jolteon/") + "\n"
	s += helpDescStyle.Render("  3. Add ") + codeStyle.Render(".jolteon/") + helpDescStyle.Render(" to .gitignore (or commit it!)") + "\n\n"

	s += renderHelp([]keyBinding{
		{Key: "enter", Desc: "continue"},
		{Key: "ctrl+c", Desc: "quit"},
	})

	return s
}

func (m setupModel) viewDone() string {
	s := successStyle.Render("✔︎") + " Jolteon is ready to use\n\n"
	s += helpDescStyle.Render("Run ") + codeStyle.Render("jolteon init") + helpDescStyle.Render(" in a project directory to get started.") + "\n\n"
	s += renderHelp([]keyBinding{
		{Key: "enter", Desc: "exit"},
	})

	return s
}

type setupKeyMap struct {
	Enter key.Binding
}

var setupKeys = setupKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
}
