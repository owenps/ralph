package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/owenps/jolteon/internal"
)

// Version can be set via ldflags at build time
var Version = "v0.2.0"

var (
	colorYellow     = lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#FFD93D"}
	colorDarkYellow = lipgloss.AdaptiveColor{Light: "#806000", Dark: "#B8960C"}
	colorDimYellow  = lipgloss.AdaptiveColor{Light: "#A07808", Dark: "#8B7500"}
	colorText       = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#FFFFFF"}
	colorGray       = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	colorDarkGray   = lipgloss.AdaptiveColor{Light: "#E5E5E5", Dark: "#444444"}
	colorBlack      = lipgloss.Color("#000000")
	// Muted category colors
	colorBug      = lipgloss.AdaptiveColor{Light: "#B05050", Dark: "#E88080"}
	colorFeature  = lipgloss.AdaptiveColor{Light: "#3D7068", Dark: "#5BA89D"}
	colorRefactor = lipgloss.AdaptiveColor{Light: "#8B6FC0", Dark: "#B9A8E0"}
	colorResearch = lipgloss.AdaptiveColor{Light: "#5B80C0", Dark: "#8FB0E0"}
	colorNotes    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	// Status colors
	colorPending = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	colorActive  = lipgloss.AdaptiveColor{Light: "#2E7D32", Dark: "#66BB6A"}
	colorDone    = lipgloss.AdaptiveColor{Light: "#1565C0", Dark: "#42A5F5"}
	colorFailed  = lipgloss.AdaptiveColor{Light: "#B05050", Dark: "#E88080"}
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorGray).
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
			BorderForeground(colorDimYellow)

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

	researchBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorResearch).
			Padding(0, 1)

	notesBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorNotes).
			Padding(0, 1)

	// Status badges
	pendingBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPending).
			Padding(0, 1)

	activeBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorActive).
			Padding(0, 1)

	doneBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorDone).
			Padding(0, 1)

	failedBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorFailed).
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

	warnBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBug).
			Padding(1, 2)

	successStyle = lipgloss.NewStyle().
			Foreground(colorActive).
			Bold(true)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorGray).
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
	case "research":
		return researchBadge
	case "notes":
		return notesBadge
	default:
		return inactiveTabStyle
	}
}

func statusBadge(status internal.TaskStatus) lipgloss.Style {
	switch status {
	case internal.TaskStatusPending:
		return pendingBadge
	case internal.TaskStatusActive:
		return activeBadge
	case internal.TaskStatusDone:
		return doneBadge
	case internal.TaskStatusFailed:
		return failedBadge
	default:
		return pendingBadge
	}
}

func appTitle() string {
	bracketStyle := lipgloss.NewStyle().Foreground(colorDimYellow)
	versionStyle := lipgloss.NewStyle().Foreground(colorGray)
	promptStyle := lipgloss.NewStyle().Foreground(colorGray)
	labelStyle := lipgloss.NewStyle().Foreground(colorGray)
	titleBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGray).
		Padding(0, 1)

	// Get current directory, replace home with ~
	dir, _ := os.Getwd()
	if home, err := os.UserHomeDir(); err == nil {
		dir = strings.Replace(dir, home, "~", 1)
	}

	title := promptStyle.Render(">") + " " +
		bracketStyle.Render("[") +
		titleStyle.Render("jolteon") +
		bracketStyle.Render("]") + " " +
		versionStyle.Render("("+Version+")") + "\n\n" +
		labelStyle.Render("directory:") + " " + dir

	return titleBoxStyle.Render(title)
}

// renderHelp renders a help bar from key bindings.
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
