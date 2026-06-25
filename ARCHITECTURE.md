# Agent Mission Control Architecture

## 1. Purpose

Agent Mission Control is a local orchestration system for managing many coding-agent workflows across many existing local repositories.

It combines two capabilities:

1. A manager agent that plans, delegates, supervises, reviews, and reports on work.
2. A control panel that shows all active tasks, agents, repos, blockers, approvals, diffs, logs, and manager activity.

The system is coding-agent agnostic. Claude Code, OpenCode, Pi, Codex, Aider, or any other terminal-based harness should be usable through adapters.

The goal is to let the user manage goals, approvals, decisions, and final review while the system manages plans, documents, worktrees, sessions, routine supervision, and worker coordination.

## 2. Non-Goals

- Do not build a Claude-only tool.
- Do not require repos to be recloned if they already exist locally.
- Do not make the dashboard the source of truth.
- Do not make worker agents share a single dirty checkout.
- Do not auto-merge or deploy without explicit user approval.
- Do not hide terminal access; users must be able to open any live agent session.

## 3. Core Concepts

### 3.1 Source Repo

A source repo is an existing local clone registered with the system.

Example:

```text
~/Src/company/backend
~/Src/company/frontend
```

The source repo is treated as the stable local base. Agent work should normally happen in task worktrees created from this repo, not in the user's primary checkout.

### 3.2 Task Workspace

A task workspace is the unit of work. It can contain one or more repo worktrees.

Example:

```text
~/.agentctl/tasks/auth-refresh-042/
├── task.yaml
├── plan.md
├── manager-log.md
├── decisions.md
├── approvals.jsonl
├── briefs/
│   ├── backend.md
│   └── frontend.md
├── worktrees/
│   ├── backend/
│   └── frontend/
└── logs/
    ├── manager.log
    ├── backend-agent.log
    └── frontend-agent.log
```

This enables a single task to span backend, frontend, mobile, infra, docs, or any other combination of repos.

### 3.3 Manager Agent

The manager agent owns the task lifecycle.

Responsibilities:

- Investigate relevant repos.
- Create plans, notes, contracts, and implementation briefs.
- Ask for plan approval before dispatching workers when required by policy.
- Spawn worker agents.
- Monitor worker output and status.
- Detect blocked, stale, failed, or completed work.
- Nudge workers when recovery is routine.
- Escalate only meaningful decisions to the user.
- Review diffs, test results, logs, and PR readiness.
- Prepare final reports.

The manager is not just a dashboard summarizer. It actively works on behalf of the user.

### 3.4 Worker Agent

A worker agent performs implementation, investigation, testing, review, or QA inside a constrained workspace.

Worker agents may be scoped to:

- One repo.
- Several repos.
- One brief.
- One test/review task.

### 3.5 Control Panel

The control panel is the user's cockpit.

It shows:

- All tasks.
- All agents.
- All repos/worktrees.
- Current manager activity.
- Blockers.
- Approval requests.
- Recent events.
- Logs.
- Diffs.
- PR status.

The control panel reads from the same state store as the manager. It should not maintain separate state.

### 3.6 Harness Adapter

A harness adapter integrates a coding-agent runtime.

Examples:

- Claude Code.
- OpenCode.
- Pi.
- Codex.
- Aider.
- Future SDK/API-based agents.

The core system depends on a stable adapter interface, not vendor-specific behavior.

## 4. High-Level Architecture

```text
                        ┌────────────────────┐
                        │        User        │
                        └─────────┬──────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │ CLI / TUI / Future Web UI │
                    └─────────────┬─────────────┘
                                  │
                        ┌─────────┴──────────┐
                        │ agentctl daemon    │
                        └─────────┬──────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              │                   │                   │
       ┌──────▼──────┐    ┌───────▼───────┐    ┌──────▼──────┐
       │ SQLite DB   │    │ Task files    │    │ Event log   │
       └──────┬──────┘    └───────┬───────┘    └──────┬──────┘
              │                   │                   │
              └───────────────────┼───────────────────┘
                                  │
                      ┌───────────▼───────────┐
                      │ Manager agent loop    │
                      └───────────┬───────────┘
                                  │
       ┌──────────────────────────┼──────────────────────────┐
       │                          │                          │
┌──────▼──────┐            ┌──────▼──────┐            ┌──────▼──────┐
│ Worker A    │            │ Worker B    │            │ Reviewer    │
│ tmux/session│            │ tmux/session│            │ tmux/session│
└──────┬──────┘            └──────┬──────┘            └──────┬──────┘
       │                          │                          │
┌──────▼──────┐            ┌──────▼──────┐            ┌──────▼──────┐
│ backend wt  │            │ frontend wt │            │ task wt     │
└─────────────┘            └─────────────┘            └─────────────┘
```

## 5. Technology Choices

### 5.1 Language

Use Go.

Reasons:

- Single binary distribution.
- Strong process management.
- Good long-running daemon support.
- Excellent for filesystem, git, tmux, and log orchestration.
- Mature SQLite support.
- Strong TUI ecosystem through Charmbracelet.
- Easier for coding agents to modify quickly than Rust in early product stages.

### 5.2 Core Libraries

Recommended initial stack:

```text
CLI:        cobra
TUI:        bubbletea, bubbles, lipgloss
Forms:      huh
Markdown:   glamour
Storage:    sqlite
Config:     yaml or toml
Logs:       slog + JSONL event files
Git:        shell out to git initially
Tmux:       shell out to tmux initially
```

Avoid building a web UI first. Build the reliable local core and TUI first. Add a web UI later over the same daemon API and database.

## 6. State and Storage

### 6.1 State Store

SQLite is the authoritative structured store.

Suggested tables:

```text
repos
tasks
task_repos
agents
sessions
approvals
events
artifacts
pr_links
settings
```

### 6.2 File Store

Markdown and JSONL files are used for artifacts that humans and agents need to read directly.

Examples:

- `plan.md`
- `manager-log.md`
- `decisions.md`
- `risk-review.md`
- `briefs/backend.md`
- `briefs/frontend.md`
- `logs/backend-agent.jsonl`
- `approvals.jsonl`

SQLite stores metadata and indexes. Files store durable human-readable context.

### 6.3 Event Log

All major actions produce events.

Examples:

```text
task.created
plan.started
plan.ready
plan.approved
agent.spawned
agent.output_observed
agent.blocked
approval.requested
approval.resolved
worker.completed
manager.review_started
task.ready_for_user_review
task.archived
```

Events support the dashboard, manager reasoning, audit trails, and recovery after restarts.

## 7. Task Lifecycle

Tasks move through a state machine.

```text
created
↓
planning
↓
awaiting_plan_approval
↓
dispatching
↓
running
↓       ↘
blocked  reviewing
↓          ↓
running    ready_for_user_review
           ↓
           done
           ↓
           archived
```

Failed tasks may enter:

```text
failed
cancelled
paused
```

## 8. Manager Agent Loop

The manager loop is the core of the product.

It repeatedly performs:

```text
observe → classify → decide → act → record → report if necessary
```

### 8.1 Observe

The daemon gathers:

- tmux pane output tails.
- agent logs.
- git status.
- git diffs.
- test output summaries.
- PR/CI status when available.
- approval queue state.
- last activity timestamps.

### 8.2 Classify

The manager or daemon classifies each session:

```text
idle
running
waiting_for_input
blocked
stale
failed
completed
needs_review
unknown
```

### 8.3 Decide

The manager decides whether to:

- Do nothing.
- Nudge a worker.
- Send clarifying context.
- Create an approval request.
- Spawn a helper/reviewer.
- Pause a task.
- Mark task ready.
- Escalate to the user.

### 8.4 Act

Actions are executed through the daemon, not directly through hidden magic.

Examples:

- `tmux send-keys` to a session.
- Spawn a new worker.
- Update task state.
- Create approval.
- Run a safe git command.
- Open a draft PR if policy permits.

### 8.5 Report

The manager reports to the user only when:

- Work is blocked on a meaningful decision.
- A risk or plan change appears.
- An approval is required.
- Work is done and ready for review.
- Repeated recovery attempts failed.

Routine status updates stay in the dashboard and event log.

## 9. Planning Workflow

The manager should plan before dispatching workers.

Command:

```sh
agentctl plan "Add refresh-token auth across backend and frontend" \
  --repo backend \
  --repo frontend
```

Manager outputs:

```text
plan.md
architecture-notes.md
api-contract.md
risk-review.md
decisions.md
briefs/backend.md
briefs/frontend.md
briefs/integration.md
```

User approval:

```sh
agentctl approve-plan auth-refresh-042
```

Dispatch:

```sh
agentctl dispatch auth-refresh-042
```

One-shot mode:

```sh
agentctl run "Add refresh-token auth across backend and frontend" \
  --repo backend \
  --repo frontend \
  --plan-first
```

## 10. Multi-Repo Workflow

The system must support tasks spanning multiple local repos.

Example:

```sh
agentctl task start "Add refresh-token auth flow" \
  --repo backend \
  --repo frontend
```

Generated workspace:

```text
~/.agentctl/tasks/auth-refresh-042/worktrees/backend
~/.agentctl/tasks/auth-refresh-042/worktrees/frontend
```

Worker strategies:

1. Single worker with access to all task worktrees.
2. One worker per repo.
3. Lead manager plus repo workers plus integration reviewer.

Default strategy should be manager plus repo workers for multi-repo implementation tasks.

## 11. Harness-Agnostic Agent Interface

The core system should not know Claude-specific or OpenCode-specific behavior.

Define an internal interface similar to:

```go
type Harness interface {
    Name() string
    Start(ctx context.Context, spec SessionSpec) (*Session, error)
    Send(ctx context.Context, session SessionRef, message string) error
    Interrupt(ctx context.Context, session SessionRef) error
    Tail(ctx context.Context, session SessionRef, lines int) (string, error)
    Classify(ctx context.Context, output string) (SessionState, error)
    Stop(ctx context.Context, session SessionRef) error
}
```

Initial adapters can be terminal/tmux based.

Later adapters can use:

- JSONL logs.
- SDKs.
- HTTP APIs.
- MCP tools.

### 11.1 Harness Capability Levels

Level 1: Universal terminal harness.

```text
Can start in tmux.
Can receive text.
Can produce terminal output.
Can be stopped.
```

Level 2: Observable harness.

```text
Provides logs, JSON events, or session files.
Can classify tool calls and approval prompts more accurately.
```

Level 3: Controllable harness.

```text
Has API or SDK.
Can stream structured events.
Can approve/deny programmatically.
Can resume sessions reliably.
```

The product must work with Level 1 and improve with Level 2/3.

## 12. Harness Configuration

Example config:

```yaml
harnesses:
  claude:
    command: claude
    mode: interactive
    transport: tmux
    detect:
      busy:
        - "Working"
        - "esc to interrupt"
      approval:
        - "Do you want to"
        - "Allow this command"
    supports:
      resume: true
      json_events: false

  opencode:
    command: opencode
    mode: interactive
    transport: tmux
    detect:
      busy:
        - "thinking"
        - "working"
      approval:
        - "allow"
        - "permission"
    supports:
      resume: true
      json_events: true

roles:
  manager:
    harness: claude
  worker:
    harness: opencode
  reviewer:
    harness: codex
```

## 13. Control Panel

### 13.1 TUI First

Command:

```sh
agentctl dashboard
```

Initial dashboard panes:

```text
Overview
Tasks
Agents
Blocked
Approvals
Repos
Logs
Manager Feed
```

Example layout:

```text
┌ Agent Mission Control ───────────────────────────────────────────┐
│ Overview  Tasks  Agents  Blocked  Approvals  Repos  Logs        │
├──────────────────────┬───────────────────────────────────────────┤
│ auth-refresh-042     │ State: BLOCKED                            │
│ billing-webhook-017  │ Repos: backend, frontend                  │
│ dashboard-redesign   │ Manager: reviewing API contract           │
│ mobile-login-009     │ Workers: backend done, frontend blocked   │
├──────────────────────┴───────────────────────────────────────────┤
│ Approval: Add expiresAt to refresh-token response?               │
│ [a] approve [d] deny [m] modify [o] open tmux [v] diff [l] logs  │
└──────────────────────────────────────────────────────────────────┘
```

### 13.2 Future Web UI

The web UI should be added after the daemon/state model is stable.

It should use the same daemon API and SQLite state.

Potential stack:

- Go HTTP server + HTMX for a lightweight local dashboard.
- Or Go API + React if richer interaction is needed.

### 13.3 Remote Monitoring

Remote monitoring is a post-MVP feature, but the architecture should leave room for it from the beginning.

The goal is to let the user check task status, blockers, approvals, logs, and manager summaries away from the local terminal.

Remote monitoring should start as read-mostly and approval-safe, not full remote shell control.

Recommended phases:

1. Local web dashboard bound to `127.0.0.1`.
2. Optional authenticated LAN access.
3. Optional secure tunnel for remote access.
4. Optional mobile/chat notifications for blockers and completion.
5. Optional remote approvals for low/medium-risk actions.

Remote access modes:

```text
read_only
  Can view tasks, status, logs, manager summaries, and PR links.

approve_safe
  Can approve predefined low/medium-risk approval types.
  Cannot approve destructive actions, deploys, merges, or force pushes.

operator
  Can pause/resume/nudge/send instructions.
  Requires stronger authentication and explicit local opt-in.
```

Remote monitoring should never expose raw shell access by default. Opening an interactive tmux pane should remain local-only until a deliberate remote-operator mode is designed.

Possible delivery options:

```text
agentctl web
  Starts local web dashboard.

agentctl web --lan
  Binds to LAN with authentication.

agentctl tunnel
  Starts a secure tunnel using a configured provider.

agentctl notify setup
  Configures Telegram, Discord, Slack, email, or webhook notifications.
```

Notification events:

```text
task.blocked
approval.requested
task.ready_for_review
task.failed
manager.needs_decision
```

Security requirements for remote monitoring:

- Authentication required for any non-local access.
- CSRF protection for browser actions.
- Short-lived approval tokens for notification links.
- Audit log for every remote action.
- Risk-tier enforcement so dangerous approvals require local confirmation.
- Redaction of secrets and sensitive file contents from remote logs.
- Option to run remote dashboard as read-only.

## 14. Approval System

Approvals are first-class state, not terminal-only prompts.

Approval record fields:

```text
id
task_id
agent_id
type
title
description
risk
recommended_action
options
created_at
resolved_at
resolution
```

Common approval types:

```text
plan_approval
api_contract_change
database_migration
dependency_install
external_system_write
push_branch
open_pr
merge
deploy
destructive_command
```

Policy modes:

```text
cautious
normal
yolo
```

Initial default: `normal`.

Normal mode:

- Plan approval required.
- Routine edits/tests allowed.
- Ask for API/product/security decisions.
- PR creation allowed if configured.
- Merge/deploy always require approval.

## 15. Command Surface

### 15.1 Setup

```sh
agentctl repo scan ~/Src
agentctl repo add backend ~/Src/company/backend
agentctl repo add frontend ~/Src/company/frontend
agentctl repo list
agentctl config set roles.manager claude
agentctl config set roles.worker opencode
agentctl config set roles.reviewer codex
```

### 15.2 Planning

```sh
agentctl plan "Add refresh-token auth flow" --repo backend --repo frontend
agentctl review-plan auth-refresh-042
agentctl approve-plan auth-refresh-042
```

### 15.3 Dispatch and Supervision

```sh
agentctl dispatch auth-refresh-042
agentctl supervise auth-refresh-042
agentctl afk
```

### 15.4 Dashboard and Status

```sh
agentctl dashboard
agentctl status
agentctl blocked
agentctl approvals
agentctl events
```

### 15.5 Manager Interaction

```sh
agentctl ask "What needs my attention?"
agentctl ask "Summarize auth-refresh-042"
agentctl ask "Pause all frontend work until I review auth-refresh"
```

### 15.6 Inspection

```sh
agentctl inspect auth-refresh-042
agentctl logs auth-refresh-042
agentctl diff auth-refresh-042 --repo backend
agentctl open auth-refresh-042 --agent frontend-agent
```

### 15.7 Completion

```sh
agentctl review auth-refresh-042
agentctl pr auth-refresh-042
agentctl archive auth-refresh-042
```

## 16. Daemon Responsibilities

The daemon coordinates long-running work.

Responsibilities:

- Manage task lifecycle.
- Own SQLite connections.
- Spawn and monitor tmux sessions.
- Track process/session liveness.
- Periodically poll git status and logs.
- Maintain event log.
- Run manager supervision loops.
- Serve local API for TUI/web.
- Recover state after restart.

The CLI should be a client of the daemon when the daemon is running, but simple commands may operate directly for early MVP.

## 17. Tmux Model

Initial implementation should use tmux because it works with most terminal coding agents.

Naming convention:

```text
agentctl:<task-id>:manager
agentctl:<task-id>:backend-agent
agentctl:<task-id>:frontend-agent
```

Each session/pane should have metadata in SQLite linking it to:

- Task.
- Agent role.
- Harness.
- Working directory.
- Log file.
- Current state.

Users must be able to attach:

```sh
agentctl open auth-refresh-042 --agent backend-agent
```

## 18. Git and Worktree Model

For each registered repo in a task:

1. Verify source repo is clean enough or warn.
2. Fetch default branch when policy allows.
3. Create task branch.
4. Create worktree under the task workspace.
5. Record source repo, branch, and worktree path.

Example branch name:

```text
agent/auth-refresh-042
```

The system should also support attaching existing manual worktrees:

```sh
agentctl task attach auth-refresh-042 \
  --repo backend=~/Src/worktrees/backend-auth-refresh \
  --repo frontend=~/Src/worktrees/frontend-auth-refresh
```

## 19. Safety Model

Safety is policy-based.

Always require explicit user approval for:

- Merge.
- Deploy.
- Force push.
- Destructive git operations.
- Database destructive operations.
- Deleting files outside task workspace.
- Writing to production/external systems.
- Accessing secrets beyond configured policy.

Prefer sandboxing where available, but do not make Docker mandatory for MVP.

Future sandbox options:

- Yolobox integration.
- Docker container per task.
- macOS sandbox/seatbelt.
- VM isolation for risky work.

## 20. MVP Scope

The first useful version should include:

1. Repo registry.
2. Task workspace creation.
3. Git worktree creation from existing local clones.
4. Harness config for at least one terminal harness.
5. tmux session spawning.
6. Manager planning command.
7. Plan approval.
8. Worker dispatch.
9. Basic event log.
10. Basic status command.
11. Basic TUI dashboard.
12. Open tmux session from dashboard/CLI.
13. Manual approval queue.

## 21. Implementation Phases

### Phase 1: Local Core

- Build Go CLI skeleton.
- Add config file handling.
- Add SQLite store.
- Add repo registry and scan.
- Add task workspace creation.
- Add worktree creation.

### Phase 2: Sessions

- Add tmux session manager.
- Add harness abstraction.
- Add first harness adapter.
- Add logs and event capture.
- Add `agentctl open`.

### Phase 3: Manager Planning

- Add `agentctl plan`.
- Manager creates plan/docs/briefs.
- Add `review-plan` and `approve-plan`.
- Store plan state.

### Phase 4: Dispatch and Supervision

- Add worker dispatch from briefs.
- Add session state classification.
- Add stale/blocked detection.
- Add manager supervision loop.
- Add approval queue.

### Phase 5: TUI Cockpit

- Add dashboard overview.
- Add tasks list.
- Add blocked/approval views.
- Add logs/diff/open actions.
- Add manager feed.

### Phase 6: Review and PR Flow

- Add diff summarization.
- Add review agent role.
- Add test result collection.
- Add PR preparation.
- Add archive/cleanup.

### Phase 7: Richer Integrations

- Add more harness adapters.
- Add structured event adapters where available.
- Add web UI.
- Add notification channels.
- Add sandbox integrations.

## 22. Open Questions

1. What should the tool be called: `agentctl`, `mission`, `conductor`, or something else?
2. Which harness should be implemented first?
3. Should manager planning use the same harness as workers by default?
4. Should tasks live under `~/.agentctl/tasks` or inside a configurable workspace root?
5. Should PR creation be enabled in the MVP or deferred?
6. Should the first dashboard be TUI only, or TUI plus minimal web status page?
7. What should the default policy mode be for the user's daily workflow?

## 23. Recommended Initial Defaults

```text
Language: Go
Storage: SQLite + markdown artifacts
Interface: CLI + TUI
Session runtime: tmux
Task root: ~/.agentctl/tasks
Repo handling: existing local clones + git worktrees
Policy: normal
Manager role: configurable harness
Worker role: configurable harness
Review role: configurable harness
Web UI: later
```

## 24. Success Criteria

The product is useful when the user can:

1. Register existing local repos.
2. Start a multi-repo task from one command.
3. Get a manager-generated plan before coding begins.
4. Approve the plan.
5. Let the manager dispatch workers.
6. Walk away without watching terminals.
7. Return to a dashboard showing the true state of all tasks.
8. See only meaningful blockers and approvals.
9. Inspect logs, diffs, and sessions when desired.
10. Receive a final report when work is ready.
