package core

import "testing"

func TestShellQuote(t *testing.T) {
	got := ShellQuote("/tmp/it's/log.txt")
	want := "'/tmp/it'\\''s/log.txt'"
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}
