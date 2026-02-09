package internal

import (
	"fmt"
	"os/exec"
)

// BuildClaudeCmd builds an exec.Cmd for launching Claude Code in the given worktree.
// If resume is true, adds --resume flag to continue a previous session.
func BuildClaudeCmd(worktreePath string, resume bool) (*exec.Cmd, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, fmt.Errorf("claude CLI not found: install it from https://docs.anthropic.com/en/docs/claude-code")
	}

	args := []string{}
	if resume {
		args = append(args, "--resume")
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = worktreePath
	return cmd, nil
}
