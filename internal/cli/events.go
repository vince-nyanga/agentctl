package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEventsCommand(ctx *appContext) *cobra.Command {
	var taskID string
	var limit int
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show recent event log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			events, err := ctx.store.ListEvents(taskID, limit)
			if err != nil {
				return err
			}
			if len(events) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no events")
				return nil
			}
			for _, event := range events {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", event.CreatedAt.Format("2006-01-02 15:04:05"), event.TaskID, event.AgentName, event.Type, event.Message)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&taskID, "task", "", "filter events by task id")
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum events to show")
	return cmd
}
