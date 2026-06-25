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
