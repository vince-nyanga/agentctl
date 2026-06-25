package cli

import (
	"strings"
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestBuildHarnessCommandUsesPromptArgForOpenCodeRun(t *testing.T) {
	command, prompt := buildHarnessCommand(core.Harness{Command: "opencode", Args: []string{"run", "--dangerously-skip-permissions"}, Mode: "prompt_arg"}, "hello manager")
	if prompt != "" {
		t.Fatalf("prompt to send = %q", prompt)
	}
	if !strings.Contains(command, "opencode run --dangerously-skip-permissions") || !strings.Contains(command, "'hello manager'") {
		t.Fatalf("command = %q", command)
	}
}

func TestBuildHarnessCommandSendsPromptForInteractiveHarness(t *testing.T) {
	command, prompt := buildHarnessCommand(core.Harness{Command: "cat"}, "hello")
	if command != "cat" || prompt != "hello" {
		t.Fatalf("command=%q prompt=%q", command, prompt)
	}
}

func TestResolvePlanReposDefaultsToAllRegisteredRepos(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	state.Repos["frontend"] = core.Repo{Name: "frontend"}
	state.Repos["backend"] = core.Repo{Name: "backend"}
	repos, err := resolvePlanRepos(state, nil)
	if err != nil {
		t.Fatalf("resolvePlanRepos() error = %v", err)
	}
	want := []string{"backend", "frontend"}
	if len(repos) != len(want) {
		t.Fatalf("repos = %#v", repos)
	}
	for i := range want {
		if repos[i] != want[i] {
			t.Fatalf("repos = %#v, want %#v", repos, want)
		}
	}
}

func TestResolvePlanReposUsesExplicitRepos(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	repos, err := resolvePlanRepos(state, []string{"api"})
	if err != nil {
		t.Fatalf("resolvePlanRepos() error = %v", err)
	}
	if len(repos) != 1 || repos[0] != "api" {
		t.Fatalf("repos = %#v", repos)
	}
}

func TestResolvePlanReposErrorsWithoutRegisteredRepos(t *testing.T) {
	state := core.DefaultState(t.TempDir())
	if _, err := resolvePlanRepos(state, nil); err == nil {
		t.Fatalf("expected error without registered repos")
	}
}
