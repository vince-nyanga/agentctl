package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newApprovalsCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "approvals",
		Short: "Show pending approvals",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			count := 0
			for _, task := range state.Tasks {
				if task.State != "planning" {
					continue
				}
				count++
				fmt.Fprintf(cmd.OutOrStdout(), "%s\tplan_approval\t%s\n", task.ID, task.Goal)
			}
			if count == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no pending approvals")
			}
			return nil
		},
	}
}
