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
			approvals, err := ctx.store.ListApprovals("", "pending")
			if err != nil {
				return err
			}
			if len(approvals) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no pending approvals")
				return nil
			}
			for _, approval := range approvals {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%s\t%s\n", approval.ID, approval.TaskID, approval.Type, approval.Risk, approval.Title)
			}
			return nil
		},
	}
}
