package registry

import "strings"

// Agent describes a known AI coding CLI and how to recognise it.
type Agent struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Binaries    []string `json:"binaries"`
	Color       string   `json:"color"`
	Glyph       string   `json:"glyph,omitempty"`
	Description string   `json:"description"`
	VariantOf   string   `json:"variant_of,omitempty"`
	Install     string   `json:"install,omitempty"`
	Uninstall   string   `json:"uninstall,omitempty"`
}

// UninstallCmd returns the shell command to remove the agent, falling back
// to deleting the binary itself when no managed uninstaller is known.
func (a Agent) UninstallCmd() string {
	if a.Uninstall != "" {
		return a.Uninstall
	}
	if len(a.Binaries) > 0 {
		return "rm -f \"$(command -v " + a.Binaries[0] + ")\""
	}
	return ""
}

// Known returns the built-in catalog of AI CLI agents.
func Known() []Agent {
	return []Agent{
		{ID: "claude", Name: "Claude Code", Binaries: []string{"claude"}, Color: "#D97757", Glyph: "\uf544", Description: "Anthropic's coding agent",
			Install: "brew install --cask claude", Uninstall: "brew uninstall --cask claude"},
		{ID: "codex", Name: "Codex CLI", Binaries: []string{"codex"}, Color: "#10A37F", Glyph: "\uf120", Description: "OpenAI's coding agent",
			Install: "npm install -g @openai/codex", Uninstall: "npm uninstall -g @openai/codex"},
		{ID: "gemini", Name: "Gemini CLI", Binaries: []string{"gemini"}, Color: "#4285F4", Glyph: "\uf3a5", Description: "Google's AI agent",
			Install: "npm install -g @google/gemini-cli", Uninstall: "npm uninstall -g @google/gemini-cli"},
		{ID: "opencode", Name: "OpenCode", Binaries: []string{"opencode"}, Color: "#7C86FF", Glyph: "\uf121", Description: "Open-source AI coding agent",
			Install: "npm install -g opencode-ai", Uninstall: "npm uninstall -g opencode-ai"},
		{ID: "crush", Name: "Crush", Binaries: []string{"crush"}, Color: "#FF6B6B", Glyph: "\uf004", Description: "Charm's AI coding agent",
			Install: "brew install charmbracelet/tap/crush", Uninstall: "brew uninstall charmbracelet/tap/crush"},
		{ID: "kilocode", Name: "Kilo Code", Binaries: []string{"kilocode"}, Color: "#A78BFA", Glyph: "\uf0e7", Description: "Open-source coding agent",
			Install: "npm install -g kilocode", Uninstall: "npm uninstall -g kilocode"},
		{ID: "aider", Name: "Aider", Binaries: []string{"aider"}, Color: "#F59E0B", Glyph: "\uf086", Description: "AI pair programming in your terminal",
			Install: "pipx install aider-chat", Uninstall: "pipx uninstall aider-chat"},
		{ID: "cursor", Name: "Cursor Agent", Binaries: []string{"cursor-agent"}, Color: "#E5E7EB", Glyph: "\uf246", Description: "Cursor's CLI agent",
			Install: "npm install -g @cursor-agent/cli", Uninstall: "npm uninstall -g @cursor-agent/cli"},
		{ID: "goose", Name: "Goose", Binaries: []string{"goose"}, Color: "#FF8A3D", Glyph: "\uf52d", Description: "Block's autonomous agent",
			Install: "brew install block/goose/goose", Uninstall: "brew uninstall block/goose/goose"},
		{ID: "amp", Name: "Amp", Binaries: []string{"amp"}, Color: "#F5C518", Glyph: "\uf1e6", Description: "Sourcegraph's coding agent"},
		{ID: "copilot", Name: "GitHub Copilot", Binaries: []string{"copilot"}, Color: "#58A6FF", Glyph: "\uf09b", Description: "GitHub's CLI agent",
			Install: "npm install -g @github/copilot", Uninstall: "npm uninstall -g @github/copilot"},
		{ID: "qwen", Name: "Qwen Code", Binaries: []string{"qwen"}, Color: "#615CED", Glyph: "\uf0c2", Description: "Alibaba's coding agent"},
		{ID: "droid", Name: "Factory Droid", Binaries: []string{"droid"}, Color: "#34D399", Glyph: "\uf17b", Description: "Factory's autonomous dev agent"},
		{ID: "cline", Name: "Cline", Binaries: []string{"cline"}, Color: "#4A9EFF", Glyph: "\uf0e8", Description: "Autonomous coding agent"},
		{ID: "openhands", Name: "OpenHands", Binaries: []string{"openhands"}, Color: "#FF6B4A", Glyph: "\uf256", Description: "Open-source software dev agent"},
		{ID: "mistral", Name: "Mistral CLI", Binaries: []string{"mistral"}, Color: "#FF7000", Glyph: "\uf72e", Description: "Mistral's coding CLI"},
		{ID: "hermes", Name: "Hermes", Binaries: []string{"hermes"}, Color: "#C084FC", Glyph: "\uf0e0", Description: "Local AI agent"},
		{ID: "muse", Name: "Muse", Binaries: []string{"muse"}, Color: "#F472B6", Glyph: "\uf001", Description: "Terminal AI agent"},
		{ID: "feynman", Name: "Feynman", Binaries: []string{"feynman"}, Color: "#38BDF8", Glyph: "\uf0c3", Description: "Terminal AI agent"},

		// Local variants of known agents.
		{ID: "claude-omniroute", Name: "Claude (omniroute)", Binaries: []string{"claude-omniroute"}, Color: "#D97757", Glyph: "\uf544", VariantOf: "claude", Description: "Claude Code via omniroute"},
		{ID: "claude-free", Name: "Claude (free)", Binaries: []string{"claude-free"}, Color: "#D97757", Glyph: "\uf544", VariantOf: "claude", Description: "Claude Code free tier"},
		{ID: "cursor-omniroute", Name: "Cursor (omniroute)", Binaries: []string{"cursor-omniroute"}, Color: "#E5E7EB", Glyph: "\uf246", VariantOf: "cursor", Description: "Cursor Agent via omniroute"},
		{ID: "pi-omniroute", Name: "Pi (omniroute)", Binaries: []string{"pi-omniroute"}, Color: "#FBBF24", Glyph: "\uf1d9", Description: "Pi agent via omniroute"},
	}
}

// BinIndex maps every known binary name to its agent (used by the live
// process monitor to recognise running agent processes).
func BinIndex(agents []Agent) map[string]Agent {
	m := map[string]Agent{}
	for _, a := range agents {
		for _, b := range a.Binaries {
			m[b] = a
		}
	}
	return m
}

// Normalize lower-cases and strips every non-alphanumeric rune so that
// "Claude Code", "claude-code" and "claudecode" all compare equal.
func Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Match finds agents for a user-typed query using exact, prefix then
// substring matching (in that order of precedence).
func Match(agents []Agent, query string) []Agent {
	q := Normalize(query)
	if q == "" {
		return nil
	}
	var exact, prefix, contains []Agent
	for _, a := range agents {
		cands := []string{a.ID, a.Name}
		for _, bin := range a.Binaries {
			cands = append(cands, bin)
		}
		tier := -1
		for _, c := range cands {
			nc := Normalize(c)
			if nc == q {
				tier = 0
				break
			}
			if strings.HasPrefix(nc, q) || strings.HasPrefix(q, nc) {
				if tier != 0 {
					tier = 1
				}
				continue
			}
			if tier < 0 && strings.Contains(nc, q) {
				tier = 2
			}
		}
		switch tier {
		case 0:
			exact = append(exact, a)
		case 1:
			prefix = append(prefix, a)
		case 2:
			contains = append(contains, a)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	if len(prefix) > 0 {
		return prefix
	}
	return contains
}
