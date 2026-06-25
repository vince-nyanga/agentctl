package core

import "testing"

func TestParseManagerActions(t *testing.T) {
	text := `summary
AGENTCTL_ACTIONS:
[
  {"type":"approval","approval_type":"api_contract","title":"Approve contract"},
  {"type":"nudge","agent_name":"backend-agent","message":"Run tests"}
]
END_AGENTCTL_ACTIONS`
	actions, err := ParseManagerActions(text)
	if err != nil {
		t.Fatalf("ParseManagerActions() error = %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("actions len = %d", len(actions))
	}
	if actions[0].Type != "approval" || actions[1].AgentName != "backend-agent" {
		t.Fatalf("actions = %#v", actions)
	}
}

func TestParseManagerActionsRejectsUnknown(t *testing.T) {
	_, err := ParseManagerActions(`AGENTCTL_ACTIONS:
[{"type":"shell"}]
END_AGENTCTL_ACTIONS`)
	if err == nil {
		t.Fatalf("expected unsupported action error")
	}
}
