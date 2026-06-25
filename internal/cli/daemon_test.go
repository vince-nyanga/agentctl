package cli

import "testing"

func TestAcquireDaemonLock(t *testing.T) {
	root := t.TempDir()
	release, err := acquireDaemonLock(root)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if _, err := acquireDaemonLock(root); err == nil {
		t.Fatalf("expected second lock to fail")
	}
	release()
	if release, err := acquireDaemonLock(root); err != nil {
		t.Fatalf("reacquire lock: %v", err)
	} else {
		release()
	}
}
