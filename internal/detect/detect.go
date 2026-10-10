package detect

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/12britz/rove/internal/registry"
)

// Install is a registry agent plus the resolved binary path (empty when
// the agent is not installed).
type Install struct {
	Agent registry.Agent
	Path  string
}

// Installed reports whether a binary was found.
func (i Install) Installed() bool { return i.Path != "" }

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07`)

func extraDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, "Library", "pnpm"),
		"/opt/homebrew/bin",
		"/usr/local/bin",
	}
	var out []string
	for _, d := range candidates {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

// Locate resolves the first available binary for an agent, honouring PATH
// order first and falling back to common install dirs.
func Locate(a registry.Agent) (string, bool) {
	extras := extraDirs()
	for _, bin := range a.Binaries {
		if p, err := exec.LookPath(bin); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs, true
			}
			return p, true
		}
		for _, d := range extras {
			p := filepath.Join(d, bin)
			if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
				return p, true
			}
		}
	}
	return "", false
}

// Scan resolves every known agent against the filesystem.
// With missing=false only installed agents are returned.
func Scan(agents []registry.Agent, missing bool) []Install {
	var out []Install
	for _, a := range agents {
		path, ok := Locate(a)
		if !ok && !missing {
			continue
		}
		out = append(out, Install{Agent: a, Path: path})
	}
	return out
}

// ProbeVersion runs `<bin> --version` with a short timeout and returns a
// cleaned, single-line version string ("" when unavailable).
func ProbeVersion(path string) string {
	if path == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb", "CLICOLOR=0")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	return CleanVersion(buf.String())
}

// CleanVersion extracts a tidy first line from raw --version output.
func CleanVersion(raw string) string {
	raw = ansiRE.ReplaceAllString(raw, "")
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len([]rune(line)) > 48 {
			line = string([]rune(line)[:45]) + "…"
		}
		return line
	}
	return ""
}

// ProbeVersions probes every install in parallel and returns id → version.
func ProbeVersions(installs []Install) map[string]string {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out = map[string]string{}
	)
	for _, inst := range installs {
		if !inst.Installed() {
			continue
		}
		wg.Add(1)
		go func(inst Install) {
			defer wg.Done()
			v := ProbeVersion(inst.Path)
			mu.Lock()
			out[inst.Agent.ID] = v
			mu.Unlock()
		}(inst)
	}
	wg.Wait()
	return out
}
