package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newManagerCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "manager", Short: "Manager-agent supervision helpers"}
	cmd.AddCommand(newManagerTickCommand(ctx))
	return cmd
}

func newManagerTickCommand(ctx *appContext) *cobra.Command {
	var send bool
	var eventLimit int
	var outputLines int
	cmd := &cobra.Command{
		Use:   "tick <task-id>",
		Short: "Build a manager supervision prompt for a task",
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
			events, err := ctx.store.ListEvents(task.ID, eventLimit)
			if err != nil {
				return err
			}
			outputs := collectAgentOutputs(task, outputLines)
			prompt := core.BuildManagerTickPrompt(task, events, outputs)
			promptPath := filepath.Join(task.Workspace, "manager-tick.md")
			if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
				return err
			}

			if !send {
				fmt.Fprint(cmd.OutOrStdout(), prompt)
				return nil
			}
			manager, err := findAgentByRole(task, "manager")
			if err != nil {
				return err
			}
			if !core.TmuxSessionExists(manager.TmuxName) {
				return fmt.Errorf("manager session %s is not running", manager.TmuxName)
			}
			if err := core.SendTmux(manager.TmuxName, prompt); err != nil {
				return err
			}
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, AgentName: manager.Name, Type: "manager.tick_sent", Message: "sent supervision prompt to manager"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sent manager tick to %s\n", manager.Name)
			return nil
		},
	}
	cmd.Flags().BoolVar(&send, "send", false, "send prompt to the manager tmux session")
	cmd.Flags().IntVar(&eventLimit, "events", 10, "number of recent events to include")
	cmd.Flags().IntVar(&outputLines, "lines", 80, "number of tmux lines to include per live agent")
	return cmd
}

func collectAgentOutputs(task core.Task, lines int) []core.AgentOutput {
	var outputs []core.AgentOutput
	for _, agent := range task.Agents {
		if !core.TmuxSessionExists(agent.TmuxName) {
			continue
		}
		output, err := core.TailTmux(agent.TmuxName, lines)
		if err != nil {
			continue
		}
		outputs = append(outputs, core.AgentOutput{AgentName: agent.Name, Output: output})
	}
	return outputs
}

func findAgentByRole(task core.Task, role string) (core.Agent, error) {
	for _, agent := range task.Agents {
		if agent.Role == role {
			return agent, nil
		}
	}
	return core.Agent{}, fmt.Errorf("task %s has no %s agent", task.ID, role)
}
