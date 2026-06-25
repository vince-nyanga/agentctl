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
	model := newDashboardModel(state, map[string][]core.Event{
		task.ID: {
			{TaskID: task.ID, AgentName: "manager-agent", Type: "task.created", Message: "created planning workspace", CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)},
		},
	})
	model.width = 120

	view := model.View()
	for _, want := range []string{"Recent Events", "manager-agent", "created planning workspace"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard view missing %q:\n%s", want, view)
		}
	}
}
