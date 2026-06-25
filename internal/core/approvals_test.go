package core

import "testing"

func TestApprovalLifecycle(t *testing.T) {
	store := NewStore(t.TempDir())
	created, err := store.CreateApproval(Approval{TaskID: "task-1", Type: "plan", Title: "Approve plan", RecommendedAction: "approve"})
	if err != nil {
		t.Fatalf("CreateApproval() error = %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected approval id")
	}

	pending, err := store.ListApprovals("task-1", "pending")
	if err != nil {
		t.Fatalf("ListApprovals() error = %v", err)
	}
	if len(pending) != 1 || pending[0].Title != "Approve plan" {
		t.Fatalf("pending approvals = %#v", pending)
	}

	resolved, err := store.ResolveApproval(created.ID, "approved")
	if err != nil {
		t.Fatalf("ResolveApproval() error = %v", err)
	}
	if resolved.State != "resolved" || resolved.Resolution != "approved" {
		t.Fatalf("resolved approval = %#v", resolved)
	}
}
