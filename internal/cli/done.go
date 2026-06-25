package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newDoneCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "done <task-id>",
		Short: "Mark a task as done without cleaning it up",
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
			if task.State == "archived" {
				return fmt.Errorf("task %s is archived", task.ID)
			}
			task.State = "done"
			task.UpdatedAt = time.Now()
			state.Tasks[task.ID] = task
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.done", Message: "task marked done"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "marked %s done\n", task.ID)
			return nil
		},
	}
}
