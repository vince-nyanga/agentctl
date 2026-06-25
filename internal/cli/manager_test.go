package cli

import (
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestFindAgentByRole(t *testing.T) {
	task := core.Task{ID: "task-1", Agents: []core.Agent{{Name: "manager-agent", Role: "manager"}}}
	agent, err := findAgentByRole(task, "manager")
	if err != nil {
		t.Fatalf("find manager: %v", err)
	}
	if agent.Name != "manager-agent" {
		t.Fatalf("agent = %q", agent.Name)
	}
	if _, err := findAgentByRole(task, "worker"); err == nil {
		t.Fatalf("expected missing role error")
	}
}
