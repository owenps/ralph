package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenps/jolteon/internal"
	"github.com/owenps/jolteon/internal/ui"
)

func main() {
	var app tea.Model

	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			app = ui.NewAppForInit()
		case "sync":
			runSync()
			return
		case "clean":
			runClean()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "version", "--version", "-v":
			fmt.Println("jolteon " + ui.Version)
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			fmt.Fprintf(os.Stderr, "Run 'jolteon help' for usage.\n")
			os.Exit(1)
		}
	} else {
		app = ui.NewApp()
	}

	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running jolteon: %v\n", err)
		os.Exit(1)
	}
}

func runSync() {
	tasksPath, err := internal.TasksFilePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !internal.ProjectExists() {
		fmt.Fprintf(os.Stderr, "No jolteon project found. Run 'jolteon init' first.\n")
		os.Exit(1)
	}

	store := internal.NewTaskStore(tasksPath)
	if err := store.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	added, err := internal.SyncIssues(store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing issues: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Synced %d new issues from GitHub.\n", added)
}

func runClean() {
	tasksPath, err := internal.TasksFilePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !internal.ProjectExists() {
		fmt.Fprintf(os.Stderr, "No jolteon project found. Run 'jolteon init' first.\n")
		os.Exit(1)
	}

	store := internal.NewTaskStore(tasksPath)
	if err := store.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	cleaned := internal.WorktreeCleanAll(store)
	fmt.Printf("Cleaned %d worktrees.\n", cleaned)
}

func printHelp() {
	help := `Jolteon - A thin coordination layer above Claude Code

Usage:
  jolteon          Launch Jolteon in the current project
  jolteon init     Initialize Jolteon in the current directory
  jolteon sync     Import GitHub issues into task backlog
  jolteon clean    Remove worktrees for completed tasks
  jolteon help     Show this help message
  jolteon version  Show version

Getting started:
  1. Run 'jolteon init' in any project directory
  2. Tasks are stored in .jolteon/tasks.yaml
  3. Run 'jolteon' to manage tasks and launch Claude sessions

Navigation:
  j/k or arrows  Navigate up/down
  enter          Start/resume Claude session on task
  n              New task
  p              Create PR for active task
  x              Clean worktree
  S              Sync GitHub issues
  q              Quit
`
	fmt.Print(help)
}
