package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func StartTmuxAgent(sessionName, workdir, command, prompt, logPath string) error {
	if !HasCommand("tmux") {
		return fmt.Errorf("tmux is required but was not found")
	}
	if command == "" {
		return fmt.Errorf("agent command is empty")
	}
	if err := exec.Command("tmux", "has-session", "-t", sessionName).Run(); err == nil {
		return nil
	}
	if err := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", workdir, command).Run(); err != nil {
		return err
	}
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return err
		}
		pipeCommand := "cat >> " + shellQuote(logPath)
		if err := exec.Command("tmux", "pipe-pane", "-o", "-t", sessionName, pipeCommand).Run(); err != nil {
			return err
		}
	}
	if strings.TrimSpace(prompt) != "" {
		return SendTmux(sessionName, prompt)
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func TmuxSessionExists(target string) bool {
	if target == "" || !HasCommand("tmux") {
		return false
	}
	return exec.Command("tmux", "has-session", "-t", target).Run() == nil
}

func SendTmux(target, message string) error {
	if err := exec.Command("tmux", "send-keys", "-t", target, message).Run(); err != nil {
		return err
	}
	return exec.Command("tmux", "send-keys", "-t", target, "Enter").Run()
}

func TailTmux(target string, lines int) (string, error) {
	if lines <= 0 {
		lines = 40
	}
	out, err := exec.Command("tmux", "capture-pane", "-t", target, "-p", "-J", "-S", fmt.Sprintf("-%d", lines)).Output()
	return string(out), err
}

func KillTmuxSession(target string) error {
	if target == "" {
		return nil
	}
	if err := exec.Command("tmux", "has-session", "-t", target).Run(); err != nil {
		return nil
	}
	return exec.Command("tmux", "kill-session", "-t", target).Run()
}
