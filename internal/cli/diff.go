package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newDiffCommand(ctx *appContext) *cobra.Command {
	var repoName string
	var base string
	var stat bool
	cmd := &cobra.Command{
		Use:   "diff <task-id>",
		Short: "Show git diff for a task repo worktree",
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
			repo, err := selectTaskRepo(task.Repos, repoName)
			if err != nil {
				return err
			}
			if base == "" {
				base = repo.Branch + "~0"
			}
			output, err := core.GitDiff(repo.WorktreePath, base, stat)
			if err != nil {
				return err
			}
			if output == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "no diff")
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), output)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoName, "repo", "", "repo name; required when task has multiple repos")
	cmd.Flags().StringVar(&base, "base", "", "base ref to diff against; defaults to current branch base")
	cmd.Flags().BoolVar(&stat, "stat", false, "show diff stat only")
	return cmd
}

func selectTaskRepo(repos []core.TaskRepo, name string) (core.TaskRepo, error) {
	if len(repos) == 0 {
		return core.TaskRepo{}, fmt.Errorf("task has no repos")
	}
	if name == "" {
		if len(repos) > 1 {
			return core.TaskRepo{}, fmt.Errorf("task has multiple repos; pass --repo")
		}
		return repos[0], nil
	}
	for _, repo := range repos {
		if repo.Name == name {
			return repo, nil
		}
	}
	return core.TaskRepo{}, fmt.Errorf("repo %q not found", name)
}
