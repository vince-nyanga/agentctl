package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newArchiveCommand(ctx *appContext) *cobra.Command {
	var keepWorktrees bool
	var keepSessions bool
	var force bool
	cmd := &cobra.Command{
		Use:   "archive <task-id>",
		Short: "Archive a task and clean up its sessions and worktrees",
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

			if !keepSessions {
				for _, agent := range task.Agents {
					if err := core.KillTmuxSession(agent.TmuxName); err != nil {
						return fmt.Errorf("kill %s: %w", agent.Name, err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "stopped session %s\n", agent.TmuxName)
				}
			}

			if !keepWorktrees {
				seenSources := map[string]bool{}
				for _, repo := range task.Repos {
					if !repo.Owned {
						fmt.Fprintf(cmd.OutOrStdout(), "kept attached worktree %s\n", repo.WorktreePath)
						continue
					}
					status := core.GitStatusShort(repo.WorktreePath)
					if core.IsDirtyStatus(status) && !force {
						return fmt.Errorf("worktree %s has uncommitted changes; inspect it or rerun with --force", repo.WorktreePath)
					}
					if err := core.RemoveWorktree(repo.SourcePath, repo.WorktreePath); err != nil {
						return fmt.Errorf("remove worktree %s: %w", repo.Name, err)
					}
					seenSources[repo.SourcePath] = true
					fmt.Fprintf(cmd.OutOrStdout(), "removed worktree %s\n", repo.WorktreePath)
				}
				for source := range seenSources {
					if err := core.PruneWorktrees(source); err != nil {
						return err
					}
				}
			}

			for i := range task.Agents {
				task.Agents[i].State = "stopped"
			}
			task.State = "archived"
			task.UpdatedAt = time.Now()
			state.Tasks[task.ID] = task
			if err := ctx.store.Save(state); err != nil {
				return err
			}
			if err := ctx.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.archived", Message: "archived task and cleaned up owned resources"}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "archived task %s\n", task.ID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&keepWorktrees, "keep-worktrees", false, "do not remove task worktrees")
	cmd.Flags().BoolVar(&keepSessions, "keep-sessions", false, "do not stop tmux sessions")
	cmd.Flags().BoolVar(&force, "force", false, "remove dirty task worktrees")
	return cmd
}
