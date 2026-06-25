package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newSuperviseCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "supervise <task-id>",
		Short: "Reconcile tracked agent sessions for a task",
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
			changed := false
			for i := range task.Agents {
				agent := &task.Agents[i]
				nextState := "stopped"
				if core.TmuxSessionExists(agent.TmuxName) {
					nextState = "running"
				}
				if agent.State != nextState {
					previous := agent.State
					agent.State = nextState
					changed = true
					if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, AgentName: agent.Name, Type: "agent.state_changed", Message: fmt.Sprintf("%s -> %s", previous, nextState)}); err != nil {
						return err
					}
				}
			}
			if changed {
				task.UpdatedAt = time.Now()
				state.Tasks[task.ID] = task
				if err := ctx.store.Save(state); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "supervised %s: %d agents, changed=%v\n", task.ID, len(task.Agents), changed)
			return nil
		},
	}
}
