package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

type appContext struct {
	root  string
	store *core.Store
}

func Execute() error {
	ctx := &appContext{}
	rootCmd := &cobra.Command{
		Use:   "agentctl",
		Short: "Agent Mission Control for multi-repo coding-agent workflows",
		Long: `Agent Mission Control for multi-repo coding-agent workflows.

Run agentctl with no command to open the dashboard.

Common flow:
  agentctl doctor
  agentctl repo scan ~/Src --register
  agentctl plan "Implement a small safe change" --repo backend --repo frontend
  agentctl dashboard

Useful direct commands:
  agentctl status
  agentctl approvals
  agentctl blocked
  agentctl inspect <task-id>
  agentctl logs <task-id> --agent <agent-name>
  agentctl open <task-id> --agent <agent-name>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashboard(ctx.store)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx.store = core.NewStore(ctx.root)
			return ctx.store.Init()
		},
	}
	rootCmd.PersistentFlags().StringVar(&ctx.root, "root", core.ConfigFromEnv(), "agentctl state root")

	rootCmd.AddCommand(newInitCommand(ctx))
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newDoctorCommand(ctx))
	rootCmd.AddCommand(newDaemonCommand(ctx))
	rootCmd.AddCommand(newAFKCommand(ctx))
	rootCmd.AddCommand(newRepoCommand(ctx))
	rootCmd.AddCommand(newConfigCommand(ctx))
	rootCmd.AddCommand(newManagerCommand(ctx))
	rootCmd.AddCommand(newPlanCommand(ctx))
	rootCmd.AddCommand(newAttachCommand(ctx))
	rootCmd.AddCommand(newReviewPlanCommand(ctx))
	rootCmd.AddCommand(newApprovePlanCommand(ctx))
	rootCmd.AddCommand(newDispatchCommand(ctx))
	rootCmd.AddCommand(newStatusCommand(ctx))
	rootCmd.AddCommand(newBlockedCommand(ctx))
	rootCmd.AddCommand(newApprovalsCommand(ctx))
	rootCmd.AddCommand(newInspectCommand(ctx))
	rootCmd.AddCommand(newLogsCommand(ctx))
	rootCmd.AddCommand(newDiffCommand(ctx))
	rootCmd.AddCommand(newEventsCommand(ctx))
	rootCmd.AddCommand(newSuperviseCommand(ctx))
	rootCmd.AddCommand(newDashboardCommand(ctx))
	rootCmd.AddCommand(newWebCommand(ctx))
	rootCmd.AddCommand(newOpenCommand(ctx))
	rootCmd.AddCommand(newPRCommand(ctx))
	rootCmd.AddCommand(newDoneCommand(ctx))
	rootCmd.AddCommand(newArchiveCommand(ctx))

	return rootCmd.Execute()
}

func newInitCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize agentctl state",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "initialized agentctl at %s\n", state.Config.Root)
			return nil
		},
	}
}

func newConfigCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage configuration"}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", state.Config.Root)
			fmt.Fprintln(cmd.OutOrStdout(), "roles:")
			for role, harness := range state.Config.Roles {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", role, harness)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "harnesses:")
			for name, harness := range state.Config.Harnesses {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s %s\n", name, harness.Command, strings.Join(harness.Args, " "))
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set-role <role> <harness>",
		Short: "Set harness for a role",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			if _, ok := state.Config.Harnesses[args[1]]; !ok {
				return fmt.Errorf("unknown harness %q", args[1])
			}
			state.Config.Roles[args[0]] = args[1]
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set role %s to harness %s\n", args[0], args[1])
			return nil
		},
	})
	var harnessMode string
	setHarnessCmd := &cobra.Command{
		Use:   "set-harness <name> <command> [args...]",
		Short: "Add or update a harness command",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			if harnessMode == "" {
				harnessMode = "interactive"
			}
			state.Config.Harnesses[args[0]] = core.Harness{Command: args[1], Args: args[2:], Mode: harnessMode}
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set harness %s\n", args[0])
			return nil
		},
	}
	setHarnessCmd.Flags().StringVar(&harnessMode, "mode", "interactive", "harness mode: interactive or prompt_arg")
	cmd.AddCommand(setHarnessCmd)
	return cmd
}
