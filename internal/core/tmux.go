package core

import (
	"fmt"
	"os/exec"
	"strings"
)

func HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func StartTmuxAgent(sessionName, workdir, command, prompt string) error {
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
	if strings.TrimSpace(prompt) != "" {
		return SendTmux(sessionName, prompt)
	}
	return nil
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
	out, err := exec.Command("tmux", "capture-pane", "-t", target, "-p", "-S", fmt.Sprintf("-%d", lines)).Output()
	return string(out), err
}
