package cli

import (
	"strings"
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestAttentionReason(t *testing.T) {
	if got := attentionReason(core.Task{State: "planning"}); !strings.Contains(got, "plan") {
		t.Fatalf("planning reason = %q", got)
	}
	if got := attentionReason(core.Task{State: "running", Agents: []core.Agent{{Name: "worker", State: "stopped"}}}); !strings.Contains(got, "worker") {
		t.Fatalf("agent reason = %q", got)
	}
	if got := attentionReason(core.Task{State: "running", Agents: []core.Agent{{Name: "worker", State: "running"}}}); got != "" {
		t.Fatalf("running reason = %q", got)
	}
}
