#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${AGENTCTL_BIN:-$ROOT_DIR/agentctl}"

if [[ ! -x "$BIN" ]]; then
  echo "agentctl binary not found at $BIN" >&2
  echo "run: go build -o agentctl ." >&2
  exit 1
fi

if ! command -v opencode >/dev/null 2>&1; then
  echo "opencode is required for this opt-in E2E test" >&2
  exit 1
fi

TMP="$(mktemp -d)"
cleanup() {
  set +e
  if [[ -n "${TASK_ID:-}" && -n "${STATE:-}" ]]; then
    "$BIN" --root "$STATE" archive "$TASK_ID" --force >/dev/null 2>&1 || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT

STATE="$TMP/state"
REPO="$TMP/repo"

mkdir -p "$REPO"
git init -b main "$REPO" >/dev/null
git -C "$REPO" config user.name "agentctl manager e2e"
git -C "$REPO" config user.email "agentctl-manager-e2e@example.com"
cat >"$REPO/go.mod" <<'EOF'
module example.com/hello

go 1.26.4
EOF
cat >"$REPO/main.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("hello")
}
EOF
git -C "$REPO" add .
git -C "$REPO" commit -m "Initial app" >/dev/null

"$BIN" --root "$STATE" init >/dev/null
"$BIN" --root "$STATE" config set-harness opencode-run opencode --mode prompt_arg -- run --dangerously-skip-permissions >/dev/null
"$BIN" --root "$STATE" config set-role manager opencode-run >/dev/null
"$BIN" --root "$STATE" config set-role worker opencode-run >/dev/null
"$BIN" --root "$STATE" repo add app "$REPO" >/dev/null

PLAN_OUTPUT="$TMP/plan.out"
"$BIN" --root "$STATE" plan "Add a --name flag to the Go CLI so it prints hello, <name>; default remains hello" --repo app >"$PLAN_OUTPUT"
TASK_ID="$(awk '/created task/ {print $3}' "$PLAN_OUTPUT")"
if [[ -z "$TASK_ID" ]]; then
  echo "failed to parse task id" >&2
  cat "$PLAN_OUTPUT" >&2
  exit 1
fi

for _ in 1 2 3 4 5 6; do
  sleep 15
  "$BIN" --root "$STATE" supervise "$TASK_ID" >/dev/null
  if ! tmux has-session -t "agentctl-$TASK_ID-manager-agent" 2>/dev/null; then
    break
  fi
done

"$BIN" --root "$STATE" review-plan "$TASK_ID" --briefs | grep -qi "ready for approval"
"$BIN" --root "$STATE" approve-plan "$TASK_ID" >/dev/null
"$BIN" --root "$STATE" dispatch "$TASK_ID" >/dev/null

for _ in 1 2 3 4 5 6; do
  sleep 15
  "$BIN" --root "$STATE" supervise "$TASK_ID" >/dev/null
  if ! tmux has-session -t "agentctl-$TASK_ID-app-agent" 2>/dev/null; then
    break
  fi
done

WORKTREE="$STATE/tasks/$TASK_ID/worktrees/app"
test -f "$WORKTREE/main.go"
git -C "$WORKTREE" diff -- main.go | grep -q 'flag.String("name"'
(
  cd "$WORKTREE"
  go test ./...
  [[ "$(go run .)" == "hello" ]]
  [[ "$(go run . --name Alice)" == "hello, Alice" ]]
)
"$BIN" --root "$STATE" logs "$TASK_ID" --agent app-agent --lines 200 | grep -q "Verification passed\|go test ./\|hello, Alice"

echo "real opencode e2e passed: $TASK_ID"
