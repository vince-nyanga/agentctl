package core

import "testing"

func TestTaskSlug(t *testing.T) {
	tests := []struct {
		name string
		goal string
		want string
	}{
		{name: "simple", goal: "Add refresh-token auth flow", want: "add-refresh-token-auth"},
		{name: "punctuation", goal: "Backend + Frontend: OAuth v2!", want: "backend-frontend-oauth-v2"},
		{name: "empty", goal: "!!!", want: "task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TaskSlug(tt.goal); got != tt.want {
				t.Fatalf("TaskSlug() = %q, want %q", got, tt.want)
			}
		})
	}
}
