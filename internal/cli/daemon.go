package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newDaemonCommand(ctx *appContext) *cobra.Command {
	var interval time.Duration
	var once bool
	var managerTick bool
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the foreground supervision loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupervisionLoop(cmd, ctx, interval, once, managerTick)
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 30*time.Second, "supervision interval")
	cmd.Flags().BoolVar(&once, "once", false, "run one supervision tick and exit")
	cmd.Flags().BoolVar(&managerTick, "manager-tick", false, "send supervision prompts to running manager agents")
	return cmd
}

func newAFKCommand(ctx *appContext) *cobra.Command {
	var interval time.Duration
	var managerTick bool
	cmd := &cobra.Command{
		Use:   "afk",
		Short: "Run walk-away supervision in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupervisionLoop(cmd, ctx, interval, false, managerTick)
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 60*time.Second, "supervision interval")
	cmd.Flags().BoolVar(&managerTick, "manager-tick", false, "send supervision prompts to running manager agents")
	return cmd
}

func runSupervisionLoop(cmd *cobra.Command, ctx *appContext, interval time.Duration, once bool, managerTick bool) error {
	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for {
		changed, err := superviseAll(ctx, managerTick)
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

func superviseAll(ctx *appContext, managerTick bool) (int, error) {
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
	}
	return changed, nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
