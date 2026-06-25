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
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the foreground supervision loop",
		RunE: func(cmd *cobra.Command, args []string) error {
			runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			for {
				changed, err := superviseAll(ctx)
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
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 30*time.Second, "supervision interval")
	cmd.Flags().BoolVar(&once, "once", false, "run one supervision tick and exit")
	return cmd
}

func superviseAll(ctx *appContext) (int, error) {
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
	}
	return changed, nil
}
