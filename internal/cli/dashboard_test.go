package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestDashboardRendersRecentEvents(t *testing.T) {
	task := core.Task{
		ID:        "auth-refresh-123456",
		Goal:      "Add refresh-token auth flow",
		State:     "running",
		Workspace: t.TempDir(),
		CreatedAt: time.Now(),
	}
	state := core.DefaultState(t.TempDir())
	state.Tasks[task.ID] = task
	events := []core.Event{{TaskID: task.ID, AgentName: "manager-agent", Type: "task.created", Message: "created planning workspace", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}}
	model := newDashboardModel(state, map[string][]core.Event{
		task.ID: {
			events[0],
		},
	}, events, nil)
	model.width = 120
	model.tab = 4

	view := model.View()
	for _, want := range []string{"Recent Events", "manager-agent", "created planning workspace"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard view missing %q:\n%s", want, view)
		}
	}
}

func TestDashboardRendersOverviewAndTabs(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	state.Tasks["task-1"] = core.Task{ID: "task-1", Goal: "Goal", State: "planning", CreatedAt: time.Now()}
	model := newDashboardModel(state, nil, nil, []core.Approval{{ID: 1, TaskID: "task-1", Type: "plan", Title: "Approve task plan"}})
	model.width = 120

	view := model.View()
	for _, want := range []string{"Overview", "Tasks", "Approvals", "Needs attention"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard overview missing %q:\n%s", want, view)
		}
	}
}
