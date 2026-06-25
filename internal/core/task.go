package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func NewTaskID(goal string) string {
	return fmt.Sprintf("%s-%s", TaskSlug(goal), time.Now().Format("150405"))
}

func TaskSlug(goal string) string {
	slug := strings.ToLower(goal)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	parts := strings.Split(slug, "-")
	if len(parts) > 4 {
		parts = parts[:4]
	}
	slug = strings.Join(parts, "-")
	if slug == "" {
		slug = "task"
	}
	return slug
}

func CreateTaskWorkspace(root string, task Task) error {
	dirs := []string{
		task.Workspace,
		filepath.Join(task.Workspace, "briefs"),
		filepath.Join(task.Workspace, "worktrees"),
		filepath.Join(task.Workspace, "logs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func WritePlanArtifacts(task Task) error {
	plan := fmt.Sprintf("# Plan: %s\n\nTask ID: `%s`\n\n## Goal\n\n%s\n\n## Repositories\n\n%s\n\n## Manager Instructions\n\nInvestigate the registered worktrees, produce a concrete implementation plan, identify repo-specific work, define contracts between repos, and update the briefs in `briefs/` before dispatch.\n\nDo not start coding until the plan is approved unless policy explicitly allows it.\n", task.Goal, task.ID, task.Goal, taskRepoList(task))

	managerPrompt := fmt.Sprintf(`You are the manager agent for Agent Mission Control task %s.

Goal:
%s

Repos:
%s

Workspace:
%s

Your job:
1. Inspect the relevant repos.
2. Write or refine plan.md.
3. Write architecture-notes.md if needed.
4. Write decisions.md with open decisions.
5. Write repo-specific implementation briefs in briefs/.
6. Stop and report that the plan is ready for approval.

Do not implement the feature yet. Plan first.
`, task.ID, task.Goal, taskRepoList(task), task.Workspace)

	files := map[string]string{
		"plan.md":               plan,
		"manager-prompt.md":     managerPrompt,
		"decisions.md":          "# Decisions\n\n",
		"manager-log.md":        "# Manager Log\n\n",
		"risk-review.md":        "# Risk Review\n\n",
		"architecture-notes.md": "# Architecture Notes\n\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(task.Workspace, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	for _, repo := range task.Repos {
		brief := fmt.Sprintf("# Brief: %s\n\nTask: `%s`\n\nGoal:\n%s\n\nRepo worktree:\n%s\n\nStatus:\nDraft. Manager should refine this before dispatch.\n", repo.Name, task.ID, task.Goal, repo.WorktreePath)
		if err := os.WriteFile(filepath.Join(task.Workspace, "briefs", repo.Name+".md"), []byte(brief), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func taskRepoList(task Task) string {
	var b strings.Builder
	for _, repo := range task.Repos {
		fmt.Fprintf(&b, "- %s: %s\n", repo.Name, repo.WorktreePath)
	}
	return b.String()
}
