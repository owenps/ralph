package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/ralph/internal/ui"
)

func main() {
	var app tea.Model

	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			app = ui.NewAppForInit()
		case "help", "--help", "-h":
			printHelp()
			return
		case "version", "--version", "-v":
			fmt.Println("ralph v0.1.0")
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			fmt.Fprintf(os.Stderr, "Run 'ralph help' for usage.\n")
			os.Exit(1)
		}
	} else {
		app = ui.NewApp()
	}

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Ralph: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	help := `Ralph - TUI task management for your codebase

Usage:
  ralph          Launch Ralph in the current project
  ralph init     Initialize Ralph in the current directory
  ralph help     Show this help message
  ralph version  Show version

Getting started:
  1. Run 'ralph init' in any project directory
  2. Tasks are stored in .ralph/tasks.yaml
  3. Run 'ralph' to manage tasks

Navigation:
  j/k or arrows  Navigate up/down
  enter          Select/confirm
  esc            Back/cancel
  q              Quit
`
	fmt.Print(help)
}
