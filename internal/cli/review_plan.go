package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newReviewPlanCommand(ctx *appContext) *cobra.Command {
	var showBriefs bool
	cmd := &cobra.Command{
		Use:   "review-plan <task-id>",
		Short: "Print a task plan before approval",
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

			planPath := filepath.Join(task.Workspace, "plan.md")
			plan, err := os.ReadFile(planPath)
			if err != nil {
				return fmt.Errorf("read plan: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(plan))

			if !showBriefs {
				return nil
			}
			for _, repo := range task.Repos {
				briefPath := filepath.Join(task.Workspace, "briefs", repo.Name+".md")
				brief, err := os.ReadFile(briefPath)
				if err != nil {
					return fmt.Errorf("read brief %s: %w", repo.Name, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n\n--- brief: %s ---\n\n%s", repo.Name, string(brief))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&showBriefs, "briefs", false, "include repo implementation briefs")
	return cmd
}
