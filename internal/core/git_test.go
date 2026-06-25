package core

import "testing"

func TestIsDirtyStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "clean", status: "clean", want: false},
		{name: "empty", status: "", want: false},
		{name: "unknown", status: "unknown", want: false},
		{name: "modified", status: " M README.md", want: true},
		{name: "untracked", status: "?? scratch.txt", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDirtyStatus(tt.status); got != tt.want {
				t.Fatalf("IsDirtyStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
