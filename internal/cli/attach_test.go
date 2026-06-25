package cli

import "testing"

func TestParseRepoPathSpec(t *testing.T) {
	name, path, err := parseRepoPathSpec("backend=/tmp/backend")
	if err != nil {
		t.Fatalf("parseRepoPathSpec() error = %v", err)
	}
	if name != "backend" || path != "/tmp/backend" {
		t.Fatalf("got %q %q", name, path)
	}

	for _, spec := range []string{"backend", "=/tmp/backend", "backend="} {
		if _, _, err := parseRepoPathSpec(spec); err == nil {
			t.Fatalf("expected error for %q", spec)
		}
	}
}
