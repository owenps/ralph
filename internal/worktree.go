package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const worktreeDir = ".worktrees"

// WorktreeCreate creates a git worktree for the given task.
// Branch: jolteon/task-{id}, Path: .worktrees/task-{id}
func WorktreeCreate(task *Task) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	wtPath := filepath.Join(cwd, worktreeDir, "task-"+task.ID)
	branch := "jolteon/task-" + task.ID

	// If worktree already exists, reuse it
	if _, err := os.Stat(wtPath); err == nil {
		return wtPath, branch, nil
	}

	// Ensure .worktrees directory exists
	if err := os.MkdirAll(filepath.Join(cwd, worktreeDir), 0755); err != nil {
		return "", "", fmt.Errorf("failed to create worktrees dir: %w", err)
	}

	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Branch might already exist from a previous attempt
		cmd2 := exec.Command("git", "worktree", "add", wtPath, branch)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return "", "", fmt.Errorf("git worktree add failed: %s\n%s", string(out), string(out2))
		}
	}

	return wtPath, branch, nil
}

// WorktreeRemove removes the worktree and branch for a task.
func WorktreeRemove(taskID string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	wtPath := filepath.Join(cwd, worktreeDir, "task-"+taskID)

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
	cmd.Run() // Ignore errors — may not exist

	// Delete the branch
	branch := "jolteon/task-" + taskID
	cmd = exec.Command("git", "branch", "-d", branch)
	cmd.Run() // Ignore errors — may not exist

	// Prune stale worktrees
	cmd = exec.Command("git", "worktree", "prune")
	cmd.Run()

	return nil
}

// WorktreeExists checks if a worktree for the task exists.
func WorktreeExists(taskID string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	wtPath := filepath.Join(cwd, worktreeDir, "task-"+taskID)
	_, err = os.Stat(wtPath)
	return err == nil
}

// WorktreePush pushes the branch to origin.
func WorktreePush(branch string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s", string(out))
	}
	return nil
}

// WorktreeCreatePR creates a pull request using gh CLI.
// Returns the PR number.
func WorktreeCreatePR(task *Task, branch string) (int, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return 0, fmt.Errorf("gh CLI not found: install it from https://cli.github.com")
	}

	title := fmt.Sprintf("[jolteon] %s: %s", task.Category, task.Description)
	body := task.Description
	if len(task.Steps) > 0 {
		body += "\n\n## Steps\n"
		for i, step := range task.Steps {
			body += fmt.Sprintf("%d. %s\n", i+1, step)
		}
	}
	if task.Source == TaskSourceGitHub && task.IssueNumber > 0 {
		body += fmt.Sprintf("\n\nCloses #%d", task.IssueNumber)
	}

	args := []string{"pr", "create", "--title", title, "--body", body, "--head", branch}
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("gh pr create failed: %s", string(out))
	}

	// Parse PR number from URL output (e.g., https://github.com/owner/repo/pull/42)
	url := strings.TrimSpace(string(out))
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		if num, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			return num, nil
		}
	}

	return 0, nil
}

// WorktreeCleanAll removes worktrees for done/failed tasks.
func WorktreeCleanAll(store *TaskStore) int {
	cleaned := 0
	for _, task := range store.All() {
		if (task.Status == TaskStatusDone || task.Status == TaskStatusFailed) && WorktreeExists(task.ID) {
			if err := WorktreeRemove(task.ID); err == nil {
				store.SetWorktreePath(task.ID, "")
				store.SetBranch(task.ID, "")
				cleaned++
			}
		}
	}
	if cleaned > 0 {
		store.Save()
	}
	return cleaned
}
