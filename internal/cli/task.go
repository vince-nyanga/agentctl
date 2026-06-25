package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newPlanCommand(ctx *appContext) *cobra.Command {
	var repoNames []string
	var startManager bool
	cmd := &cobra.Command{
		Use:   "plan <goal>",
		Short: "Create a planning-first task workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			if len(repoNames) == 0 {
				return fmt.Errorf("at least one --repo is required")
			}

			now := time.Now()
			taskID := core.NewTaskID(args[0])
			workspace := filepath.Join(core.TasksDir(ctx.store.Root()), taskID)
			task := core.Task{ID: taskID, Goal: args[0], State: "planning", Workspace: workspace, CreatedAt: now, UpdatedAt: now}

			for _, repoName := range repoNames {
				repo, ok := state.Repos[repoName]
				if !ok {
					return fmt.Errorf("unknown repo %q", repoName)
				}
				branch := "agent/" + taskID
				worktreePath := filepath.Join(workspace, "worktrees", repoName)
				if err := core.CreateWorktree(repo.Path, branch, worktreePath); err != nil {
					return err
				}
				task.Repos = append(task.Repos, core.TaskRepo{Name: repoName, SourcePath: repo.Path, WorktreePath: worktreePath, Branch: branch})
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
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.created", Message: "created planning workspace"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created task %s at %s\n", task.ID, task.Workspace)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&repoNames, "repo", nil, "registered repo to include")
	cmd.Flags().BoolVar(&startManager, "start-manager", true, "start manager harness in tmux")
	return cmd
}

func newDispatchCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "dispatch <task-id>",
		Short: "Start worker agents for a planned task",
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
				briefPath := filepath.Join(task.Workspace, "briefs", repo.Name+".md")
				agentName := repo.Name + "-agent"
				agent, err := startAgent(state, task, "worker", agentName, repo.Name, repo.WorktreePath, briefPath)
				if err != nil {
					return err
				}
				task.Agents = append(task.Agents, agent)
			}
			task.State = "running"
			task.UpdatedAt = time.Now()
			state.Tasks[task.ID] = task
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.dispatched", Message: fmt.Sprintf("dispatched %d workers", len(task.Repos))}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "dispatched %d workers for %s\n", len(task.Repos), task.ID)
			return nil
		},
	}
}

func startAgent(state core.State, task core.Task, role, name, repo, workdir, promptPath string) (core.Agent, error) {
	harnessName := state.Config.Roles[role]
	harness, ok := state.Config.Harnesses[harnessName]
	if !ok {
		return core.Agent{}, fmt.Errorf("role %s uses unknown harness %q", role, harnessName)
	}
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		return core.Agent{}, err
	}
	tmuxName := fmt.Sprintf("agentctl-%s-%s", task.ID, name)
	command := strings.TrimSpace(strings.Join(append([]string{harness.Command}, harness.Args...), " "))
	if err := core.StartTmuxAgent(tmuxName, workdir, command, string(prompt)); err != nil {
		return core.Agent{}, err
	}
	return core.Agent{Name: name, Role: role, Harness: harnessName, Repo: repo, State: "running", TmuxName: tmuxName, Workdir: workdir, LogPath: filepath.Join(task.Workspace, "logs", name+".log"), CreatedAt: time.Now()}, nil
}
