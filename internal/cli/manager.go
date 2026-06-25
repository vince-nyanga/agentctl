package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newManagerCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "manager", Short: "Manager-agent supervision helpers"}
	cmd.AddCommand(newManagerTickCommand(ctx))
	cmd.AddCommand(newManagerApplyCommand(ctx))
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
			return runManagerTick(ctx, args[0], send, eventLimit, outputLines, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&send, "send", false, "send prompt to the manager tmux session")
	cmd.Flags().IntVar(&eventLimit, "events", 10, "number of recent events to include")
	cmd.Flags().IntVar(&outputLines, "lines", 80, "number of tmux lines to include per live agent")
	return cmd
}

func newManagerApplyCommand(ctx *appContext) *cobra.Command {
	var file string
	var fromTmux bool
	var lines int
	cmd := &cobra.Command{
		Use:   "apply <task-id>",
		Short: "Apply a structured manager action block",
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
			var text string
			if fromTmux {
				manager, err := findAgentByRole(task, "manager")
				if err != nil {
					return err
				}
				if !core.TmuxSessionExists(manager.TmuxName) {
					return fmt.Errorf("manager session %s is not running", manager.TmuxName)
				}
				output, err := core.TailTmux(manager.TmuxName, lines)
				if err != nil {
					return err
				}
				text = output
			} else {
				if file == "" {
					file = filepath.Join(task.Workspace, "manager-response.md")
				}
				data, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				text = string(data)
			}
			actions, err := core.ParseManagerActions(text)
			if err != nil {
				return err
			}
			for _, action := range actions {
				if err := applyManagerAction(ctx, &state, task, action); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "applied %d manager actions\n", len(actions))
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "manager response file; defaults to task manager-response.md")
	cmd.Flags().BoolVar(&fromTmux, "from-tmux", false, "read action block from live manager tmux output")
	cmd.Flags().IntVar(&lines, "lines", 300, "number of tmux lines to capture with --from-tmux")
	return cmd
}

func applyManagerAction(ctx *appContext, state *core.State, task core.Task, action core.ManagerAction) error {
	switch action.Type {
	case "approval":
		approvalType := action.ApprovalType
		if approvalType == "" {
			approvalType = "other"
		}
		if _, err := ctx.store.CreateApproval(core.Approval{TaskID: task.ID, AgentName: "manager-agent", Type: approvalType, Title: action.Title, Description: action.Description, Risk: action.Risk, RecommendedAction: action.RecommendedAction}); err != nil {
			return err
		}
		return ctx.store.AddEvent(core.Event{TaskID: task.ID, AgentName: "manager-agent", Type: "approval.requested", Message: action.Title})
	case "nudge":
		agent, err := findAgentByName(task, action.AgentName)
		if err != nil {
			return err
		}
		if !core.TmuxSessionExists(agent.TmuxName) {
			return fmt.Errorf("agent session %s is not running", agent.TmuxName)
		}
		if err := core.SendTmux(agent.TmuxName, action.Message); err != nil {
			return err
		}
		return ctx.store.AddEvent(core.Event{TaskID: task.ID, AgentName: agent.Name, Type: "agent.nudged", Message: action.Message})
	case "done":
		task.State = "done"
		task.ManagerNote = action.Message
		task.UpdatedAt = time.Now()
		state.Tasks[task.ID] = task
		if err := ctx.store.Save(*state); err != nil {
			return err
		}
		return ctx.store.AddEvent(core.Event{TaskID: task.ID, AgentName: "manager-agent", Type: "task.done", Message: action.Message})
	default:
		return fmt.Errorf("unsupported manager action %q", action.Type)
	}
}

func findAgentByName(task core.Task, name string) (core.Agent, error) {
	for _, agent := range task.Agents {
		if agent.Name == name {
			return agent, nil
		}
	}
	return core.Agent{}, fmt.Errorf("task %s has no agent %q", task.ID, name)
}

func runManagerTick(ctx *appContext, taskID string, send bool, eventLimit, outputLines int, out io.Writer) error {
	state, err := ctx.store.Load()
	if err != nil {
		return err
	}
	task, ok := state.Tasks[taskID]
	if !ok {
		return fmt.Errorf("unknown task %q", taskID)
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
		fmt.Fprint(out, prompt)
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
	fmt.Fprintf(out, "sent manager tick to %s\n", manager.Name)
	return nil
}

func taskHasRunningManager(task core.Task) bool {
	manager, err := findAgentByRole(task, "manager")
	if err != nil {
		return false
	}
	return core.TmuxSessionExists(manager.TmuxName)
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
