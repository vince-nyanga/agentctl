package core

import (
	"fmt"
	"strings"
)

type AgentOutput struct {
	AgentName string
	Output    string
}

func BuildManagerTickPrompt(task Task, events []Event, outputs []AgentOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are the manager agent for Agent Mission Control task %s.\n\n", task.ID)
	fmt.Fprintf(&b, "Goal:\n%s\n\n", task.Goal)
	fmt.Fprintf(&b, "Current state: %s\n", task.State)
	fmt.Fprintf(&b, "Workspace: %s\n\n", task.Workspace)

	b.WriteString("Repositories:\n")
	if len(task.Repos) == 0 {
		b.WriteString("- none\n")
	}
	for _, repo := range task.Repos {
		ownership := "owned"
		if !repo.Owned {
			ownership = "attached"
		}
		fmt.Fprintf(&b, "- %s: %s (branch: %s, %s)\n", repo.Name, repo.WorktreePath, repo.Branch, ownership)
	}

	b.WriteString("\nAgents:\n")
	if len(task.Agents) == 0 {
		b.WriteString("- none\n")
	}
	for _, agent := range task.Agents {
		fmt.Fprintf(&b, "- %s: role=%s harness=%s repo=%s state=%s tmux=%s\n", agent.Name, agent.Role, agent.Harness, agent.Repo, agent.State, agent.TmuxName)
	}

	b.WriteString("\nRecent events:\n")
	if len(events) == 0 {
		b.WriteString("- none\n")
	}
	for _, event := range events {
		actor := event.AgentName
		if actor == "" {
			actor = "system"
		}
		fmt.Fprintf(&b, "- %s [%s] %s: %s\n", event.CreatedAt.Format("2006-01-02 15:04:05"), actor, event.Type, event.Message)
	}

	b.WriteString("\nRecent agent output:\n")
	if len(outputs) == 0 {
		b.WriteString("- none captured\n")
	}
	for _, output := range outputs {
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", output.AgentName, strings.TrimSpace(output.Output))
	}

	b.WriteString(`
Your job:
1. Classify each worker as running, blocked, stale, failed, done, or unknown.
2. If routine recovery is possible, state the exact instruction that should be sent to the worker.
3. If user input is required, write a concise approval/request with the decision needed and your recommendation.
4. If implementation appears complete, say what review/test/PR step should happen next.
5. Do not ask the user for routine implementation details you can resolve from the plan, briefs, code, logs, or tests.
6. Keep the response operational: status, actions taken or recommended, blockers, next step.
`)

	return b.String()
}
