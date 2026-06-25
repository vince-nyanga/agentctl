package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestPrepareWorkerDispatchPrompt(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "briefs"), 0o755); err != nil {
		t.Fatalf("mkdir briefs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "briefs", "app.md"), []byte("Do not implement until approved."), 0o644); err != nil {
		t.Fatalf("write brief: %v", err)
	}
	path, err := prepareWorkerDispatchPrompt(core.Task{ID: "task-1", Workspace: workspace}, core.TaskRepo{Name: "app"})
	if err != nil {
		t.Fatalf("prepareWorkerDispatchPrompt() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	text := string(data)
	for _, want := range []string{"must now implement", "approval gate is now satisfied", "Original Brief", "Do not implement until approved"} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt missing %q:\n%s", want, text)
		}
	}
}
