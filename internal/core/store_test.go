package core

import "testing"

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
