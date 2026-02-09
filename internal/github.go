package internal

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type GitHubIssue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

type ghIssueLabelRaw struct {
	Name string `json:"name"`
}

type ghIssueRaw struct {
	Number int               `json:"number"`
	Title  string            `json:"title"`
	Body   string            `json:"body"`
	Labels []ghIssueLabelRaw `json:"labels"`
}

// FetchOpenIssues uses gh CLI to fetch open issues.
func FetchOpenIssues() ([]GitHubIssue, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found: install it from https://cli.github.com")
	}

	cmd := exec.Command("gh", "issue", "list", "--state", "open", "--json", "number,title,body,labels", "--limit", "100")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh issue list failed: %w", err)
	}

	var raw []ghIssueRaw
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}

	issues := make([]GitHubIssue, len(raw))
	for i, r := range raw {
		labels := make([]string, len(r.Labels))
		for j, l := range r.Labels {
			labels[j] = l.Name
		}
		issues[i] = GitHubIssue{
			Number: r.Number,
			Title:  r.Title,
			Body:   r.Body,
			Labels: labels,
		}
	}

	return issues, nil
}

// IssueToTask converts a GitHub issue to a Task.
func IssueToTask(issue GitHubIssue) *Task {
	category := inferCategory(issue.Labels)

	var steps []string
	if issue.Body != "" {
		// Split body into lines and use non-empty lines as steps
		for _, line := range strings.Split(issue.Body, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				steps = append(steps, trimmed)
			}
		}
	}

	return &Task{
		ID:          fmt.Sprintf("gh-%d", issue.Number),
		Category:    category,
		Description: issue.Title,
		Steps:       steps,
		Status:      TaskStatusPending,
		Source:      TaskSourceGitHub,
		IssueNumber: issue.Number,
	}
}

// SyncIssues imports open GitHub issues into the task store.
// Returns the number of new tasks added.
func SyncIssues(store *TaskStore) (int, error) {
	issues, err := FetchOpenIssues()
	if err != nil {
		return 0, err
	}

	added := 0
	for _, issue := range issues {
		id := fmt.Sprintf("gh-%d", issue.Number)
		if store.FindByID(id) != nil {
			continue // Already imported
		}
		task := IssueToTask(issue)
		store.Add(task)
		added++
	}

	if added > 0 {
		if err := store.Save(); err != nil {
			return added, err
		}
	}

	return added, nil
}

// CloseIssue closes a GitHub issue by number.
func CloseIssue(number int) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found")
	}

	cmd := exec.Command("gh", "issue", "close", fmt.Sprintf("%d", number))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh issue close failed: %s", string(out))
	}
	return nil
}

func inferCategory(labels []string) Category {
	for _, label := range labels {
		lower := strings.ToLower(label)
		switch {
		case strings.Contains(lower, "bug"):
			return CategoryBug
		case strings.Contains(lower, "feature") || strings.Contains(lower, "enhancement"):
			return CategoryFeature
		case strings.Contains(lower, "refactor"):
			return CategoryRefactor
		case strings.Contains(lower, "research"):
			return CategoryResearch
		}
	}
	return CategoryFeature // default
}
