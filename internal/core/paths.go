package core

import (
	"os"
	"path/filepath"
)

const appDirName = ".agentctl"

func DefaultRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".agentctl"
	}
	return filepath.Join(home, appDirName)
}

func StatePath(root string) string {
	return filepath.Join(root, "state.json")
}

func TasksDir(root string) string {
	return filepath.Join(root, "tasks")
}

func ConfigFromEnv() string {
	if root := os.Getenv("AGENTCTL_ROOT"); root != "" {
		return root
	}
	return DefaultRoot()
}
