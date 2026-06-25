package core

import (
	"strings"
	"testing"
	"time"
)

func TestBuildManagerTickPrompt(t *testing.T) {
	task := Task{
		ID:        "auth-refresh-123456",
		Goal:      "Add refresh-token auth",
		State:     "running",
		Workspace: "/tmp/task",
		Repos:     []TaskRepo{{Name: "backend", WorktreePath: "/tmp/backend", Branch: "agent/auth", Owned: true}},
		Agents:    []Agent{{Name: "backend-agent", Role: "worker", Harness: "opencode", Repo: "backend", State: "running", TmuxName: "tmux-backend"}},
	}
	events := []Event{{TaskID: task.ID, AgentName: "backend-agent", Type: "agent.state_changed", Message: "stopped -> running", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}}
	prompt := BuildManagerTickPrompt(task, events, []AgentOutput{{AgentName: "backend-agent", Output: "tests passing"}})

	for _, want := range []string{"auth-refresh-123456", "Add refresh-token auth", "backend-agent", "tests passing", "Classify each worker"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
