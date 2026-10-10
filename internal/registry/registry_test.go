package registry

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Claude Code": "claudecode",
		"claude-code": "claudecode",
		"  Aider  ":   "aider",
		"Qwen.CLI":    "qwencli",
		"MiStRaL-CLI": "mistralcli",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	agents := Known()

	exact := Match(agents, "codex")
	if len(exact) != 1 || exact[0].ID != "codex" {
		t.Fatalf("exact match failed: %+v", exact)
	}

	prefix := Match(agents, "cl")
	found := false
	for _, a := range prefix {
		if a.ID == "claude" {
			found = true
		}
	}
	if !found {
		t.Errorf("prefix match should include claude, got %+v", prefix)
	}

	sub := Match(agents, "route")
	if len(sub) == 0 {
		t.Fatalf("substring match should find omniroute variants")
	}

	if Match(agents, "") != nil {
		t.Errorf("empty query should match nothing")
	}
}

func TestInstallCommands(t *testing.T) {
	byID := map[string]Agent{}
	for _, a := range Known() {
		a := a
		byID[a.ID] = a
		if a.Install == "" && a.Uninstall == "" && len(a.Binaries) == 0 {
			t.Errorf("agent %q has no install/uninstall info", a.ID)
		}
	}
	for _, id := range []string{"codex", "claude", "gemini", "opencode"} {
		if byID[id].Install == "" {
			t.Errorf("agent %q should have an install command", id)
		}
		if byID[id].Uninstall == "" {
			t.Errorf("agent %q should have an uninstall command", id)
		}
	}
}

func TestUninstallCmdFallback(t *testing.T) {
	if got := (Agent{Binaries: []string{"hermes"}}).UninstallCmd(); got != `rm -f "$(command -v hermes)"` {
		t.Errorf("fallback uninstall = %q", got)
	}
	if (Agent{}).UninstallCmd() != "" {
		t.Errorf("agent with no binaries should have no uninstall command")
	}
	explicit := Agent{Binaries: []string{"codex"}, Uninstall: "npm uninstall -g @openai/codex"}
	if got := explicit.UninstallCmd(); got != "npm uninstall -g @openai/codex" {
		t.Errorf("explicit uninstall should win, got %q", got)
	}
}
