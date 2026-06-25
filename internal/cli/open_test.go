package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestResolveOpenTargetNoTasks(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	target, message, err := resolveOpenTarget(state, "", "")
	if err != nil {
		t.Fatalf("resolveOpenTarget() error = %v", err)
	}
	if target != "" || !strings.Contains(message, "no active tasks") {
		t.Fatalf("target=%q message=%q", target, message)
	}
}

func TestResolveOpenTargetSingleTaskSingleAgent(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	state.Tasks["task-1"] = core.Task{ID: "task-1", State: "running", CreatedAt: time.Now(), Agents: []core.Agent{{Name: "manager-agent", TmuxName: "tmux-target"}}}
	target, message, err := resolveOpenTarget(state, "", "")
	if err != nil {
		t.Fatalf("resolveOpenTarget() error = %v", err)
	}
	if target != "tmux-target" || message != "" {
		t.Fatalf("target=%q message=%q", target, message)
	}
}

func TestResolveOpenTargetMultipleAgents(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	state.Tasks["task-1"] = core.Task{ID: "task-1", State: "running", CreatedAt: time.Now(), Agents: []core.Agent{{Name: "manager-agent"}, {Name: "worker-agent"}}}
	target, message, err := resolveOpenTarget(state, "task-1", "")
	if err != nil {
		t.Fatalf("resolveOpenTarget() error = %v", err)
	}
	if target != "" || !strings.Contains(message, "multiple agents") {
		t.Fatalf("target=%q message=%q", target, message)
	}
}
