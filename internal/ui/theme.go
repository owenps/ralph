package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Version can be set via ldflags at build time
var Version = "v0.1.0"

var (
	colorYellow     = lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#FFD93D"}
	colorDarkYellow = lipgloss.AdaptiveColor{Light: "#806000", Dark: "#B8960C"}
	colorTeal       = lipgloss.AdaptiveColor{Light: "#0D9488", Dark: "#5EEAD4"}
	colorText       = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#FFFFFF"}
	colorGray       = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	colorDarkGray   = lipgloss.AdaptiveColor{Light: "#E5E5E5", Dark: "#444444"}
	colorBlack      = lipgloss.Color("#000000")
	colorBug        = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#FF6B6B"}
	colorFeature    = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#2A9D8F"}
	colorRefactor   = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
)

var (
	completedStyle = lipgloss.NewStyle().
		Foreground(colorDarkYellow)
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true).
			Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 2)

	selectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true).
				Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Padding(1, 0)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorTeal)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(colorBlack).
			Background(colorYellow).
			Padding(0, 1).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorDarkGray).
				Padding(0, 1)

	bugBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorBug).
			Padding(0, 1)

	featureBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorFeature).
			Padding(0, 1)

	refactorBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorRefactor).
			Padding(0, 1)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorBug).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(colorDarkYellow).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorFeature).
			Bold(true)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorTeal).
				Bold(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	codeStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Background(colorDarkGray).
			Padding(0, 1)
)

func categoryBadge(category string) lipgloss.Style {
	switch category {
	case "bug":
		return bugBadge
	case "feature":
		return featureBadge
	case "refactor":
		return refactorBadge
	default:
		return inactiveTabStyle
	}
}

func banner() string {
	lines := []string{
		"█████▀█████      ▄▄▄▄   █████               ████",
		"█████ █████  ▄███ ████  █████     ▄▄█▀███▄  ████",
		"█████ █████  ████ ████  █████    ████ ████  ████▄███▄",
		"█████▀███▄▄  ▄▄▄▄▄████  █████ ▄  ████▀███▀  ████ ████",
		"█████ █████  ████▄████  ▀████▄█  ▀███       ████ ████",
	}

	// Column positions for white highlights (vertical stems in each letter)
	highlightCols := map[int]bool{
		2: true, 15: true, 25: true, 34: true, 45: true,
	}

	baseStyle := lipgloss.NewStyle().Foreground(colorYellow)
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	var result strings.Builder
	for i, line := range lines {
		col := 0
		for _, r := range line {
			// Only highlight on rows 1, 2, 3 (middle rows)
			if highlightCols[col] && i >= 1 && i <= 3 {
				result.WriteString(highlightStyle.Render(string(r)))
			} else {
				result.WriteString(baseStyle.Render(string(r)))
			}
			col++
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return borderStyle.Padding(1).Render(result.String())
}
