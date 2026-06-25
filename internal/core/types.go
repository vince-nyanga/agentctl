package core

import "time"

type Config struct {
	Root      string             `json:"root"`
	Roles     map[string]string  `json:"roles"`
	Harnesses map[string]Harness `json:"harnesses"`
}

type Harness struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type Repo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Remote    string    `json:"remote,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	ID          string     `json:"id"`
	Goal        string     `json:"goal"`
	State       string     `json:"state"`
	Repos       []TaskRepo `json:"repos"`
	Agents      []Agent    `json:"agents"`
	Workspace   string     `json:"workspace"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ManagerNote string     `json:"manager_note,omitempty"`
}

type TaskRepo struct {
	Name         string `json:"name"`
	SourcePath   string `json:"source_path"`
	WorktreePath string `json:"worktree_path"`
	Branch       string `json:"branch"`
}

type Agent struct {
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Harness   string    `json:"harness"`
	Repo      string    `json:"repo,omitempty"`
	State     string    `json:"state"`
	TmuxName  string    `json:"tmux_name,omitempty"`
	Workdir   string    `json:"workdir,omitempty"`
	LogPath   string    `json:"log_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type State struct {
	Config Config          `json:"config"`
	Repos  map[string]Repo `json:"repos"`
	Tasks  map[string]Task `json:"tasks"`
}
