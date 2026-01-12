package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type LoopState struct {
	RunID          string
	CurrentTaskIdx int
	CurrentTaskID  string
	TotalTasks     int
	Iteration      int
	MaxIterations  int
	Status         RunStatus
	Output         string
	GracefulStop   bool
	ImmediateAbort bool
}

type LoopRunner struct {
	config   *GlobalConfig
	tasks    []Task
	taskIDs  []string
	store    *TaskStore
	progress *ProgressManager
	state    *LoopState
	cancel   context.CancelFunc
}

func NewLoopRunner(config *GlobalConfig, tasks []Task, taskIDs []string, store *TaskStore, maxIterations int) (*LoopRunner, error) {
	pm, err := NewProgressManager()
	if err != nil {
		return nil, err
	}

	// Filter to only the selected tasks in order
	taskMap := make(map[string]Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	var selectedTasks []Task
	for _, id := range taskIDs {
		if t, ok := taskMap[id]; ok {
			selectedTasks = append(selectedTasks, t)
		}
	}

	run := pm.StartRun(taskIDs, tasks, maxIterations)
	if err := pm.Save(); err != nil {
		return nil, err
	}

	return &LoopRunner{
		config:   config,
		tasks:    selectedTasks,
		taskIDs:  taskIDs,
		store:    store,
		progress: pm,
		state: &LoopState{
			RunID:         run.ID,
			TotalTasks:    len(selectedTasks),
			MaxIterations: maxIterations,
			Status:        RunStatusRunning,
		},
	}, nil
}

func (lr *LoopRunner) GetState() *LoopState {
	return lr.state
}

func (lr *LoopRunner) RequestGracefulStop() {
	lr.state.GracefulStop = true
}

func (lr *LoopRunner) RequestImmediateAbort() {
	lr.state.ImmediateAbort = true
	if lr.cancel != nil {
		lr.cancel()
	}
}

func (lr *LoopRunner) Run(ctx context.Context, outputChan chan<- string) error {
	ctx, lr.cancel = context.WithCancel(ctx)
	defer lr.cancel()

	for i, task := range lr.tasks {
		if lr.state.GracefulStop || lr.state.ImmediateAbort {
			break
		}

		lr.state.CurrentTaskIdx = i
		lr.state.CurrentTaskID = task.ID

		lr.progress.StartTask(lr.state.RunID, task.ID)
		lr.progress.Save()

		outputChan <- fmt.Sprintf("Starting task %d/%d: %s", i+1, lr.state.TotalTasks, task.Description)

		completed, err := lr.runTask(ctx, task, outputChan)
		if err != nil {
			if lr.state.ImmediateAbort {
				lr.progress.CompleteRun(lr.state.RunID, RunStatusAborted, "User aborted")
				lr.progress.Save()
				lr.state.Status = RunStatusAborted
				return nil
			}

			lr.progress.FailTask(lr.state.RunID, task.ID, err.Error())
			lr.progress.CompleteRun(lr.state.RunID, RunStatusFailed, err.Error())
			lr.progress.Save()
			lr.state.Status = RunStatusFailed
			return err
		}

		if completed {
			// Auto-commit
			commitSHA, err := lr.commitChanges(task)
			if err != nil {
				lr.progress.FailTask(lr.state.RunID, task.ID, "Commit failed: "+err.Error())
				lr.progress.CompleteRun(lr.state.RunID, RunStatusFailed, err.Error())
				lr.progress.Save()
				lr.state.Status = RunStatusFailed
				return err
			}

			lr.progress.CompleteTask(lr.state.RunID, task.ID, commitSHA)
			lr.progress.Save()

			// Mark task as done in task store
			lr.store.ToggleDone(task.ID)
			lr.store.Save()

			outputChan <- fmt.Sprintf("Task completed: %s (commit: %s)", task.Description, commitSHA[:7])
		}

		if lr.state.GracefulStop {
			break
		}
	}

	if lr.state.GracefulStop {
		lr.progress.CompleteRun(lr.state.RunID, RunStatusAborted, "User requested stop")
		lr.state.Status = RunStatusAborted
	} else {
		lr.progress.CompleteRun(lr.state.RunID, RunStatusCompleted, "")
		lr.state.Status = RunStatusCompleted
	}
	lr.progress.Save()

	return nil
}

func (lr *LoopRunner) runTask(ctx context.Context, task Task, outputChan chan<- string) (bool, error) {
	for lr.state.Iteration < lr.state.MaxIterations {
		if lr.state.GracefulStop || lr.state.ImmediateAbort {
			return false, nil
		}

		lr.state.Iteration++
		lr.progress.IncrementIteration(lr.state.RunID, task.ID)

		outputChan <- fmt.Sprintf("Iteration %d/%d", lr.state.Iteration, lr.state.MaxIterations)

		output, completed, err := lr.invokeClaud(ctx, task)
		if err != nil {
			return false, err
		}

		lr.progress.AddOutputLog(lr.state.RunID, task.ID, lr.state.Iteration, output)
		lr.progress.Save()

		lr.state.Output = output
		outputChan <- output

		if completed {
			return true, nil
		}
	}

	return false, fmt.Errorf("max iterations reached without completing task")
}

func (lr *LoopRunner) invokeClaud(ctx context.Context, task Task) (string, bool, error) {
	prompt := lr.buildPrompt(task)

	args := []string{"-p", prompt, "--output-format", "json"}

	// Add configured flags
	if lr.config.Claude.Model != "" {
		args = append(args, "--model", lr.config.Claude.Model)
	}
	if len(lr.config.Claude.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(lr.config.Claude.AllowedTools, ","))
	}
	if lr.config.Claude.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", lr.config.Claude.MaxTurns))
	}
	if lr.config.Claude.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", lr.config.Claude.SystemPrompt)
	}

	timeout := time.Duration(lr.config.Claude.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", false, fmt.Errorf("timeout after %v", timeout)
		}
		if ctx.Err() == context.Canceled {
			return "", false, nil
		}
		return stderr.String(), false, fmt.Errorf("claude error: %v - %s", err, stderr.String())
	}

	output := stdout.String()

	// Parse JSON output to check for completion
	completed := lr.checkCompletion(output)

	return output, completed, nil
}

func (lr *LoopRunner) buildPrompt(task Task) string {
	var sb strings.Builder

	sb.WriteString("You are completing a task for the Ralph task runner.\n\n")
	sb.WriteString("TASK:\n")
	sb.WriteString(fmt.Sprintf("- ID: %s\n", task.ID))
	sb.WriteString(fmt.Sprintf("- Category: %s\n", task.Category))
	sb.WriteString(fmt.Sprintf("- Description: %s\n", task.Description))

	if len(task.Steps) > 0 {
		sb.WriteString("- Steps:\n")
		for i, step := range task.Steps {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
	}

	sb.WriteString("\nComplete this task. When finished, ensure all changes are saved.\n")
	sb.WriteString("When you have completed the task, output exactly: TASK_COMPLETE\n")
	sb.WriteString("If you cannot complete the task, output exactly: TASK_FAILED: <reason>\n")

	return sb.String()
}

func (lr *LoopRunner) checkCompletion(output string) bool {
	// Try to parse as JSON first
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		// Check for completion in JSON structure
		if text, ok := result["result"].(string); ok {
			if strings.Contains(text, "TASK_COMPLETE") {
				return true
			}
		}
	}

	// Fall back to string search
	return strings.Contains(output, "TASK_COMPLETE")
}

func (lr *LoopRunner) commitChanges(task Task) (string, error) {
	// Stage all changes
	addCmd := exec.Command("git", "add", "-A")
	if err := addCmd.Run(); err != nil {
		return "", fmt.Errorf("git add failed: %w", err)
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOut, err := statusCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}

	if len(statusOut) == 0 {
		// No changes to commit, return empty SHA
		return "no-changes", nil
	}

	// Commit with formatted message
	commitMsg := fmt.Sprintf("[ralph] %s: %s", task.Category, task.Description)
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	if err := commitCmd.Run(); err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Get commit SHA
	shaCmd := exec.Command("git", "rev-parse", "HEAD")
	shaOut, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	return strings.TrimSpace(string(shaOut)), nil
}

func CheckWorkingDirectoryClean() error {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// Filter out untracked files in .ralph directory
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		// Skip .ralph directory changes
		if strings.Contains(line, ".ralph/") {
			continue
		}
		// If there's any other change, working directory is not clean
		if len(strings.TrimSpace(line)) > 0 {
			return fmt.Errorf("working directory has uncommitted changes")
		}
	}

	return nil
}
