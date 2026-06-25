# agentctl

Agent Mission Control for multi-repo coding-agent workflows.

`agentctl` is a local command/control layer for planning, dispatching, supervising, and reviewing coding-agent work across existing local repositories.

## Current MVP

This first build includes:

- Go CLI scaffold.
- Configurable state root via `--root` or `AGENTCTL_ROOT`.
- SQLite-backed state in `agentctl.db`.
- Repo registry with `repo add`, `repo list`, and `repo scan`.
- Multi-repo task workspaces under `~/.agentctl/tasks` by default.
- Git worktree creation from existing local clones.
- Planning-first task artifacts: `plan.md`, `manager-prompt.md`, `briefs/*`, `decisions.md`, `risk-review.md`.
- Configurable harness roles from day one.
- OpenCode as the default manager/worker/reviewer harness.
- tmux-backed manager and worker sessions.
- Basic event log and `supervise` reconciliation command.
- Supervision snapshots live tmux output to per-agent log files.
- Foreground daemon/AFK loop uses a root-scoped lock file to prevent duplicate supervisors.
- Bubble Tea TUI dashboard with recent task events.
- GitHub PR creation command using `gh`.
- Task archival cleanup for tmux sessions and git worktrees.

Daemonized supervision, richer approvals, and remote monitoring are specified in `ARCHITECTURE.md` and will come after the local workflow is solid.

## Requirements

- Go 1.26+
- git
- tmux
- GitHub CLI (`gh`) for PR creation
- At least one coding-agent harness, defaulting to `opencode`

## Build

Install from GitHub:

```sh
go install github.com/vince-nyanga/agentctl@latest
```

Install a tagged version:

```sh
go install github.com/vince-nyanga/agentctl@v0.1.0
```

Tagged releases publish macOS/Linux binary archives from GitHub Actions:

```sh
git tag v0.1.0
git push origin v0.1.0
```

Build from a local checkout:

```sh
make check
```

`make check` runs formatting, `go vet`, unit tests, build, and an end-to-end smoke test against a temporary git repo.
The smoke test covers both the repo/worktree lifecycle and a real tmux-backed manager session using a harmless `cat` harness.

Run the opt-in real OpenCode manager/worker E2E test locally when OpenCode is authenticated:

```sh
make real-opencode-e2e
```

Equivalent manual commands:

```sh
gofmt -w .
go test ./...
go build -o agentctl .
```

## Basic Usage

Open the main TUI dashboard:

```sh
agentctl
```

Initialize state:

```sh
./agentctl init
```

Register existing repos:

```sh
./agentctl repo add backend ~/Src/company/backend
./agentctl repo add frontend ~/Src/company/frontend
```

Ask the manager what to do. When `--repo` is omitted, all registered repos are included:

```sh
./agentctl ask "Tell me what this codebase is all about"
./agentctl ask "Implement auth across backend and frontend"
```

Or scan a directory:

```sh
./agentctl repo scan ~/Src
./agentctl repo scan ~/Src --register
```

Show configuration:

```sh
./agentctl config show
```

Check local dependencies and harness availability:

```sh
./agentctl doctor
```

Run the foreground supervision loop:

```sh
./agentctl daemon
./agentctl daemon --once
./agentctl afk
./agentctl afk --manager-tick
./agentctl afk --manager-tick --manager-apply
```

Build or send an AI manager supervision prompt for a task:

```sh
./agentctl manager tick <task-id>
./agentctl manager tick <task-id> --send
./agentctl manager apply <task-id> --file manager-response.md
./agentctl manager apply <task-id> --from-tmux
```

Configure harness roles:

```sh
./agentctl config set-harness opencode opencode
./agentctl config set-harness opencode-run opencode --mode prompt_arg -- run --dangerously-skip-permissions
./agentctl config set-role manager opencode
./agentctl config set-role worker opencode
./agentctl config set-role reviewer opencode
```

Create an explicit planning-first multi-repo task:

```sh
./agentctl plan "Add refresh-token auth flow"

# Or restrict to specific registered repos:
./agentctl plan "Add refresh-token auth flow" \
  --repo backend \
  --repo frontend
```

When `--repo` is omitted, `agentctl` gives the manager access to all registered repos.

Create a task from worktrees you already created manually:

```sh
./agentctl attach "Add refresh-token auth flow" \
  --repo backend=~/Src/worktrees/backend-auth-refresh \
  --repo frontend=~/Src/worktrees/frontend-auth-refresh
```

Attached worktrees are not removed by `archive`; only worktrees created by `agentctl plan` are cleaned up automatically.

Review the plan and optional repo briefs:

```sh
./agentctl review-plan <task-id>
./agentctl review-plan <task-id> --briefs
```

Approve the plan after the manager has produced/refined briefs:

```sh
./agentctl approve-plan <task-id>
```

Then dispatch repo workers:

```sh
./agentctl dispatch <task-id>
```

Bypass the approval gate only when you explicitly want to run before approval:

```sh
./agentctl dispatch <task-id> --force
```

Open the dashboard:

```sh
./agentctl dashboard
./agentctl web
```

Dashboard keys:

```text
h/l       switch tabs
j/k       move task selection
r         refresh state
a         approve selected task plan
d         dispatch selected approved task, asks y/n
x         archive selected task, asks y/n
m         send manager tick to selected task
p         apply manager actions from selected task manager tmux
q         quit
```

Attach to a running agent:

```sh
./agentctl open
./agentctl open <task-id> --agent manager-agent
```

Reconcile tracked tmux sessions and update agent state:

```sh
./agentctl supervise <task-id>
```

Inspect recent events:

```sh
./agentctl events
./agentctl events --task <task-id> --limit 50
```

Inspect one task without opening the TUI:

```sh
./agentctl inspect <task-id>
```

Show tasks that need attention:

```sh
./agentctl blocked
```

Show pending approvals:

```sh
./agentctl approvals
```

Read recent output for a tracked agent:

```sh
./agentctl logs <task-id> --agent manager-agent
```

Review a task repo diff:

```sh
./agentctl diff <task-id> --repo backend
./agentctl diff <task-id> --repo backend --stat
```

Create a PR for a task repo:

```sh
./agentctl pr <task-id> --repo backend
```

PR creation refuses dirty worktrees by default, pushes the task branch to `origin`, then calls `gh pr create`.

Mark a task done without cleaning up its sessions/worktrees:

```sh
./agentctl done <task-id>
```

Archive a completed task and clean up agent sessions/worktrees:

```sh
./agentctl archive <task-id>
```

Archive refuses to remove dirty worktrees by default. Use `--force` only when you intentionally want to discard uncommitted task worktree changes:

```sh
./agentctl archive <task-id> --force
```

Keep worktrees or sessions if you still need to inspect them:

```sh
./agentctl archive <task-id> --keep-worktrees
./agentctl archive <task-id> --keep-sessions
```

## Choosing State Location

Use `--root` for one command:

```sh
./agentctl --root ~/agentctl-state init
```

Or set an environment variable:

```sh
export AGENTCTL_ROOT=~/agentctl-state
```

Default root:

```text
~/.agentctl
```

## Architecture Decisions

`agentctl` is intentionally local-first. The first job is to make local agentic development reliable before adding remote dashboards, notification channels, or cloud coordination.

### Why Go

Go was chosen over Rust, Python, Node.js, and a desktop-first stack for pragmatic reasons:

- Single static-ish binary distribution is straightforward.
- Process orchestration is simple and reliable.
- Filesystem, git, tmux, and long-running daemon work are natural fits.
- The standard library is strong enough for most of the core system.
- The Go codebase should be easier for coding agents to extend safely than an equivalent Rust codebase at this stage.
- Performance is more than enough because this is orchestration-heavy, not CPU-bound.
- The Charmbracelet ecosystem gives Go a strong path for polished terminal UIs.

Rust remains attractive for maximum correctness and systems-level control, but the early product risk is workflow design, not memory safety or raw performance. Go should let the project reach useful daily-driver status faster.

### Why TUI First

The primary workflow is terminal-native. A TUI gives immediate visibility without forcing a browser app too early.

The dashboard is the cockpit:

- Tasks.
- Agents.
- Repos.
- Blockers.
- Approvals.
- Manager feed.
- Logs and diffs.

A web UI is planned later, using the same daemon/state model, once the local core is stable.

### Why SQLite

SQLite is used for durable local state because the tool needs structured queries, recovery after restarts, and a reliable event log without requiring a server.

Markdown files still remain important. Human-readable artifacts such as `plan.md`, `decisions.md`, `risk-review.md`, and repo briefs live in the task workspace so both humans and agents can inspect them directly.

### Why tmux First

tmux is the lowest common denominator for terminal coding agents. Most agent harnesses can run inside a tmux session even if they do not expose an SDK or structured event stream.

The initial harness capability level is:

- Start a terminal session.
- Send a prompt.
- Capture output.
- Attach manually.
- Stop the session.

Richer adapters can be added later for harnesses that expose JSONL logs, SDKs, HTTP APIs, or MCP interfaces.

### Why Harness-Agnostic Roles

The core should not know about Claude Code, OpenCode, Pi, or Codex directly. It knows about roles:

- `manager`
- `worker`
- `reviewer`

Each role maps to a configurable harness. That allows workflows like:

```text
manager: Claude Code
backend worker: OpenCode
frontend worker: Pi
reviewer: Codex
```

OpenCode is the default first harness because it is the immediate target, but the architecture must remain agent-agnostic.

### Why Task Workspaces

The unit of work is a task, not a repo. A single task may require backend, frontend, mobile, infra, and documentation changes.

Task workspaces give each task isolated git worktrees created from existing local clones:

```text
~/.agentctl/tasks/<task-id>/
├── plan.md
├── briefs/
├── worktrees/
│   ├── backend/
│   └── frontend/
└── logs/
```

This keeps the user's normal checkouts clean while letting agents coordinate multi-repo work.

### Why Cleanup Is First-Class

Agentic workflows create many branches, worktrees, sessions, and logs. If cleanup is not built in, the tool becomes another source of clutter.

`agentctl archive <task-id>` stops tracked tmux sessions, removes agent-owned worktrees, prunes worktree metadata, and marks the task archived while preserving planning/report artifacts for later inspection.

### Remote Monitoring Later

Remote monitoring is planned, but not in the first core. It should be built after the local state model and supervision loop are trustworthy.

The planned progression is:

1. Local TUI.
2. Local web dashboard bound to `127.0.0.1`.
3. Authenticated LAN mode.
4. Secure tunnel or notification channel.
5. Remote read-only and approval-safe modes.

Remote access should not expose raw shell control by default.
