package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newDaemonCommand(ctx *appContext) *cobra.Command {
	var interval time.Duration
	var once bool
	var managerTick bool
	var managerApply bool
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the foreground supervision loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupervisionLoop(cmd, ctx, interval, once, managerTick, managerApply)
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 30*time.Second, "supervision interval")
	cmd.Flags().BoolVar(&once, "once", false, "run one supervision tick and exit")
	cmd.Flags().BoolVar(&managerTick, "manager-tick", false, "send supervision prompts to running manager agents")
	cmd.Flags().BoolVar(&managerApply, "manager-apply", false, "apply structured manager actions from running manager tmux output")
	return cmd
}

func newAFKCommand(ctx *appContext) *cobra.Command {
	var interval time.Duration
	var managerTick bool
	var managerApply bool
	cmd := &cobra.Command{
		Use:   "afk",
		Short: "Run walk-away supervision in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupervisionLoop(cmd, ctx, interval, false, managerTick, managerApply)
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 60*time.Second, "supervision interval")
	cmd.Flags().BoolVar(&managerTick, "manager-tick", false, "send supervision prompts to running manager agents")
	cmd.Flags().BoolVar(&managerApply, "manager-apply", false, "apply structured manager actions from running manager tmux output")
	return cmd
}

func runSupervisionLoop(cmd *cobra.Command, ctx *appContext, interval time.Duration, once bool, managerTick bool, managerApply bool) error {
	if !once {
		release, err := acquireDaemonLock(ctx.store.Root())
		if err != nil {
			return err
		}
		defer release()
	}
	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for {
		changed, err := superviseAll(ctx, managerTick, managerApply)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "supervision tick: changed=%d\n", changed)
		if once {
			return nil
		}
		select {
		case <-runCtx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

func acquireDaemonLock(root string) (func(), error) {
	path := core.DaemonLockPath(root)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("daemon already appears to be running for %s; remove %s if this is stale", root, path)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())
	_ = file.Close()
	return func() { _ = os.Remove(path) }, nil
}

func superviseAll(ctx *appContext, managerTick bool, managerApply bool) (int, error) {
	state, err := ctx.store.Load()
	if err != nil {
		return 0, err
	}
	changed := 0
	for taskID, task := range state.Tasks {
		if task.State == "archived" {
			continue
		}
		_, didChange, err := reconcileTaskAgents(ctx.store, &state, taskID)
		if err != nil {
			return changed, err
		}
		if didChange {
			changed++
		}
		if managerTick && taskHasRunningManager(state.Tasks[taskID]) {
			if err := runManagerTick(ctx, taskID, true, 10, 80, ioDiscard{}); err != nil {
				return changed, err
			}
		}
		if managerApply && taskHasRunningManager(state.Tasks[taskID]) {
			if err := runManagerApplyFromTmux(ctx, taskID, 300, ioDiscard{}); err != nil {
				_ = ctx.store.AddEvent(core.Event{TaskID: taskID, AgentName: "manager-agent", Type: "manager.apply_failed", Message: err.Error()})
			}
		}
	}
	return changed, nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
