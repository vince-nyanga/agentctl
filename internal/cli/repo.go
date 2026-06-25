package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newRepoCommand(ctx *appContext) *cobra.Command {
	cmd := &cobra.Command{Use: "repo", Short: "Manage registered repos"}
	cmd.AddCommand(newRepoAddCommand(ctx))
	cmd.AddCommand(newRepoListCommand(ctx))
	cmd.AddCommand(newRepoScanCommand(ctx))
	return cmd
}

func newRepoAddCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <path>",
		Short: "Register an existing local git repo",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[1])
			if err != nil {
				return err
			}
			if !core.IsGitRepo(path) {
				return fmt.Errorf("%s is not a git repo", path)
			}
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			state.Repos[args[0]] = core.Repo{Name: args[0], Path: path, Remote: core.GitRemote(path), CreatedAt: time.Now()}
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "registered %s -> %s\n", args[0], path)
			return nil
		},
	}
}

func newRepoListCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			if len(state.Repos) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no repos registered")
				return nil
			}
			for name, repo := range state.Repos {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", name, repo.Path, repo.Remote)
			}
			return nil
		},
	}
}

func newRepoScanCommand(ctx *appContext) *cobra.Command {
	var register bool
	cmd := &cobra.Command{
		Use:   "scan <directory>",
		Short: "Scan a directory for git repos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := core.ScanRepos(args[0])
			if err != nil {
				return err
			}
			if !register {
				for _, result := range results {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", result.Name, result.Path, result.Remote)
				}
				return nil
			}
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			for _, result := range results {
				state.Repos[result.Name] = core.Repo{Name: result.Name, Path: result.Path, Remote: result.Remote, CreatedAt: time.Now()}
			}
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "registered %d repos\n", len(results))
			return nil
		},
	}
	cmd.Flags().BoolVar(&register, "register", false, "register discovered repos")
	return cmd
}
