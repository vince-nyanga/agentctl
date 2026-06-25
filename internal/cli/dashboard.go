package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

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
			model, err := loadDashboardModel(ctx.store)
			if err != nil {
				return err
			}
			_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
			return err
		},
	}
}

type dashboardModel struct {
	state     core.State
	store     *core.Store
	tasks     []core.Task
	selected  int
	width     int
	height    int
	events    map[string][]core.Event
	allEvents []core.Event
	approvals []core.Approval
	tab       int
	message   string
	pending   string
}

func loadDashboardModel(store *core.Store) (dashboardModel, error) {
	state, err := store.Load()
	if err != nil {
		return dashboardModel{}, err
	}
	eventsByTask := map[string][]core.Event{}
	for taskID := range state.Tasks {
		events, err := store.ListEvents(taskID, 5)
		if err == nil {
			eventsByTask[taskID] = events
		}
	}
	allEvents, err := store.ListEvents("", 30)
	if err != nil {
		allEvents = nil
	}
	approvals, err := store.ListApprovals("", "pending")
	if err != nil {
		approvals = nil
	}
	model := newDashboardModel(state, eventsByTask, allEvents, approvals)
	model.store = store
	return model, nil
}

func newDashboardModel(state core.State, events map[string][]core.Event, allEvents []core.Event, approvals []core.Approval) dashboardModel {
	tasks := make([]core.Task, 0, len(state.Tasks))
	for _, task := range state.Tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
	if events == nil {
		events = map[string][]core.Event{}
	}
	return dashboardModel{state: state, tasks: tasks, events: events, allEvents: allEvents, approvals: approvals}
}

func (m dashboardModel) Init() tea.Cmd { return nil }

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.pending != "" {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
				return m.confirmPending()
			case key.Matches(msg, key.NewBinding(key.WithKeys("n", "esc"))):
				m.message = "cancelled " + m.pending
				m.pending = ""
				return m, nil
			}
		}
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			return m.refresh()
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			return m.approveSelectedPlan()
		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			return m.stageAction("dispatch")
		case key.Matches(msg, key.NewBinding(key.WithKeys("x"))):
			return m.stageAction("archive")
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
	footerText := "h/l tabs | j/k tasks | r refresh | a approve | d dispatch | x archive | q quit"
	if m.pending != "" {
		footerText = "confirm " + m.pending + "? y/n"
	}
	if m.message != "" {
		footerText = m.message + " | " + footerText
	}
	footer := footerStyle.Width(m.width).Render(footerText)
	if len(m.tasks) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, header, tabs, emptyStyle.Render("No tasks yet. Start with: agentctl plan \"Your goal\" --repo <name>"), footer)
	}

	body := m.renderActiveTab(m.width)
	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, footer)
}

func (m dashboardModel) stageAction(action string) (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		m.message = "no task selected"
		return m, nil
	}
	m.pending = action
	m.message = action + " staged for " + m.tasks[m.selected].ID
	return m, nil
}

func (m dashboardModel) confirmPending() (tea.Model, tea.Cmd) {
	switch m.pending {
	case "dispatch":
		m.pending = ""
		return m.dispatchSelectedTask()
	case "archive":
		m.pending = ""
		return m.archiveSelectedTask()
	default:
		m.message = "unknown pending action"
		m.pending = ""
		return m, nil
	}
}

func (m dashboardModel) refresh() (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.message = "refresh unavailable"
		return m, nil
	}
	refreshed, err := loadDashboardModel(m.store)
	if err != nil {
		m.message = "refresh failed: " + err.Error()
		return m, nil
	}
	refreshed.width = m.width
	refreshed.height = m.height
	refreshed.tab = m.tab
	if m.selected < len(refreshed.tasks) {
		refreshed.selected = m.selected
	}
	refreshed.message = "refreshed"
	return refreshed, nil
}

func (m dashboardModel) approveSelectedPlan() (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.message = "approval unavailable"
		return m, nil
	}
	if len(m.tasks) == 0 {
		m.message = "no task selected"
		return m, nil
	}
	task := m.tasks[m.selected]
	if task.State != "planning" {
		m.message = "selected task is not awaiting plan approval"
		return m, nil
	}
	state, err := m.store.Load()
	if err != nil {
		m.message = "approval failed: " + err.Error()
		return m, nil
	}
	task = state.Tasks[task.ID]
	task.State = "plan_approved"
	task.UpdatedAt = time.Now()
	state.Tasks[task.ID] = task
	if err := m.store.Save(state); err != nil {
		m.message = "approval failed: " + err.Error()
		return m, nil
	}
	approvals, err := m.store.ListApprovals(task.ID, "pending")
	if err == nil {
		for _, approval := range approvals {
			if approval.Type == "plan" {
				_, _ = m.store.ResolveApproval(approval.ID, "approved")
			}
		}
	}
	_ = m.store.AddEvent(core.Event{TaskID: task.ID, Type: "plan.approved", Message: "plan approved from dashboard"})
	refreshed, _ := m.refresh()
	next := refreshed.(dashboardModel)
	next.message = "approved plan for " + task.ID
	return next, nil
}

func (m dashboardModel) dispatchSelectedTask() (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.message = "dispatch unavailable"
		return m, nil
	}
	if len(m.tasks) == 0 {
		m.message = "no task selected"
		return m, nil
	}
	state, err := m.store.Load()
	if err != nil {
		m.message = "dispatch failed: " + err.Error()
		return m, nil
	}
	task := state.Tasks[m.tasks[m.selected].ID]
	if task.State != "plan_approved" {
		m.message = "dispatch requires approved plan"
		return m, nil
	}
	for _, repo := range task.Repos {
		briefPath, err := prepareWorkerDispatchPrompt(task, repo)
		if err != nil {
			m.message = "dispatch failed: " + err.Error()
			return m, nil
		}
		agentName := repo.Name + "-agent"
		agent, err := startAgent(state, task, "worker", agentName, repo.Name, repo.WorktreePath, briefPath)
		if err != nil {
			m.message = "dispatch failed: " + err.Error()
			return m, nil
		}
		task.Agents = append(task.Agents, agent)
	}
	task.State = "running"
	task.UpdatedAt = time.Now()
	state.Tasks[task.ID] = task
	if err := m.store.Save(state); err != nil {
		m.message = "dispatch failed: " + err.Error()
		return m, nil
	}
	_ = m.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.dispatched", Message: fmt.Sprintf("dispatched %d workers from dashboard", len(task.Repos))})
	refreshed, _ := m.refresh()
	next := refreshed.(dashboardModel)
	next.message = "dispatched " + task.ID
	return next, nil
}

func (m dashboardModel) archiveSelectedTask() (tea.Model, tea.Cmd) {
	if m.store == nil {
		m.message = "archive unavailable"
		return m, nil
	}
	if len(m.tasks) == 0 {
		m.message = "no task selected"
		return m, nil
	}
	state, err := m.store.Load()
	if err != nil {
		m.message = "archive failed: " + err.Error()
		return m, nil
	}
	task := state.Tasks[m.tasks[m.selected].ID]
	for _, agent := range task.Agents {
		if err := core.KillTmuxSession(agent.TmuxName); err != nil {
			m.message = "archive failed: " + err.Error()
			return m, nil
		}
	}
	for _, repo := range task.Repos {
		if !repo.Owned {
			continue
		}
		status := core.GitStatusShort(repo.WorktreePath)
		if core.IsDirtyStatus(status) {
			m.message = "archive blocked: dirty worktree " + repo.Name
			return m, nil
		}
		if err := core.RemoveWorktree(repo.SourcePath, repo.WorktreePath); err != nil {
			m.message = "archive failed: " + err.Error()
			return m, nil
		}
		_ = core.PruneWorktrees(repo.SourcePath)
	}
	for i := range task.Agents {
		task.Agents[i].State = "stopped"
	}
	task.State = "archived"
	task.UpdatedAt = time.Now()
	state.Tasks[task.ID] = task
	if err := m.store.Save(state); err != nil {
		m.message = "archive failed: " + err.Error()
		return m, nil
	}
	_ = m.store.AddEvent(core.Event{TaskID: task.ID, Type: "task.archived", Message: "archived task from dashboard"})
	refreshed, _ := m.refresh()
	next := refreshed.(dashboardModel)
	next.message = "archived " + task.ID
	return next, nil
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
		fmt.Sprintf("Approvals:      %d", len(m.approvals)),
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
	for _, approval := range m.approvals {
		rows = append(rows, fmt.Sprintf("- #%d | %s | %s | %s | %s", approval.ID, approval.TaskID, approval.Type, approval.Risk, approval.Title))
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
