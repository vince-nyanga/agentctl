#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${AGENTCTL_BIN:-$ROOT_DIR/agentctl}"

if [[ ! -x "$BIN" ]]; then
  echo "agentctl binary not found at $BIN" >&2
  echo "run: go build -o agentctl ." >&2
  exit 1
fi

TMP="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP"
}
trap cleanup EXIT

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "expected output to contain: $needle" >&2
    echo "actual output:" >&2
    printf '%s\n' "$haystack" >&2
    exit 1
  fi
}

STATE_ROOT="$TMP/state"
REPO="$TMP/repo"

git init -b main "$REPO" >/dev/null
git -C "$REPO" config user.name "agentctl smoke"
git -C "$REPO" config user.email "agentctl-smoke@example.com"
cat >"$REPO/README.md" <<'EOF'
# Smoke Repo
EOF
git -C "$REPO" add README.md
git -C "$REPO" commit -m "Initial commit" >/dev/null

"$BIN" --root "$STATE_ROOT" init >/dev/null
"$BIN" --root "$STATE_ROOT" doctor >/dev/null
"$BIN" --root "$STATE_ROOT" repo add smoke "$REPO" >/dev/null

PLAN_OUTPUT="$TMP/plan.out"
"$BIN" --root "$STATE_ROOT" plan "Smoke test task lifecycle" --repo smoke --start-manager=false >"$PLAN_OUTPUT"
TASK_ID="$(awk '/created task/ {print $3}' "$PLAN_OUTPUT")"
if [[ -z "$TASK_ID" ]]; then
  echo "failed to parse task id" >&2
  cat "$PLAN_OUTPUT" >&2
  exit 1
fi

WORKTREE="$STATE_ROOT/tasks/$TASK_ID/worktrees/smoke"
test -d "$WORKTREE"
test -f "$STATE_ROOT/tasks/$TASK_ID/plan.md"
test -f "$STATE_ROOT/tasks/$TASK_ID/briefs/smoke.md"

assert_contains "$("$BIN" --root "$STATE_ROOT" status)" "$TASK_ID"
assert_contains "$("$BIN" --root "$STATE_ROOT" review-plan "$TASK_ID")" "Smoke test task lifecycle"
assert_contains "$("$BIN" --root "$STATE_ROOT" approvals)" "$TASK_ID"
assert_contains "$("$BIN" --root "$STATE_ROOT" blocked)" "plan needs review/approval"
assert_contains "$("$BIN" --root "$STATE_ROOT" inspect "$TASK_ID")" "status: clean"
assert_contains "$("$BIN" --root "$STATE_ROOT" events --task "$TASK_ID")" "task.created"

"$BIN" --root "$STATE_ROOT" approve-plan "$TASK_ID" >/dev/null
assert_contains "$("$BIN" --root "$STATE_ROOT" approvals)" "no pending approvals"
"$BIN" --root "$STATE_ROOT" daemon --once >/dev/null
assert_contains "$("$BIN" --root "$STATE_ROOT" manager tick "$TASK_ID")" "Classify each worker"
assert_contains "$("$BIN" --root "$STATE_ROOT" diff "$TASK_ID" --repo smoke --stat)" "no diff"
"$BIN" --root "$STATE_ROOT" done "$TASK_ID" >/dev/null
"$BIN" --root "$STATE_ROOT" archive "$TASK_ID" >/dev/null

if [[ -d "$WORKTREE" ]]; then
  echo "worktree was not removed by archive" >&2
  exit 1
fi

assert_contains "$(git -C "$REPO" worktree list)" "$REPO"

TMUX_ROOT="$TMP/tmux-state"
"$BIN" --root "$TMUX_ROOT" init >/dev/null
"$BIN" --root "$TMUX_ROOT" config set-harness cat cat >/dev/null
"$BIN" --root "$TMUX_ROOT" config set-role manager cat >/dev/null
"$BIN" --root "$TMUX_ROOT" config set-role worker cat >/dev/null
"$BIN" --root "$TMUX_ROOT" repo add smoke "$REPO" >/dev/null

TMUX_PLAN_OUTPUT="$TMP/tmux-plan.out"
"$BIN" --root "$TMUX_ROOT" plan "Smoke test tmux manager" --repo smoke >"$TMUX_PLAN_OUTPUT"
TMUX_TASK_ID="$(awk '/created task/ {print $3}' "$TMUX_PLAN_OUTPUT")"
if [[ -z "$TMUX_TASK_ID" ]]; then
  echo "failed to parse tmux task id" >&2
  cat "$TMUX_PLAN_OUTPUT" >&2
  exit 1
fi

assert_contains "$("$BIN" --root "$TMUX_ROOT" inspect "$TMUX_TASK_ID")" "manager-agent"
assert_contains "$("$BIN" --root "$TMUX_ROOT" logs "$TMUX_TASK_ID" --agent manager-agent --lines 120)" "You are the manager agent"
"$BIN" --root "$TMUX_ROOT" supervise "$TMUX_TASK_ID" >/dev/null
"$BIN" --root "$TMUX_ROOT" manager tick "$TMUX_TASK_ID" --send >/dev/null
assert_contains "$("$BIN" --root "$TMUX_ROOT" events --task "$TMUX_TASK_ID")" "manager.tick_sent"
"$BIN" --root "$TMUX_ROOT" archive "$TMUX_TASK_ID" >/dev/null
assert_contains "$("$BIN" --root "$TMUX_ROOT" inspect "$TMUX_TASK_ID")" "State: archived"

echo "smoke test passed: $TASK_ID and $TMUX_TASK_ID"
