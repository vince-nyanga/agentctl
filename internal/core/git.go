package core

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

func GitRemote(path string) string {
	out, err := exec.Command("git", "-C", path, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func GitTopLevel(path string) (string, error) {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GitDefaultBranch(path string) string {
	candidates := [][]string{
		{"symbolic-ref", "refs/remotes/origin/HEAD", "--short"},
		{"branch", "--show-current"},
	}
	for _, args := range candidates {
		cmdArgs := append([]string{"-C", path}, args...)
		out, err := exec.Command("git", cmdArgs...).Output()
		if err == nil {
			branch := strings.TrimSpace(string(out))
			branch = strings.TrimPrefix(branch, "origin/")
			if branch != "" {
				return branch
			}
		}
	}
	return "main"
}

func CreateWorktree(source, branch, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	base := GitDefaultBranch(source)
	cmd := exec.Command("git", "-C", source, "worktree", "add", "-b", branch, target, base)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if strings.Contains(msg, "already exists") {
			cmd = exec.Command("git", "-C", source, "worktree", "add", target, branch)
			stderr.Reset()
			cmd.Stderr = &stderr
			if retryErr := cmd.Run(); retryErr == nil {
				return nil
			}
		}
		return fmt.Errorf("git worktree add failed: %s", strings.TrimSpace(msg))
	}
	return nil
}

func RemoveWorktree(source, target string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return nil
	}
	cmd := exec.Command("git", "-C", source, "worktree", "remove", target)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func PruneWorktrees(source string) error {
	cmd := exec.Command("git", "-C", source, "worktree", "prune")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree prune failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func GitStatusShort(path string) string {
	out, err := exec.Command("git", "-C", path, "status", "--short").Output()
	if err != nil {
		return "unknown"
	}
	status := strings.TrimSpace(string(out))
	if status == "" {
		return "clean"
	}
	return status
}

func IsDirtyStatus(status string) bool {
	status = strings.TrimSpace(status)
	return status != "" && status != "clean" && status != "unknown"
}

func CurrentBranch(path string) string {
	out, err := exec.Command("git", "-C", path, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func CreatePullRequest(path, title, body string) error {
	if !HasCommand("gh") {
		return fmt.Errorf("gh is required to create pull requests")
	}
	args := []string{"pr", "create", "--title", title, "--body", body}
	cmd := exec.Command("gh", args...)
	cmd.Dir = path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr create failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}
