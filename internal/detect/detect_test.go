package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/12britz/rove/internal/registry"
)

func TestCleanVersion(t *testing.T) {
	raw := "\x1b[1m\x1b[32m2.0.14 (Claude Code)\x1b[0m\ncopyright line\n"
	got := CleanVersion(raw)
	if got != "2.0.14 (Claude Code)" {
		t.Errorf("CleanVersion = %q", got)
	}
	if CleanVersion("\n\n\n") != "" {
		t.Errorf("blank input should give empty version")
	}
}

func TestLocateEnv(t *testing.T) {
	a := registry.Agent{ID: "env-test", Binaries: []string{"rare-env-bin-does-not-exist-12345"}}
	if _, ok := Locate(a); ok {
		t.Errorf("phantom binary should not be found")
	}
	a = registry.Agent{ID: "env-test", Binaries: []string{os.Args[0]}}
	path, ok := Locate(a)
	if !ok {
		t.Fatalf("test binary %q should be located", os.Args[0])
	}
	if filepath.Base(path) != filepath.Base(os.Args[0]) {
		t.Errorf("unexpected path %q", path)
	}
}

func TestScanMissing(t *testing.T) {
	agents := []registry.Agent{{ID: "x", Binaries: []string{"no-such-binary-zzz"}}}
	if got := Scan(agents, false); len(got) != 0 {
		t.Errorf("Scan(missing=false) should be empty, got %+v", got)
	}
	if got := Scan(agents, true); len(got) != 1 || got[0].Installed() {
		t.Errorf("Scan(missing=true) should mark not-installed, got %+v", got)
	}
}
