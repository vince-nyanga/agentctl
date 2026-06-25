package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	root string
}

func NewStore(root string) *Store {
	if root == "" {
		root = ConfigFromEnv()
	}
	return &Store{root: root}
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Init() error {
	if err := os.MkdirAll(TasksDir(s.root), 0o755); err != nil {
		return err
	}
	path := StatePath(s.root)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	state := DefaultState(s.root)
	return s.Save(state)
}

func DefaultState(root string) State {
	return State{
		Config: Config{
			Root: root,
			Roles: map[string]string{
				"manager":  "opencode",
				"worker":   "opencode",
				"reviewer": "opencode",
			},
			Harnesses: map[string]Harness{
				"opencode": {Command: "opencode"},
				"claude":   {Command: "claude"},
				"codex":    {Command: "codex"},
				"pi":       {Command: "pi"},
			},
		},
		Repos: map[string]Repo{},
		Tasks: map[string]Task{},
	}
}

func (s *Store) Load() (State, error) {
	if err := s.Init(); err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(StatePath(s.root))
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	if state.Repos == nil {
		state.Repos = map[string]Repo{}
	}
	if state.Tasks == nil {
		state.Tasks = map[string]Task{}
	}
	if state.Config.Root == "" {
		state.Config.Root = s.root
	}
	return state, nil
}

func (s *Store) Save(state State) error {
	if err := os.MkdirAll(filepath.Dir(StatePath(s.root)), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(StatePath(s.root), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}
