package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newAskCommand(ctx *appContext) *cobra.Command {
	var repoNames []string
	var startManager bool
	cmd := &cobra.Command{
		Use:   "ask <message>",
		Short: "Ask the manager agent what to do across registered repos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			selectedRepos, err := resolvePlanRepos(state, repoNames)
			if err != nil {
				return err
			}
			now := time.Now()
			taskID := core.NewTaskID(args[0])
			workspace := filepath.Join(core.TasksDir(ctx.store.Root()), taskID)
			task := core.Task{ID: taskID, Goal: args[0], State: "manager_review", Workspace: workspace, CreatedAt: now, UpdatedAt: now}
			for _, repoName := range selectedRepos {
				repo, ok := state.Repos[repoName]
				if !ok {
					return fmt.Errorf("unknown repo %q", repoName)
				}
				branch := "agent/" + taskID
				worktreePath := filepath.Join(workspace, "worktrees", repoName)
				if err := core.CreateWorktree(repo.Path, branch, worktreePath); err != nil {
					return err
				}
				task.Repos = append(task.Repos, core.TaskRepo{Name: repoName, SourcePath: repo.Path, WorktreePath: worktreePath, Branch: branch, Owned: true})
			}
			if err := core.CreateTaskWorkspace(ctx.store.Root(), task); err != nil {
				return err
			}
			if err := core.WriteAskArtifacts(task); err != nil {
				return err
			}
			if startManager {
				agent, err := startAgent(state, task, "manager", "manager-agent", "", workspace, filepath.Join(workspace, "manager-prompt.md"))
				if err != nil {
					return err
				}
				task.Agents = append(task.Agents, agent)
			}
			state.Tasks[task.ID] = task
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.asked", Message: "created manager-led request"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created manager request %s at %s\n", task.ID, task.Workspace)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&repoNames, "repo", nil, "optional registered repo to include; defaults to all registered repos")
	cmd.Flags().BoolVar(&startManager, "start-manager", true, "start manager harness in tmux")
	return cmd
}
