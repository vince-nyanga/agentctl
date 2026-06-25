package cli

import (
	"testing"

	"github.com/vince-nyanga/agentctl/internal/core"
)

func TestSelectTaskRepo(t *testing.T) {
	repos := []core.TaskRepo{{Name: "backend"}, {Name: "frontend"}}

	if _, err := selectTaskRepo(repos, ""); err == nil {
		t.Fatalf("expected multi-repo selection error")
	}

	backend, err := selectTaskRepo(repos, "backend")
	if err != nil {
		t.Fatalf("select backend: %v", err)
	}
	if backend.Name != "backend" {
		t.Fatalf("selected %q", backend.Name)
	}

	if _, err := selectTaskRepo(repos, "missing"); err == nil {
		t.Fatalf("expected missing repo error")
	}
}

func TestSelectTaskRepoDefaultsSingleRepo(t *testing.T) {
	repos := []core.TaskRepo{{Name: "backend"}}
	repo, err := selectTaskRepo(repos, "")
	if err != nil {
		t.Fatalf("select single repo: %v", err)
	}
	if repo.Name != "backend" {
		t.Fatalf("selected %q", repo.Name)
	}
}
