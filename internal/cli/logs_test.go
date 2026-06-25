package cli

import (
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestSelectAgent(t *testing.T) {
	agents := []core.Agent{{Name: "manager-agent"}, {Name: "backend-agent"}}

	first, err := selectAgent(agents, "")
	if err != nil {
		t.Fatalf("select first: %v", err)
	}
	if first.Name != "manager-agent" {
		t.Fatalf("first agent = %q", first.Name)
	}

	backend, err := selectAgent(agents, "backend-agent")
	if err != nil {
		t.Fatalf("select named: %v", err)
	}
	if backend.Name != "backend-agent" {
		t.Fatalf("named agent = %q", backend.Name)
	}

	if _, err := selectAgent(agents, "missing"); err == nil {
		t.Fatalf("expected missing agent error")
	}
}
