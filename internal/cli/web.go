package cli

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/spf13/cobra"
	"github.com/vince-nyanga/agentctl/internal/core"
)

func newWebCommand(ctx *appContext) *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Serve a read-only local web dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			mux := http.NewServeMux()
			mux.HandleFunc("/", webIndexHandler(ctx.store))
			mux.HandleFunc("/api/state", webStateHandler(ctx.store))
			fmt.Fprintf(cmd.OutOrStdout(), "serving read-only dashboard at http://%s\n", addr)
			return http.ListenAndServe(addr, mux)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8765", "listen address")
	return cmd
}

func webStateHandler(store *core.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := store.Load()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		approvals, _ := store.ListApprovals("", "pending")
		events, _ := store.ListEvents("", 50)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"state": state, "approvals": approvals, "events": events})
	}
}

func webIndexHandler(store *core.Store) http.HandlerFunc {
	tmpl := template.Must(template.New("index").Parse(webIndexHTML))
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := store.Load()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		approvals, _ := store.ListApprovals("", "pending")
		events, _ := store.ListEvents("", 20)
		tasks := make([]core.Task, 0, len(state.Tasks))
		for _, task := range state.Tasks {
			tasks = append(tasks, task)
		}
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.After(tasks[j].CreatedAt) })
		data := struct {
			Tasks     []core.Task
			Approvals []core.Approval
			Events    []core.Event
		}{Tasks: tasks, Approvals: approvals, Events: events}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.Execute(w, data)
	}
}

const webIndexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>agentctl</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; background: #0b1020; color: #e8edff; }
    h1, h2 { margin-bottom: .5rem; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1rem; }
    .card { border: 1px solid #26345f; border-radius: 14px; padding: 1rem; background: #111936; }
    .muted { color: #99a6c8; }
    code { color: #7cc4ff; }
    table { width: 100%; border-collapse: collapse; }
    td, th { border-bottom: 1px solid #26345f; padding: .5rem; text-align: left; vertical-align: top; }
  </style>
</head>
<body>
  <h1>Agent Mission Control</h1>
  <p class="muted">Read-only local dashboard. Refresh the page for latest state.</p>
  <div class="grid">
    <section class="card"><h2>Tasks</h2><table><tr><th>ID</th><th>State</th><th>Goal</th></tr>{{range .Tasks}}<tr><td><code>{{.ID}}</code></td><td>{{.State}}</td><td>{{.Goal}}</td></tr>{{else}}<tr><td colspan="3">No tasks</td></tr>{{end}}</table></section>
    <section class="card"><h2>Approvals</h2><table><tr><th>ID</th><th>Task</th><th>Title</th></tr>{{range .Approvals}}<tr><td>#{{.ID}}</td><td><code>{{.TaskID}}</code></td><td>{{.Title}}</td></tr>{{else}}<tr><td colspan="3">No pending approvals</td></tr>{{end}}</table></section>
  </div>
  <section class="card" style="margin-top:1rem"><h2>Recent Events</h2><table><tr><th>Time</th><th>Task</th><th>Type</th><th>Message</th></tr>{{range .Events}}<tr><td>{{.CreatedAt.Format "15:04:05"}}</td><td><code>{{.TaskID}}</code></td><td>{{.Type}}</td><td>{{.Message}}</td></tr>{{else}}<tr><td colspan="4">No events</td></tr>{{end}}</table></section>
</body>
</html>`
