package ui

import "strings"

type keyBinding struct {
	Key  string
	Desc string
}

func renderHelp(bindings []keyBinding) string {
	var parts []string
	for _, b := range bindings {
		part := helpKeyStyle.Render(b.Key) + " " + helpDescStyle.Render(b.Desc)
		parts = append(parts, part)
	}
	return helpStyle.Render(strings.Join(parts, "  "))
}

var (
	navigationKeys = []keyBinding{
		{Key: "j/k", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "q", Desc: "quit"},
	}

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
