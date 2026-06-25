package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newBlockedCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "blocked",
		Short: "Show tasks that need attention",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			count := 0
			for _, task := range state.Tasks {
				if task.State == "archived" || task.State == "done" {
					continue
				}
				reason := attentionReason(task)
				if reason == "" {
					continue
				}
				count++
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", task.ID, task.State, reason, task.Goal)
			}
			if count == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no blocked tasks")
			}
			return nil
		},
	}
}

func attentionReason(task core.Task) string {
	switch task.State {
	case "planning":
		return "plan needs review/approval"
	case "blocked":
		return "task marked blocked"
	}
	for _, agent := range task.Agents {
		if agent.State != "running" {
			return fmt.Sprintf("agent %s is %s", agent.Name, agent.State)
		}
	}
	return ""
}
