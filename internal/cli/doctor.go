package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newDoctorCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check local dependencies and configured harnesses",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			failed := false
			fmt.Fprintln(out, "Required tools:")
			for _, tool := range []string{"git", "tmux"} {
				if !printCheck(out, tool, core.HasCommand(tool)) {
					failed = true
				}
			}

			fmt.Fprintln(out, "\nOptional tools:")
			printCheck(out, "gh", core.HasCommand("gh"))

			fmt.Fprintln(out, "\nHarnesses:")
			names := make([]string, 0, len(state.Config.Harnesses))
			for name := range state.Config.Harnesses {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				harness := state.Config.Harnesses[name]
				ok := harness.Command != "" && core.HasCommand(harness.Command)
				fmt.Fprintf(out, "  %-10s", name)
				printCheck(out, harness.Command, ok)
			}

			if failed {
				return fmt.Errorf("doctor found missing required dependencies")
			}
			return nil
		},
	}
}

func printCheck(out io.Writer, name string, ok bool) bool {
	if ok {
		fmt.Fprintf(out, "ok      %s\n", name)
		return true
	}
	fmt.Fprintf(out, "missing %s\n", name)
	return false
}
