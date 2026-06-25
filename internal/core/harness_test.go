package core

import "testing"

func TestClassifyHarnessOutput(t *testing.T) {
	harness := Harness{BusyPatterns: []string{"working"}, ApprovalPatterns: []string{"allow"}}
	if got := ClassifyHarnessOutput(harness, "please allow this command"); got != "waiting_for_approval" {
		t.Fatalf("approval classification = %q", got)
	}
	if got := ClassifyHarnessOutput(harness, "agent is working"); got != "running" {
		t.Fatalf("busy classification = %q", got)
	}
	if got := ClassifyHarnessOutput(harness, ""); got != "unknown" {
		t.Fatalf("empty classification = %q", got)
	}
}
