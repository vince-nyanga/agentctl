package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newPRCommand(ctx *appContext) *cobra.Command {
	var repoName string
	var allowDirty bool
	var noPush bool
	cmd := &cobra.Command{
		Use:   "pr <task-id>",
		Short: "Create a GitHub pull request for a task repo",
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
			for _, repo := range task.Repos {
				if repoName == "" || repo.Name == repoName {
					status := core.GitStatusShort(repo.WorktreePath)
					if core.IsDirtyStatus(status) && !allowDirty {
						return fmt.Errorf("worktree %s has uncommitted changes; commit them or rerun with --allow-dirty", repo.WorktreePath)
					}
					if !noPush {
						if err := core.PushBranch(repo.WorktreePath, "origin", core.CurrentBranch(repo.WorktreePath)); err != nil {
							return err
						}
					}
					body := fmt.Sprintf("Task: %s\n\nGoal:\n%s\n", task.ID, task.Goal)
					return core.CreatePullRequest(repo.WorktreePath, task.Goal, body)
				}
			}
			return fmt.Errorf("repo %q not found", repoName)
		},
	}
	cmd.Flags().StringVar(&repoName, "repo", "", "repo to create PR from")
	cmd.Flags().BoolVar(&allowDirty, "allow-dirty", false, "allow PR creation when the worktree has uncommitted changes")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "skip pushing the current branch before creating the PR")
	return cmd
}
