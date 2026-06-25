package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
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
	if err := os.MkdirAll(filepath.Dir(DBPath(s.root)), 0o755); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := migrate(context.Background(), db); err != nil {
		return err
	}
	return s.ensureDefaultState(context.Background(), db)
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
	db, err := s.open()
	if err != nil {
		return State{}, err
	}
	defer db.Close()

	ctx := context.Background()
	state := DefaultState(s.root)
	if err := loadConfig(ctx, db, &state.Config); err != nil {
		return State{}, err
	}
	repos, err := loadRepos(ctx, db)
	if err != nil {
		return State{}, err
	}
	tasks, err := loadTasks(ctx, db)
	if err != nil {
		return State{}, err
	}
	state.Repos = repos
	state.Tasks = tasks
	return state, nil
}

func (s *Store) Save(state State) error {
	if err := s.Init(); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := saveConfig(ctx, tx, state.Config); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agents`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_repos`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM repos`); err != nil {
		return err
	}

	for _, repo := range state.Repos {
		if err := insertRepo(ctx, tx, repo); err != nil {
			return err
		}
	}
	for _, task := range state.Tasks {
		if err := insertTask(ctx, tx, task); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) open() (*sql.DB, error) {
	db, err := sql.Open("sqlite", DBPath(s.root))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func (s *Store) ensureDefaultState(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM settings`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := saveConfig(ctx, tx, DefaultState(s.root).Config); err != nil {
		return err
	}
	return tx.Commit()
}

func migrate(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS settings (
            key TEXT PRIMARY KEY,
            value TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS repos (
            name TEXT PRIMARY KEY,
            path TEXT NOT NULL,
            remote TEXT NOT NULL DEFAULT '',
            created_at TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS tasks (
            id TEXT PRIMARY KEY,
            goal TEXT NOT NULL,
            state TEXT NOT NULL,
            workspace TEXT NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            manager_note TEXT NOT NULL DEFAULT ''
        )`,
		`CREATE TABLE IF NOT EXISTS task_repos (
            task_id TEXT NOT NULL,
            name TEXT NOT NULL,
            source_path TEXT NOT NULL,
            worktree_path TEXT NOT NULL,
            branch TEXT NOT NULL,
            owned INTEGER NOT NULL DEFAULT 1,
            PRIMARY KEY (task_id, name),
            FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS agents (
            task_id TEXT NOT NULL,
            name TEXT NOT NULL,
            role TEXT NOT NULL,
            harness TEXT NOT NULL,
            repo TEXT NOT NULL DEFAULT '',
            state TEXT NOT NULL,
            tmux_name TEXT NOT NULL DEFAULT '',
            workdir TEXT NOT NULL DEFAULT '',
            log_path TEXT NOT NULL DEFAULT '',
            created_at TEXT NOT NULL,
            PRIMARY KEY (task_id, name),
            FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS events (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            task_id TEXT NOT NULL DEFAULT '',
            agent_name TEXT NOT NULL DEFAULT '',
            type TEXT NOT NULL,
            message TEXT NOT NULL,
            created_at TEXT NOT NULL
        )`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := ensureColumn(ctx, db, "task_repos", "owned", `ALTER TABLE task_repos ADD COLUMN owned INTEGER NOT NULL DEFAULT 1`); err != nil {
		return err
	}
	return nil
}

func ensureColumn(ctx context.Context, db *sql.DB, table, column, alter string) error {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, alter)
	return err
}

func saveConfig(ctx context.Context, tx *sql.Tx, config Config) error {
	roles, err := json.Marshal(config.Roles)
	if err != nil {
		return err
	}
	harnesses, err := json.Marshal(config.Harnesses)
	if err != nil {
		return err
	}
	settings := map[string]string{
		"root":      config.Root,
		"roles":     string(roles),
		"harnesses": string(harnesses),
	}
	for key, value := range settings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
			return err
		}
	}
	return nil
}

func loadConfig(ctx context.Context, db *sql.DB, config *Config) error {
	rows, err := db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		switch key {
		case "root":
			config.Root = value
		case "roles":
			if err := json.Unmarshal([]byte(value), &config.Roles); err != nil {
				return err
			}
		case "harnesses":
			if err := json.Unmarshal([]byte(value), &config.Harnesses); err != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func loadRepos(ctx context.Context, db *sql.DB) (map[string]Repo, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, path, remote, created_at FROM repos ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	repos := map[string]Repo{}
	for rows.Next() {
		var repo Repo
		var createdAt string
		if err := rows.Scan(&repo.Name, &repo.Path, &repo.Remote, &createdAt); err != nil {
			return nil, err
		}
		repo.CreatedAt = parseTime(createdAt)
		repos[repo.Name] = repo
	}
	return repos, rows.Err()
}

func loadTasks(ctx context.Context, db *sql.DB) (map[string]Task, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, goal, state, workspace, created_at, updated_at, manager_note FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := map[string]Task{}
	for rows.Next() {
		var task Task
		var createdAt, updatedAt string
		if err := rows.Scan(&task.ID, &task.Goal, &task.State, &task.Workspace, &createdAt, &updatedAt, &task.ManagerNote); err != nil {
			return nil, err
		}
		task.CreatedAt = parseTime(createdAt)
		task.UpdatedAt = parseTime(updatedAt)
		task.Repos, err = loadTaskRepos(ctx, db, task.ID)
		if err != nil {
			return nil, err
		}
		task.Agents, err = loadAgents(ctx, db, task.ID)
		if err != nil {
			return nil, err
		}
		tasks[task.ID] = task
	}
	return tasks, rows.Err()
}

func loadTaskRepos(ctx context.Context, db *sql.DB, taskID string) ([]TaskRepo, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, source_path, worktree_path, branch, owned FROM task_repos WHERE task_id = ? ORDER BY name`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []TaskRepo
	for rows.Next() {
		var repo TaskRepo
		var owned int
		if err := rows.Scan(&repo.Name, &repo.SourcePath, &repo.WorktreePath, &repo.Branch, &owned); err != nil {
			return nil, err
		}
		repo.Owned = owned == 1
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

func loadAgents(ctx context.Context, db *sql.DB, taskID string) ([]Agent, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, role, harness, repo, state, tmux_name, workdir, log_path, created_at FROM agents WHERE task_id = ? ORDER BY name`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var agent Agent
		var createdAt string
		if err := rows.Scan(&agent.Name, &agent.Role, &agent.Harness, &agent.Repo, &agent.State, &agent.TmuxName, &agent.Workdir, &agent.LogPath, &createdAt); err != nil {
			return nil, err
		}
		agent.CreatedAt = parseTime(createdAt)
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func insertRepo(ctx context.Context, tx *sql.Tx, repo Repo) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO repos(name, path, remote, created_at) VALUES(?, ?, ?, ?)`, repo.Name, repo.Path, repo.Remote, formatTime(repo.CreatedAt))
	return err
}

func insertTask(ctx context.Context, tx *sql.Tx, task Task) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO tasks(id, goal, state, workspace, created_at, updated_at, manager_note) VALUES(?, ?, ?, ?, ?, ?, ?)`, task.ID, task.Goal, task.State, task.Workspace, formatTime(task.CreatedAt), formatTime(task.UpdatedAt), task.ManagerNote)
	if err != nil {
		return err
	}
	for _, repo := range task.Repos {
		owned := 0
		if repo.Owned {
			owned = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_repos(task_id, name, source_path, worktree_path, branch, owned) VALUES(?, ?, ?, ?, ?, ?)`, task.ID, repo.Name, repo.SourcePath, repo.WorktreePath, repo.Branch, owned); err != nil {
			return err
		}
	}
	for _, agent := range task.Agents {
		if _, err := tx.ExecContext(ctx, `INSERT INTO agents(task_id, name, role, harness, repo, state, tmux_name, workdir, log_path, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, task.ID, agent.Name, agent.Role, agent.Harness, agent.Repo, agent.State, agent.TmuxName, agent.Workdir, agent.LogPath, formatTime(agent.CreatedAt)); err != nil {
			return err
		}
	}
	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (s *Store) AddEvent(event Event) error {
	if err := s.Init(); err != nil {
		return err
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO events(task_id, agent_name, type, message, created_at) VALUES(?, ?, ?, ?, ?)`, event.TaskID, event.AgentName, event.Type, event.Message, formatTime(event.CreatedAt))
	return err
}

func (s *Store) ListEvents(taskID string, limit int) ([]Event, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, task_id, agent_name, type, message, created_at FROM events`
	args := []any{}
	if taskID != "" {
		query += ` WHERE task_id = ?`
		args = append(args, taskID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		var createdAt string
		if err := rows.Scan(&event.ID, &event.TaskID, &event.AgentName, &event.Type, &event.Message, &createdAt); err != nil {
			return nil, err
		}
		event.CreatedAt = parseTime(createdAt)
		events = append(events, event)
	}
	return events, rows.Err()
}

func dumpStateForDebug(state State) string {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Sprintf("<marshal state: %v>", err)
	}
	return string(data)
}
