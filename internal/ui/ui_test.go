package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/12britz/rove/internal/detect"
	"github.com/12britz/rove/internal/registry"
)

func TestParseHex(t *testing.T) {
	r, g, b := parseHex("#D97757")
	if r != 0xD9 || g != 0x77 || b != 0x57 {
		t.Errorf("parseHex = %d,%d,%d", r, g, b)
	}
	if r, g, b := parseHex("nope"); r != 0 || g != 0 || b != 0 {
		t.Errorf("invalid hex should parse as black, got %d,%d,%d", r, g, b)
	}
}

func TestContrast(t *testing.T) {
	if Contrast("#D97757") != lipgloss.Color("#FFFFFF") {
		t.Errorf("dark background should give white foreground")
	}
	if Contrast("#E5E7EB") != lipgloss.Color("#141414") {
		t.Errorf("light background should give dark foreground")
	}
}

func TestGradient(t *testing.T) {
	out := Gradient("rove", "#D97757", "#A78BFA")
	if out != "rove" && !strings.Contains(out, "\x1b[") {
		t.Errorf("gradient should preserve the text, got %q", out)
	}
}

func TestTableRenders(t *testing.T) {
	agents := registry.Known()
	installs := []detect.Install{{
		Agent: agents[0],
		Path:  "/usr/local/bin/claude",
	}}
	out := Table(installs, map[string]string{"claude": "2.0.14"}, 100)
	if !strings.Contains(out, "Claude Code") || !strings.Contains(out, "2.0.14") {
		t.Errorf("table missing expected content:\n%s", out)
	}
}

func TestTableBorders(t *testing.T) {
	agents := registry.Known()
	installs := []detect.Install{
		{Agent: agents[0], Path: "/usr/local/bin/claude"},
		{Agent: agents[1], Path: "/usr/local/bin/codex"},
	}
	out := Table(installs, map[string]string{"claude": "2.0.14", "codex": "0.25"}, 100)
	for _, glyph := range []string{"╭", "╮", "╰", "╯", "│", "─"} {
		if !strings.Contains(out, glyph) {
			t.Errorf("bordered table missing %q:\n%s", glyph, out)
		}
	}
	for _, hdr := range []string{"AGENT", "BINARY", "VERSION", "PATH"} {
		if !strings.Contains(out, hdr) {
			t.Errorf("bordered table missing header %q:\n%s", hdr, out)
		}
	}
	lines := strings.Split(out, "\n")
	// top + header + divider + N rows + bottom + blank + legend
	if len(lines) != 8 {
		t.Errorf("unexpected line count %d:\n%s", len(lines), out)
	}
	if len(strings.Split(out, "╮")) != 2 {
		t.Errorf("expected exactly one top border, got:\n%s", out)
	}
}

func TestTableNarrowFallsBack(t *testing.T) {
	agents := registry.Known()
	installs := []detect.Install{{Agent: agents[0], Path: "/usr/local/bin/claude"}}
	out := Table(installs, map[string]string{}, 20)
	if strings.Contains(out, "╭") {
		t.Errorf("narrow terminal should fall back to plain rows:\n%s", out)
	}
	if !strings.Contains(out, "Claude Code") {
		t.Errorf("fallback should still list the agent:\n%s", out)
	}
}

func testInstall() []detect.Install {
	agents := registry.Known()
	return []detect.Install{
		{Agent: agents[0], Path: "/usr/bin/claude"}, // claude
		{Agent: agents[4], Path: "/usr/bin/crush"},  // crush
	}
}

func testMixed() []detect.Install {
	agents := registry.Known()
	return []detect.Install{
		{Agent: agents[0], Path: "/usr/bin/claude"}, // claude (installed)
		{Agent: agents[2], Path: ""},                // gemini (missing)
	}
}

func TestModelEnterInstalledLaunches(t *testing.T) {
	m := New(testInstall())
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Action() != ActionLaunch {
		t.Errorf("enter on installed agent should launch, got %q", m.Action())
	}
	if cmd == nil {
		t.Errorf("enter should send tea.Quit, got nil cmd")
	}
	if _, ok := m.Choice(); !ok {
		t.Errorf("enter should select the first agent")
	}
}

func TestModelEnterMissingInstalls(t *testing.T) {
	m := New(testMixed())
	// cursor starts on claude (installed); move down onto gemini (missing)
	m, _ = upd(m, tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Action() != ActionInstall {
		t.Errorf("enter on missing agent should install, got %q", m.Action())
	}
	if cmd == nil {
		t.Errorf("confirm should send tea.Quit")
	}
	if _, ok := m.Choice(); !ok {
		t.Errorf("enter should select the missing agent")
	}
}

func TestModelUninstallConfirm(t *testing.T) {
	m := New(testMixed()) // cursor on claude (installed)
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Fatalf("first u should not quit (got cmd %v)", cmd != nil)
	}
	if m.Action() != ActionLaunch {
		t.Errorf("before confirming, action should still be launch")
	}
	if !m.confirming {
		t.Errorf("first u should arm the confirm state")
	}
	if v := m.View(); !strings.Contains(v, "press u again") {
		t.Errorf("confirm bar missing from view:\n%s", v)
	}
	if _, ok := m.Choice(); ok {
		t.Errorf("confirm should not choose yet")
	}

	m, cmd = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd == nil {
		t.Fatalf("second u should quit")
	}
	if m.Action() != ActionUninstall {
		t.Errorf("second u should request uninstall, got %q", m.Action())
	}
}

func TestModelUninstallMoveCancels(t *testing.T) {
	m := New(testMixed())
	m, _ = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if !m.confirming {
		t.Fatalf("expected confirm armed")
	}
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyDown}) // move away
	if cmd != nil {
		t.Fatalf("moving should not emit a command")
	}
	if m.confirming {
		t.Errorf("moving should cancel the confirm state")
	}
	m, cmd = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Errorf("without confirmation, u should not quit")
	}
}

func TestModelUninstallEscCancels(t *testing.T) {
	m := New(testMixed())
	m, _ = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("esc during confirm should not quit")
	}
	if m.confirming {
		t.Errorf("esc should cancel confirm")
	}
}

func TestModelQuit(t *testing.T) {
	m := New(testInstall())
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if _, ok := m.Choice(); !ok {
		t.Errorf("enter should select the first agent")
	}
	if cmd == nil {
		t.Errorf("enter should send tea.Quit, got nil cmd")
	}
}

func TestModelQuitKey(t *testing.T) {
	m := New(testInstall())
	m, cmd := upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if _, ok := m.Choice(); ok {
		t.Errorf("q should not select an agent")
	}
	if cmd == nil {
		t.Errorf("q should send tea.Quit, got nil cmd")
	}
}

func TestModelFilter(t *testing.T) {
	m := New(testInstall())
	m, _ = upd(m, KeySlash())
	m, _ = upd(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("cru")})
	m, _ = upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	choice, ok := m.Choice()
	if !ok || choice.Agent.ID != "crush" {
		t.Errorf("filter+enter should select crush, got %+v", choice)
	}
}

func TestModelVersionMsg(t *testing.T) {
	m := New(testInstall())
	m, _ = upd(m, versionMsg{id: "claude", version: "2.0.14"})
	if v := m.versionOf(testInstall()[0]); v == "…" || v != "2.0.14" {
		t.Errorf("version message should be stored, got %q", v)
	}
}

// KeySlash produces the "/" key message used to start filtering.
func KeySlash() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
}

func TestBanner(t *testing.T) {
	lines := bannerLines()
	if len(lines) != 5 {
		t.Fatalf("banner should be 5 lines, got %d", len(lines))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"█████", "████", "█"} {
		if !strings.Contains(joined, want) {
			t.Errorf("banner missing %q:\n%s", want, joined)
		}
	}
	for _, line := range lines {
		if w := lipgloss.Width(line); w < 22 {
			t.Errorf("banner line unexpectedly narrow (%d): %q", w, line)
		}
	}
}

func TestBannerView(t *testing.T) {
	glyph := "█████" // top line of the R wordmark
	m := New(testInstall())
	m.height = bannerMinHeight + 10
	view := m.View()
	if !strings.Contains(view, glyph) {
		t.Errorf("tall terminal should render the ROVE banner header")
	}
	if !strings.Contains(view, "Claude Code") {
		t.Errorf("banner view missing the agent list")
	}
	m2 := New(testInstall())
	m2.height = bannerMinHeight - 2
	if strings.Contains(m2.View(), glyph) {
		t.Errorf("short terminal should fall back to the compact header")
	}
}

func upd(m Model, msg tea.Msg) (Model, tea.Cmd) {
	mm, cmd := m.Update(msg)
	return mm.(Model), cmd
}

func TestGlyphModes(t *testing.T) {
	claude := registry.Known()[0]
	if got := glyphFor(claude, IconsNone); got != "●" {
		t.Errorf("IconsNone should give a plain bullet, got %q", got)
	}
	if got := glyphFor(claude, IconsASCII); got != "*" {
		t.Errorf("claude ASCII glyph should be *, got %q", got)
	}
	got := glyphFor(claude, IconsNerd)
	if lipgloss.Width(claude.Glyph) == 1 {
		if got != claude.Glyph {
			t.Errorf("nerd glyph = %q, want %q", got, claude.Glyph)
		}
	} else if got != "*" {
		t.Errorf("unsafe-width nerd glyph should fall back to ASCII, got %q", got)
	}
}

func TestBannerPhaseZeroMatchesStatic(t *testing.T) {
	base := strings.Join(bannerLines(), "\n")
	if p0 := strings.Join(bannerLinesPhase(0), "\n"); p0 != base {
		t.Errorf("phase 0 banner should equal the static banner")
	}
	static := bannerLines()
	for _, phase := range []int{1, 3, 7} {
		lines := bannerLinesPhase(phase)
		if len(lines) != len(static) {
			t.Fatalf("phase %d changed line count to %d", phase, len(lines))
		}
		for i := range lines {
			if lipgloss.Width(lines[i]) != lipgloss.Width(static[i]) {
				t.Errorf("phase %d line %d width drifted", phase, i)
			}
		}
	}
}

func TestAnimateOption(t *testing.T) {
	if cmd := New(testInstall()).bannerCmd(); cmd == nil {
		t.Errorf("animation is on by default, expected a tick command")
	}
	if cmd := New(testInstall(), WithAnimate(false)).bannerCmd(); cmd != nil {
		t.Errorf("WithAnimate(false) should disable the banner tick")
	}
}

func TestModelWideLayout(t *testing.T) {
	m := New(testInstall())
	m.width, m.height = 140, 40
	view := m.View()
	if !strings.Contains(view, "AGENTS") {
		t.Errorf("wide layout should frame the list in an AGENTS panel")
	}
	if !strings.Contains(view, "CLAUDE CODE") {
		t.Errorf("wide layout should show the agent detail card title:\n%s", view)
	}
	for _, want := range []string{"status", "binary", "path"} {
		if !strings.Contains(view, want) {
			t.Errorf("detail card missing %q field:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Errorf("wide layout should use rounded panels:\n%s", view)
	}
}

func TestModelIconModeNoneView(t *testing.T) {
	m := New(testInstall(), WithIconMode(IconsNone))
	m.width, m.height = 80, 12
	view := m.View()
	if !strings.Contains(view, "●") {
		t.Errorf("IconsNone view should use the plain bullet marker:\n%s", view)
	}
}

func TestModelFooterCounts(t *testing.T) {
	m := New(testMixed())
	m.width, m.height = 130, 12
	view := m.View()
	if !strings.Contains(view, "1 installed") || !strings.Contains(view, "2 total") {
		t.Errorf("footer should report installed/total counts:\n%s", view)
	}
}
