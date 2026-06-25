package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newAttachCommand(ctx *appContext) *cobra.Command {
	var repoSpecs []string
	var startManager bool
	cmd := &cobra.Command{
		Use:   "attach <goal>",
		Short: "Create a task from existing repo/worktree paths",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(repoSpecs) == 0 {
				return fmt.Errorf("at least one --repo name=path is required")
			}
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}

			now := time.Now()
			taskID := core.NewTaskID(args[0])
			workspace := filepath.Join(core.TasksDir(ctx.store.Root()), taskID)
			task := core.Task{ID: taskID, Goal: args[0], State: "planning", Workspace: workspace, CreatedAt: now, UpdatedAt: now}

			for _, spec := range repoSpecs {
				name, path, err := parseRepoPathSpec(spec)
				if err != nil {
					return err
				}
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				if !core.IsGitRepo(absPath) {
					return fmt.Errorf("%s is not a git repo", absPath)
				}
				topLevel, err := core.GitTopLevel(absPath)
				if err != nil || topLevel == "" {
					topLevel = absPath
				}
				task.Repos = append(task.Repos, core.TaskRepo{Name: name, SourcePath: topLevel, WorktreePath: topLevel, Branch: core.CurrentBranch(topLevel), Owned: false})
			}

			if err := core.CreateTaskWorkspace(ctx.store.Root(), task); err != nil {
				return err
			}
			if err := core.WritePlanArtifacts(task); err != nil {
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
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.attached", Message: "created planning workspace from existing worktrees"}); err != nil {
				return err
			}
			if _, err := ctx.store.CreateApproval(core.Approval{TaskID: task.ID, Type: "plan", Title: "Approve task plan", Description: task.Goal, Risk: "medium", RecommendedAction: "review plan and approve when ready"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created attached task %s at %s\n", task.ID, task.Workspace)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&repoSpecs, "repo", nil, "existing repo/worktree as name=path")
	cmd.Flags().BoolVar(&startManager, "start-manager", true, "start manager harness in tmux")
	return cmd
}

func parseRepoPathSpec(spec string) (string, string, error) {
	name, path, ok := strings.Cut(spec, "=")
	if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(path) == "" {
		return "", "", fmt.Errorf("repo spec must be name=path: %q", spec)
	}
	return strings.TrimSpace(name), strings.TrimSpace(path), nil
}
