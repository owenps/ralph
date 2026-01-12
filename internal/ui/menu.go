package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type menuOption int

const (
	menuTasks menuOption = iota
	menuSettings
)

// menuItem implements list.Item
type menuItem struct {
	title string
	option menuOption
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return "" }
func (i menuItem) FilterValue() string { return i.title }

// menuDelegate renders menu items
type menuDelegate struct{}

func (d menuDelegate) Height() int                             { return 1 }
func (d menuDelegate) Spacing() int                            { return 0 }
func (d menuDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d menuDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	mi, ok := item.(menuItem)
	if !ok {
		return
	}

	cursor := "  "
	style := menuItemStyle

	if index == m.Index() {
		cursor = cursorStyle.Render("> ")
		style = selectedMenuItemStyle
	}

	fmt.Fprint(w, cursor+style.Render(mi.title))
}

type menuModel struct {
	list   list.Model
	width  int
	height int
}

type openTasksFromMenuMsg struct{}
type openSettingsFromMenuMsg struct{}

func newMenuModel() menuModel {
	items := []list.Item{
		menuItem{title: "Tasks", option: menuTasks},
		menuItem{title: "Settings", option: menuSettings},
	}

	l := list.New(items, menuDelegate{}, 30, 4)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return menuModel{list: l}
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, menuKeys.Enter):
			if item := m.list.SelectedItem(); item != nil {
				if mi, ok := item.(menuItem); ok {
					switch mi.option {
					case menuTasks:
						return m, func() tea.Msg { return openTasksFromMenuMsg{} }
					case menuSettings:
						return m, func() tea.Msg { return openSettingsFromMenuMsg{} }
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m menuModel) View() string {
	s := appTitle() + "\n\n"
	s += m.list.View()
	s += "\n"
	s += renderHelp(menuHelpKeys)
	return s
}

type menuKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
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
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}

var menuHelpKeys = []keyBinding{
	{Key: "j/k", Desc: "navigate"},
	{Key: "enter", Desc: "select"},
	{Key: "q", Desc: "quit"},
}
