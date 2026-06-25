package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newDashboardCommand(ctx *appContext) *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the Agent Mission Control TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := ctx.store.Load()
			if err != nil {
				return err
			}
			model := newDashboardModel(state)
			_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
			return err
		},
	}
}

type dashboardModel struct {
	state    core.State
	tasks    []core.Task
	selected int
	width    int
	height   int
}

func newDashboardModel(state core.State) dashboardModel {
	tasks := make([]core.Task, 0, len(state.Tasks))
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
	return dashboardModel{state: state, tasks: tasks}
}

func (m dashboardModel) Init() tea.Cmd { return nil }

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.selected < len(m.tasks)-1 {
				m.selected++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.selected > 0 {
				m.selected--
			}
		}
	}
	return m, nil
}

func (m dashboardModel) View() string {
	if m.width == 0 {
		m.width = 100
	}
	header := titleStyle.Width(m.width).Render("Agent Mission Control")
	footer := footerStyle.Width(m.width).Render("j/k move · q quit · use `agentctl open <task> --agent <name>` to attach")
	if len(m.tasks) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, header, emptyStyle.Render("No tasks yet. Start with: agentctl plan \"Your goal\" --repo <name>"), footer)
	}

	leftWidth := 36
	if m.width < 90 {
		leftWidth = 30
	}
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 40 {
		rightWidth = 40
	}
	left := m.renderTaskList(leftWidth)
	right := m.renderTaskDetail(rightWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m dashboardModel) renderTaskList(width int) string {
	rows := []string{sectionStyle.Render("Tasks")}
	for i, task := range m.tasks {
		style := taskStyle
		if i == m.selected {
			style = selectedTaskStyle
		}
		line := fmt.Sprintf("%s\n%s · %d repos · %d agents", task.ID, task.State, len(task.Repos), len(task.Agents))
		rows = append(rows, style.Width(width).Render(line))
	}
	return panelStyle.Width(width).Render(strings.Join(rows, "\n"))
}

func (m dashboardModel) renderTaskDetail(width int) string {
	task := m.tasks[m.selected]
	var b strings.Builder
	b.WriteString(sectionStyle.Render(task.ID))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Goal: "))
	b.WriteString(task.Goal)
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("State: "))
	b.WriteString(task.State)
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Workspace: "))
	b.WriteString(task.Workspace)
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Repos"))
	b.WriteString("\n")
	for _, repo := range task.Repos {
		b.WriteString(fmt.Sprintf("• %s  %s\n  branch: %s\n", repo.Name, repo.WorktreePath, repo.Branch))
	}
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Agents"))
	b.WriteString("\n")
	if len(task.Agents) == 0 {
		b.WriteString("No agents started yet.\n")
	} else {
		for _, agent := range task.Agents {
			b.WriteString(fmt.Sprintf("• %s [%s/%s] %s\n  tmux: %s\n", agent.Name, agent.Role, agent.Harness, agent.State, agent.TmuxName))
		}
	}
	return panelStyle.Width(width).Render(b.String())
}

var (
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Padding(0, 1)
	footerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(1, 0, 0, 0)
	panelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2).MarginRight(1)
	sectionStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	labelStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	taskStyle         = lipgloss.NewStyle().Padding(0, 1).MarginTop(1).Foreground(lipgloss.Color("252"))
	selectedTaskStyle = taskStyle.Copy().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("229")).Foreground(lipgloss.Color("229"))
	emptyStyle        = lipgloss.NewStyle().Padding(2).Foreground(lipgloss.Color("244"))
)
