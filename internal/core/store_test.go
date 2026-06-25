package core

import (
	"testing"
	"time"
)

func TestStoreInitLoadSave(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if state.Config.Root != root {
		t.Fatalf("root = %q, want %q", state.Config.Root, root)
	}
	state.Repos["backend"] = Repo{Name: "backend", Path: "/tmp/backend"}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("reload error = %v", err)
	}
	if reloaded.Repos["backend"].Path != "/tmp/backend" {
		t.Fatalf("repo was not persisted")
	}
}

func TestStorePersistsTaskRepoOwnership(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	now := time.Now()
	state.Tasks["task-1"] = Task{
		ID:        "task-1",
		Goal:      "test ownership",
		State:     "planning",
		Workspace: root,
		CreatedAt: now,
		UpdatedAt: now,
		Repos: []TaskRepo{
			{Name: "owned", SourcePath: "/src/owned", WorktreePath: "/wt/owned", Branch: "agent/task-1", Owned: true},
			{Name: "attached", SourcePath: "/src/attached", WorktreePath: "/wt/attached", Branch: "feature/manual", Owned: false},
		},
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("reload error = %v", err)
	}
	repos := reloaded.Tasks["task-1"].Repos
	if len(repos) != 2 {
		t.Fatalf("repos len = %d", len(repos))
	}
	ownership := map[string]bool{}
	for _, repo := range repos {
		ownership[repo.Name] = repo.Owned
	}
	if !ownership["owned"] {
		t.Fatalf("owned repo was not persisted as owned")
	}
	if ownership["attached"] {
		t.Fatalf("attached repo was persisted as owned")
	}
}
