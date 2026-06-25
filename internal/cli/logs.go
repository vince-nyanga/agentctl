package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newLogsCommand(ctx *appContext) *cobra.Command {
	var agentName string
	var lines int
	cmd := &cobra.Command{
		Use:   "logs <task-id>",
		Short: "Show recent output for a task agent",
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
			agent, err := selectAgent(task.Agents, agentName)
			if err != nil {
				return err
			}
			if core.TmuxSessionExists(agent.TmuxName) {
				output, err := core.TailTmux(agent.TmuxName, lines)
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), output)
				return nil
			}
			if agent.LogPath != "" {
				data, err := os.ReadFile(agent.LogPath)
				if err == nil {
					fmt.Fprint(cmd.OutOrStdout(), string(data))
					return nil
				}
			}
			return fmt.Errorf("no live tmux session or log file found for agent %s", agent.Name)
		},
	}
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name; defaults to the first tracked agent")
	cmd.Flags().IntVar(&lines, "lines", 80, "number of tmux lines to capture")
	return cmd
}

func selectAgent(agents []core.Agent, name string) (core.Agent, error) {
	if len(agents) == 0 {
		return core.Agent{}, fmt.Errorf("task has no tracked agents")
	}
	if name == "" {
		return agents[0], nil
	}
	for _, agent := range agents {
		if agent.Name == name {
			return agent, nil
		}
	}
	return core.Agent{}, fmt.Errorf("agent %q not found", name)
}
