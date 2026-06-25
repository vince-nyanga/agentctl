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
				task.Repos = append(task.Repos, core.TaskRepo{Name: repoName, SourcePath: repo.Path, WorktreePath: worktreePath, Branch: branch, Owned: true})
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
			if _, err := ctx.store.CreateApproval(core.Approval{TaskID: task.ID, Type: "plan", Title: "Approve task plan", Description: task.Goal, Risk: "medium", RecommendedAction: "review plan and approve when ready"}); err != nil {
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
	var force bool
	cmd := &cobra.Command{
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
			if task.State != "plan_approved" && !force {
				return fmt.Errorf("task %s is %q; approve the plan first with `agentctl approve-plan %s` or rerun dispatch with --force", task.ID, task.State, task.ID)
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
	cmd.Flags().BoolVar(&force, "force", false, "dispatch even when the task plan has not been approved")
	return cmd
}

func newApprovePlanCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "approve-plan <task-id>",
		Short: "Approve a task plan so workers can be dispatched",
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
			if task.State == "running" || task.State == "archived" {
				return fmt.Errorf("cannot approve plan for task in state %q", task.State)
			}
			task.State = "plan_approved"
			task.UpdatedAt = time.Now()
			state.Tasks[task.ID] = task
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "plan.approved", Message: "plan approved for worker dispatch"}); err != nil {
				return err
			}
			approvals, err := ctx.store.ListApprovals(task.ID, "pending")
			if err != nil {
				return err
			}
			for _, approval := range approvals {
				if approval.Type == "plan" {
					if _, err := ctx.store.ResolveApproval(approval.ID, "approved"); err != nil {
						return err
					}
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "approved plan for %s\n", task.ID)
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
	promptText := string(prompt)
	command, promptToSend := buildHarnessCommand(harness, promptText)
	logPath := filepath.Join(task.Workspace, "logs", name+".log")
	if err := core.StartTmuxAgent(tmuxName, workdir, command, promptToSend, logPath); err != nil {
		return core.Agent{}, err
	}
	return core.Agent{Name: name, Role: role, Harness: harnessName, Repo: repo, State: "running", TmuxName: tmuxName, Workdir: workdir, LogPath: logPath, CreatedAt: time.Now()}, nil
}

func buildHarnessCommand(harness core.Harness, prompt string) (string, string) {
	parts := append([]string{harness.Command}, harness.Args...)
	if harness.Mode == "prompt_arg" || (harness.Command == "opencode" && len(harness.Args) > 0 && harness.Args[0] == "run") {
		parts = append(parts, core.ShellQuote(prompt))
		return strings.TrimSpace(strings.Join(parts, " ")), ""
	}
	return strings.TrimSpace(strings.Join(parts, " ")), prompt
}

func prepareWorkerDispatchPrompt(task core.Task, repo core.TaskRepo) (string, error) {
	briefPath := filepath.Join(task.Workspace, "briefs", repo.Name+".md")
	brief, err := os.ReadFile(briefPath)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(task.Workspace, "dispatch-prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	prompt := fmt.Sprintf(`# Approved Implementation Brief

Task: %s
Repo: %s

The user has approved the plan. You must now implement the requested change in this repo worktree.

Important: the original brief below may contain stale text such as "do not implement until approved", "ready for approval", or "await approval". That approval gate is now satisfied. Treat this wrapper as the latest instruction and proceed with implementation.

Rules:
- Work only inside this repo worktree unless explicitly instructed otherwise.
- Keep the implementation minimal and aligned with the approved brief.
- Run the verification commands listed in the brief when possible.
- Do not merge, deploy, or delete unrelated files.

--- Original Brief ---

%s`, task.ID, repo.Name, string(brief))
	out := filepath.Join(dir, repo.Name+".md")
	if err := os.WriteFile(out, []byte(prompt), 0o644); err != nil {
		return "", err
	}
	return out, nil
}
