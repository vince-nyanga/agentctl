package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWritePlanArtifacts(t *testing.T) {
	workspace := t.TempDir()
	task := Task{
		ID:        "auth-refresh-123456",
		Goal:      "Add refresh-token auth flow",
		Workspace: workspace,
		CreatedAt: time.Now(),
		Repos: []TaskRepo{
			{Name: "backend", WorktreePath: filepath.Join(workspace, "worktrees", "backend")},
			{Name: "frontend", WorktreePath: filepath.Join(workspace, "worktrees", "frontend")},
		},
	}

	if err := CreateTaskWorkspace(workspace, task); err != nil {
		t.Fatalf("CreateTaskWorkspace() error = %v", err)
	}
	if err := WritePlanArtifacts(task); err != nil {
		t.Fatalf("WritePlanArtifacts() error = %v", err)
	}

	plan, err := os.ReadFile(filepath.Join(workspace, "plan.md"))
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	if !strings.Contains(string(plan), "Add refresh-token auth flow") {
		t.Fatalf("plan does not contain goal")
	}
	if _, err := os.Stat(filepath.Join(workspace, "briefs", "backend.md")); err != nil {
		t.Fatalf("backend brief missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "manager-prompt.md")); err != nil {
		t.Fatalf("manager prompt missing: %v", err)
	}
}
