# agentctl

Agent Mission Control for multi-repo coding-agent workflows.

`agentctl` is a local command/control layer for planning, dispatching, supervising, and reviewing coding-agent work across existing local repositories.

## Current MVP

This first build includes:

- Go CLI scaffold.
- Configurable state root via `--root` or `AGENTCTL_ROOT`.
- Repo registry with `repo add`, `repo list`, and `repo scan`.
- Multi-repo task workspaces under `~/.agentctl/tasks` by default.
- Git worktree creation from existing local clones.
- Planning-first task artifacts: `plan.md`, `manager-prompt.md`, `briefs/*`, `decisions.md`, `risk-review.md`.
- Configurable harness roles from day one.
- OpenCode as the default manager/worker/reviewer harness.
- tmux-backed manager and worker sessions.
- Bubble Tea TUI dashboard.
- GitHub PR creation command using `gh`.

SQLite, daemonized supervision, richer approvals, and remote monitoring are specified in `ARCHITECTURE.md` and will come after the local workflow is solid.

## Requirements

- Go 1.26+
- git
- tmux
- GitHub CLI (`gh`) for PR creation
- At least one coding-agent harness, defaulting to `opencode`

## Build

```sh
go test ./...
go build -o agentctl .
```

## Basic Usage

Initialize state:

```sh
./agentctl init
```

Register existing repos:

```sh
./agentctl repo add backend ~/Src/company/backend
./agentctl repo add frontend ~/Src/company/frontend
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

Configure harness roles:

```sh
./agentctl config set-harness opencode opencode
./agentctl config set-role manager opencode
./agentctl config set-role worker opencode
./agentctl config set-role reviewer opencode
```

Create a planning-first multi-repo task:

```sh
./agentctl plan "Add refresh-token auth flow" \
  --repo backend \
  --repo frontend
```

Dispatch repo workers after the manager has produced/refined briefs:

```sh
./agentctl dispatch <task-id>
```

Open the dashboard:

```sh
./agentctl dashboard
```

Attach to a running agent:

```sh
./agentctl open <task-id> --agent manager-agent
```

Create a PR for a task repo:

```sh
./agentctl pr <task-id> --repo backend
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
