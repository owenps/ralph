package ui

import "github.com/charmbracelet/lipgloss"

type confirmDialog struct {
	Message  string
	Selected int // 0 = Yes, 1 = No
}

func newConfirmDialog(message string) *confirmDialog {
	return &confirmDialog{
		Message:  message,
		Selected: 1, // Default to No
	}
}

func (c *confirmDialog) Toggle() {
	if c.Selected == 0 {
		c.Selected = 1
	} else {
		c.Selected = 0
	}
}

func (c *confirmDialog) SelectYes() {
	c.Selected = 0
}

func (c *confirmDialog) IsYes() bool {
	return c.Selected == 0
}

func (c *confirmDialog) View() string {
	var yesStyle, noStyle lipgloss.Style

	if c.Selected == 0 {
		yesStyle = selectedMenuItemStyle
		noStyle = menuItemStyle
	} else {
		yesStyle = menuItemStyle
		noStyle = selectedMenuItemStyle
	}

	message := inputLabelStyle.Render(c.Message)
	yes := yesStyle.Render("Yes")
	no := noStyle.Render("No")

	return message + "\n\n" + yes + "  " + no
}
