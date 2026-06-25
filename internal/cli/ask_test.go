package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestWriteAskArtifacts(t *testing.T) {
	workspace := t.TempDir()
	task := core.Task{ID: "task-1", Goal: "Tell me about this repo", Workspace: workspace, CreatedAt: time.Now(), Repos: []core.TaskRepo{{Name: "app", WorktreePath: filepath.Join(workspace, "worktrees", "app")}}}
	if err := core.CreateTaskWorkspace(workspace, task); err != nil {
		t.Fatalf("CreateTaskWorkspace() error = %v", err)
	}
	if err := core.WriteAskArtifacts(task); err != nil {
		t.Fatalf("WriteAskArtifacts() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(workspace, "manager-prompt.md"))
	if err != nil {
		t.Fatalf("read manager prompt: %v", err)
	}
	for _, want := range []string{"Classify", "answer", "implement", "Tell me about this repo"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("manager prompt missing %q:\n%s", want, string(data))
		}
	}
}
