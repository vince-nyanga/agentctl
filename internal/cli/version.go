package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "agentctl %s\ncommit: %s\ndate: %s\n", version.Version, version.Commit, version.Date)
		},
	}
}
