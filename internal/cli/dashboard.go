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
			eventsByTask := map[string][]core.Event{}
			for taskID := range state.Tasks {
				events, err := ctx.store.ListEvents(taskID, 5)
				if err == nil {
					eventsByTask[taskID] = events
				}
			}
			allEvents, err := ctx.store.ListEvents("", 30)
			if err != nil {
				allEvents = nil
			}
			model := newDashboardModel(state, eventsByTask, allEvents)
			_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
			return err
		},
	}
}

type dashboardModel struct {
	state     core.State
	tasks     []core.Task
	selected  int
	width     int
	height    int
	events    map[string][]core.Event
	allEvents []core.Event
	tab       int
}

func newDashboardModel(state core.State, events map[string][]core.Event, allEvents []core.Event) dashboardModel {
	tasks := make([]core.Task, 0, len(state.Tasks))
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
	if events == nil {
		events = map[string][]core.Event{}
	}
	return dashboardModel{state: state, tasks: tasks, events: events, allEvents: allEvents}
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("l", "right", "tab"))):
			m.tab = (m.tab + 1) % len(dashboardTabs)
		case key.Matches(msg, key.NewBinding(key.WithKeys("h", "left", "shift+tab"))):
			m.tab--
			if m.tab < 0 {
				m.tab = len(dashboardTabs) - 1
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
	tabs := m.renderTabs(m.width)
	footer := footerStyle.Width(m.width).Render("h/l switch tabs | j/k move tasks | q quit | open: agentctl open <task> --agent <name>")
	if len(m.tasks) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, header, tabs, emptyStyle.Render("No tasks yet. Start with: agentctl plan \"Your goal\" --repo <name>"), footer)
	}

	body := m.renderActiveTab(m.width)
	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, footer)
}

var dashboardTabs = []string{"Overview", "Tasks", "Approvals", "Blocked", "Events", "Detail"}

func (m dashboardModel) renderTabs(width int) string {
	parts := make([]string, 0, len(dashboardTabs))
	for i, tab := range dashboardTabs {
		style := inactiveTabStyle
		if i == m.tab {
			style = activeTabStyle
		}
		parts = append(parts, style.Render(tab))
	}
	return tabsStyle.Width(width).Render(strings.Join(parts, " "))
}

func (m dashboardModel) renderActiveTab(width int) string {
	switch dashboardTabs[m.tab] {
	case "Overview":
		return m.renderOverview(width)
	case "Tasks":
		return m.renderTaskBoard(width)
	case "Approvals":
		return m.renderApprovals(width)
	case "Blocked":
		return m.renderBlocked(width)
	case "Events":
		return m.renderEvents(width)
	case "Detail":
		return m.renderTaskBoard(width)
	default:
		return m.renderOverview(width)
	}
}

func (m dashboardModel) renderOverview(width int) string {
	total, running, planning, done, archived := 0, 0, 0, 0, 0
	agents, stopped := 0, 0
	for _, task := range m.tasks {
		total++
		switch task.State {
		case "running":
			running++
		case "planning":
			planning++
		case "done":
			done++
		case "archived":
			archived++
		}
		for _, agent := range task.Agents {
			agents++
			if agent.State != "running" {
				stopped++
			}
		}
	}
	metrics := []string{
		fmt.Sprintf("Tasks:          %d", total),
		fmt.Sprintf("Running:        %d", running),
		fmt.Sprintf("Planning:       %d", planning),
		fmt.Sprintf("Done:           %d", done),
		fmt.Sprintf("Archived:       %d", archived),
		fmt.Sprintf("Agents:         %d", agents),
		fmt.Sprintf("Needs attention:%d", len(m.attentionTasks())),
		fmt.Sprintf("Stopped agents: %d", stopped),
	}
	return panelStyle.Width(width - 4).Render(sectionStyle.Render("Overview") + "\n\n" + strings.Join(metrics, "\n"))
}

func (m dashboardModel) renderTaskBoard(width int) string {
	leftWidth := 40
	if width < 100 {
		leftWidth = 34
	}
	rightWidth := width - leftWidth - 4
	if rightWidth < 45 {
		rightWidth = 45
	}
	left := m.renderTaskList(leftWidth)
	right := m.renderTaskDetail(rightWidth)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m dashboardModel) renderTaskList(width int) string {
	rows := []string{sectionStyle.Render("Tasks")}
	for i, task := range m.tasks {
		style := taskStyle
		if i == m.selected {
			style = selectedTaskStyle
		}
		reason := attentionReason(task)
		if reason == "" {
			reason = "ok"
		}
		line := fmt.Sprintf("%s\n%s | %d repos | %d agents\n%s", task.ID, task.State, len(task.Repos), len(task.Agents), reason)
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
		owned := "owned"
		if !repo.Owned {
			owned = "attached"
		}
		b.WriteString(fmt.Sprintf("- %s  %s\n  branch: %s | %s\n", repo.Name, repo.WorktreePath, repo.Branch, owned))
	}
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Agents"))
	b.WriteString("\n")
	if len(task.Agents) == 0 {
		b.WriteString("No agents started yet.\n")
	} else {
		for _, agent := range task.Agents {
			b.WriteString(fmt.Sprintf("- %s [%s/%s] %s\n  tmux: %s\n", agent.Name, agent.Role, agent.Harness, agent.State, agent.TmuxName))
		}
	}
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Recent Events"))
	b.WriteString("\n")
	events := m.events[task.ID]
	if len(events) == 0 {
		b.WriteString("No events recorded yet.\n")
	} else {
		for _, event := range events {
			actor := event.AgentName
			if actor == "" {
				actor = "system"
			}
			b.WriteString(fmt.Sprintf("- %s [%s] %s: %s\n", event.CreatedAt.Format("15:04:05"), actor, event.Type, event.Message))
		}
	}
	return panelStyle.Width(width).Render(b.String())
}

func (m dashboardModel) renderApprovals(width int) string {
	var rows []string
	for _, task := range m.tasks {
		if task.State == "planning" {
			rows = append(rows, fmt.Sprintf("- %s | plan approval | %s", task.ID, task.Goal))
		}
	}
	if len(rows) == 0 {
		rows = append(rows, "No pending approvals.")
	}
	return panelStyle.Width(width - 4).Render(sectionStyle.Render("Approvals") + "\n\n" + strings.Join(rows, "\n"))
}

func (m dashboardModel) renderBlocked(width int) string {
	var rows []string
	for _, task := range m.attentionTasks() {
		rows = append(rows, fmt.Sprintf("- %s | %s | %s | %s", task.ID, task.State, attentionReason(task), task.Goal))
	}
	if len(rows) == 0 {
		rows = append(rows, "No tasks need attention.")
	}
	return panelStyle.Width(width - 4).Render(sectionStyle.Render("Blocked / Needs Attention") + "\n\n" + strings.Join(rows, "\n"))
}

func (m dashboardModel) renderEvents(width int) string {
	var rows []string
	for _, event := range m.allEvents {
		actor := event.AgentName
		if actor == "" {
			actor = "system"
		}
		rows = append(rows, fmt.Sprintf("- %s | %s | %s | %s | %s", event.CreatedAt.Format("15:04:05"), event.TaskID, actor, event.Type, event.Message))
	}
	if len(rows) == 0 {
		rows = append(rows, "No events recorded yet.")
	}
	return panelStyle.Width(width - 4).Render(sectionStyle.Render("Recent Events") + "\n\n" + strings.Join(rows, "\n"))
}

func (m dashboardModel) attentionTasks() []core.Task {
	var tasks []core.Task
	for _, task := range m.tasks {
		if task.State == "archived" || task.State == "done" {
			continue
		}
		if attentionReason(task) != "" {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

var (
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Padding(0, 1)
	footerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(1, 0, 0, 0)
	tabsStyle         = lipgloss.NewStyle().Padding(1, 0, 0, 0)
	activeTabStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62")).Padding(0, 1)
	inactiveTabStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1)
	panelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2).MarginRight(1)
	sectionStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	labelStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	taskStyle         = lipgloss.NewStyle().Padding(0, 1).MarginTop(1).Foreground(lipgloss.Color("252"))
	selectedTaskStyle = taskStyle.Copy().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(lipgloss.Color("229")).Foreground(lipgloss.Color("229"))
	emptyStyle        = lipgloss.NewStyle().Padding(2).Foreground(lipgloss.Color("244"))
)
