package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newStatusCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show task status",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			if len(state.Tasks) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no tasks")
				return nil
			}
			for _, task := range state.Tasks {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d repos\t%d agents\t%s\n", task.ID, task.State, len(task.Repos), len(task.Agents), task.Goal)
			}
			return nil
		},
	}
}

func newOpenCommand(ctx *appContext) *cobra.Command {
	var agentName string
	cmd := &cobra.Command{
		Use:   "open [task-id]",
		Short: "Attach to an agent tmux session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			taskID := ""
			if len(args) == 1 {
				taskID = args[0]
			}
			target, message, err := resolveOpenTarget(state, taskID, agentName)
			if err != nil {
				return err
			}
			if target == "" {
				fmt.Fprint(cmd.OutOrStdout(), message)
				return nil
			}
			return runAttached(target)
		},
	}
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name to open")
	return cmd
}

func resolveOpenTarget(state core.State, taskID string, agentName string) (string, string, error) {
	if taskID != "" {
		task, ok := state.Tasks[taskID]
		if !ok {
			return "", "", fmt.Errorf("unknown task %q", taskID)
		}
		return resolveAgentInTask(task, agentName)
	}

	if agentName != "" {
		var matches []core.Agent
		for _, task := range state.Tasks {
			for _, agent := range task.Agents {
				if agent.Name == agentName {
					matches = append(matches, agent)
				}
			}
		}
		if len(matches) == 1 {
			return matches[0].TmuxName, "", nil
		}
		if len(matches) > 1 {
			return "", "agent name appears in multiple tasks; pass the task id too\n", nil
		}
		return "", "", fmt.Errorf("agent %q not found", agentName)
	}

	active := activeTasks(state)
	if len(active) == 1 {
		return resolveAgentInTask(active[0], "")
	}
	if len(active) == 0 {
		return "", "no active tasks. Start with `agentctl plan ...` or open `agentctl dashboard`.\n", nil
	}
	var b strings.Builder
	b.WriteString("multiple active tasks. Choose one:\n\n")
	for _, task := range active {
		fmt.Fprintf(&b, "  %s  %s  (%d agents)\n", task.ID, task.State, len(task.Agents))
		for _, agent := range task.Agents {
			fmt.Fprintf(&b, "    agent: %s [%s] state=%s\n", agent.Name, agent.Role, agent.State)
		}
	}
	b.WriteString("\nRun: agentctl open <task-id> --agent <agent-name>\n")
	b.WriteString("Or use: agentctl dashboard\n")
	return "", b.String(), nil
}

func resolveAgentInTask(task core.Task, agentName string) (string, string, error) {
	if len(task.Agents) == 0 {
		return "", fmt.Sprintf("task %s has no agents yet. Approve and dispatch it, or open `agentctl dashboard`.\n", task.ID), nil
	}
	if agentName != "" {
		for _, agent := range task.Agents {
			if agent.Name == agentName {
				return agent.TmuxName, "", nil
			}
		}
		return "", "", fmt.Errorf("agent %q not found in task %s", agentName, task.ID)
	}
	if len(task.Agents) == 1 {
		return task.Agents[0].TmuxName, "", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "task %s has multiple agents. Choose one:\n\n", task.ID)
	for _, agent := range task.Agents {
		fmt.Fprintf(&b, "  %s [%s] state=%s\n", agent.Name, agent.Role, agent.State)
	}
	b.WriteString("\nRun: agentctl open " + task.ID + " --agent <agent-name>\n")
	return "", b.String(), nil
}

func activeTasks(state core.State) []core.Task {
	tasks := make([]core.Task, 0, len(state.Tasks))
	for _, task := range state.Tasks {
		if task.State == "archived" || task.State == "done" {
			continue
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
	return tasks
}
