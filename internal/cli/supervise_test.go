package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestPersistAgentOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "agent.log")
	err := persistAgentOutput(core.Agent{Name: "agent", LogPath: path}, "hello")
	if err != nil {
		t.Fatalf("persistAgentOutput() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("log = %q", string(data))
	}
}
