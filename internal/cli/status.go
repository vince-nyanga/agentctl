package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
		Use:   "open <task-id>",
		Short: "Attach to an agent tmux session",
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
			for _, agent := range task.Agents {
				if agentName == "" || agent.Name == agentName {
					return runAttached(agent.TmuxName)
				}
			}
			return fmt.Errorf("agent %q not found", agentName)
		},
	}
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name to open")
	return cmd
}
