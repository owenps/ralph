package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	Name        string
	Description string
}

type menuModel struct {
	items    []menuItem
	cursor   int
	selected int
	width    int
	height   int
}

func newMenu(items []menuItem) menuModel {
	return menuModel{
		items:    items,
		cursor:   0,
		selected: -1,
	}
}

type menuSelectMsg struct {
	Index int
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, menuKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, menuKeys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, menuKeys.Enter):
			m.selected = m.cursor
			return m, func() tea.Msg { return menuSelectMsg{Index: m.cursor} }
		}
	}

	return m, nil
}

func (m menuModel) View() string {
	s := banner() + "\n\n"

	for i, item := range m.items {
		cursor := "  "
		style := menuItemStyle

		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
			style = selectedMenuItemStyle
		}

		s += cursor + style.Render(item.Name) + "\n"
	}

	s += "\n"
	s += renderHelp(navigationKeys)

	return s
}

type menuKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}

var menuKeys = menuKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
}
