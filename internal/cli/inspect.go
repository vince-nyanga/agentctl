package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newInspectCommand(ctx *appContext) *cobra.Command {
	var eventLimit int
	cmd := &cobra.Command{
		Use:   "inspect <task-id>",
		Short: "Inspect a task's repos, agents, and recent events",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			task, ok := state.Tasks[args[0]]
			if !ok {
				return fmt.Errorf("unknown task %q", args[0])
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Task: %s\n", task.ID)
			fmt.Fprintf(out, "State: %s\n", task.State)
			fmt.Fprintf(out, "Goal: %s\n", task.Goal)
			fmt.Fprintf(out, "Workspace: %s\n", task.Workspace)
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Repos:")
			if len(task.Repos) == 0 {
				fmt.Fprintln(out, "  none")
			}
			for _, repo := range task.Repos {
				status := core.GitStatusShort(repo.WorktreePath)
				branch := core.CurrentBranch(repo.WorktreePath)
				fmt.Fprintf(out, "  - %s\n", repo.Name)
				fmt.Fprintf(out, "    worktree: %s\n", repo.WorktreePath)
				fmt.Fprintf(out, "    branch: %s\n", branch)
				fmt.Fprintf(out, "    status: %s\n", status)
			}
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Agents:")
			if len(task.Agents) == 0 {
				fmt.Fprintln(out, "  none")
			}
			for _, agent := range task.Agents {
				live := core.TmuxSessionExists(agent.TmuxName)
				fmt.Fprintf(out, "  - %s\n", agent.Name)
				fmt.Fprintf(out, "    role: %s\n", agent.Role)
				fmt.Fprintf(out, "    harness: %s\n", agent.Harness)
				fmt.Fprintf(out, "    state: %s\n", agent.State)
				fmt.Fprintf(out, "    tmux: %s\n", agent.TmuxName)
				fmt.Fprintf(out, "    live: %v\n", live)
			}
			fmt.Fprintln(out)

			events, err := ctx.store.ListEvents(task.ID, eventLimit)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "Recent Events:")
			if len(events) == 0 {
				fmt.Fprintln(out, "  none")
			}
			for _, event := range events {
				actor := event.AgentName
				if actor == "" {
					actor = "system"
				}
				fmt.Fprintf(out, "  - %s [%s] %s: %s\n", event.CreatedAt.Format("2006-01-02 15:04:05"), actor, event.Type, event.Message)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&eventLimit, "events", 10, "number of recent events to show")
	return cmd
}
