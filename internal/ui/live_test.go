package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/12britz/rove/internal/proc"
)

func gold(t time.Duration) string {
	return time.Now().Add(-t).Format("Mon Jan 2 15:04:05 2006")
}

func liveTestRoots() []*proc.Node {
	return []*proc.Node{
		{Info: proc.Info{PID: 100, PPID: 1, CPU: 50, Mem: 5, RSS: 2048, TTY: "ttys001", Start: gold(time.Hour), Command: "claude", Args: "claude --resume"},
			Kids: []*proc.Node{
				{Info: proc.Info{PID: 101, PPID: 100, CPU: 20, Mem: 2, RSS: 1024, TTY: "ttys001", Start: gold(time.Hour), Command: "bash", Args: "bash"}},
			}},
		{Info: proc.Info{PID: 200, PPID: 1, CPU: 5, Mem: 1, RSS: 512, TTY: "??", Start: gold(time.Hour), Command: "opencode", Args: "opencode a b"}},
	}
}

func TestNewLiveDefaults(t *testing.T) {
	m := NewLive()
	if m.interval != 2000 || m.sortBy != byCPU {
		t.Errorf("defaults wrong: interval=%d sort=%s", m.interval, m.sortBy)
	}
	if m.Init() == nil {
		t.Errorf("Init should schedule a refresh")
	}
}

func TestLiveUpdateMsg(t *testing.T) {
	m := NewLive()
	mm, cmd := m.Update(liveMsg{roots: liveTestRoots()})
	if cmd == nil {
		t.Fatalf("after liveMsg the model should reschedule a tick")
	}
	m2 := mm.(LiveModel)
	if len(m2.rows) != 3 { // 2 roots + 1 child
		t.Errorf("expected 3 rows, got %d", len(m2.rows))
	}
	if m2.cpuTotal != 75 || m2.memTotal != 8 {
		t.Errorf("totals = cpu %v mem %v, want 75/8", m2.cpuTotal, m2.memTotal)
	}
	// default sort is cpu: root claude (50) beats opencode (5) at the top
	if m2.rows[0].node.Info.PID != 100 {
		t.Errorf("cpu sort should put pid 100 first, got %d", m2.rows[0].node.Info.PID)
	}
}

func TestLiveQuit(t *testing.T) {
	m := NewLive()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatalf("q should send tea.Quit")
	}
}

func TestLiveHelpToggle(t *testing.T) {
	m := NewLive()
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if cmd != nil {
		t.Fatalf("? should not emit a command")
	}
	live := mm.(LiveModel)
	if !live.help {
		t.Errorf("? should toggle help on")
	}
	if v := live.View(); !strings.Contains(v, "stats keys") {
		t.Errorf("help view should mention stats keys")
	}
}

func TestLiveSortKeys(t *testing.T) {
	cur := NewLive()
	cur, _ = notify(cur)
	for _, tc := range []struct {
		key  rune
		want metric
	}{
		{'m', byMem}, {'p', byPID}, {'n', byName}, {'c', byCPU},
	} {
		mm, _ := cur.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(string(tc.key))})
		if mm.(LiveModel).sortBy != tc.want {
			t.Errorf("key %q should set sort %s, got %s", string(tc.key), tc.want, mm.(LiveModel).sortBy)
		}
		cur = mm.(LiveModel)
	}
}

func notify(m LiveModel) (LiveModel, tea.Cmd) {
	mm, cmd := m.Update(liveMsg{roots: liveTestRoots()})
	return mm.(LiveModel), cmd
}

func TestLiveIntervalClamp(t *testing.T) {
	m := NewLive()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if mm.(LiveModel).interval != 1500 {
		t.Errorf("interval should drop to 1500, got %d", mm.(LiveModel).interval)
	}
	for i := 0; i < 20; i++ {
		mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	}
	if mm.(LiveModel).interval != 5000 {
		t.Errorf("interval should clamp at 5000, got %d", mm.(LiveModel).interval)
	}
	for i := 0; i < 20; i++ {
		mm, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	}
	if mm.(LiveModel).interval != 500 {
		t.Errorf("interval should clamp at 500, got %d", mm.(LiveModel).interval)
	}
}

func TestLiveRefreshCmd(t *testing.T) {
	m := NewLive()
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatalf("r should trigger an immediate refresh")
	}
	// the returned command should produce a liveMsg when run
	msg := cmd()
	if _, ok := msg.(liveMsg); !ok {
		t.Errorf("refresh command should emit liveMsg, got %T", msg)
	}
	_ = mm
}

func TestLiveViewRows(t *testing.T) {
	m := NewLive()
	m.width, m.height = 100, 24
	m, _ = notify(m)
	v := m.View()
	if !strings.Contains(v, "Claude Code") || !strings.Contains(v, "OpenCode") {
		t.Errorf("view missing agent rows:\n%s", v)
	}
	if !strings.Contains(v, "bash") {
		t.Errorf("view missing child row:\n%s", v)
	}
	if !strings.Contains(v, "stats") {
		t.Errorf("view missing header:\n%s", v)
	}
}

func TestLiveDashboardPanels(t *testing.T) {
	m := NewLive()
	m.width, m.height = 100, 30
	m, _ = notify(m)
	v := m.View()
	for _, want := range []string{"PERFORMANCE", "AGENTS", "CPU", "MEM", "Claude Code"} {
		if !strings.Contains(v, want) {
			t.Errorf("dashboard view missing %q:\n%s", want, v)
		}
	}
}

func TestLiveChartsToggle(t *testing.T) {
	m := NewLive()
	m.width, m.height = 100, 30
	m, _ = notify(m)
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd != nil {
		t.Fatalf("d should not emit a command")
	}
	off := mm.(LiveModel)
	if off.charts {
		t.Fatalf("d should disable charts")
	}
	if v := off.View(); strings.Contains(v, "PERFORMANCE") {
		t.Errorf("charts should be hidden after toggle:\n%s", v)
	}
}

func TestLiveViewFitsWidth(t *testing.T) {
	m := NewLive()
	m.width, m.height = 90, 28
	m, _ = notify(m)
	for _, line := range strings.Split(m.View(), "\n") {
		if w := lipgloss.Width(line); w > 90 {
			t.Errorf("line exceeds terminal width (%d > 90): %q", w, line)
		}
	}
}

func TestVGraphShape(t *testing.T) {
	vals := []float64{10, 40, 80, 100, 60, 30, 5}
	g := vgraph(vals, 100, 12, 4, "cpu")
	if len(g) != 4 {
		t.Fatalf("vgraph should return h rows, got %d", len(g))
	}
	for i, line := range g {
		if w := lipgloss.Width(line); w != 12 {
			t.Errorf("row %d width = %d, want 12", i, w)
		}
	}
	if !strings.Contains(g[0], "█") {
		t.Errorf("tallest column should reach the top row:\n%s", strings.Join(g, "\n"))
	}
}

func TestSparklineWidth(t *testing.T) {
	s := sparkline([]float64{1, 2, 3, 4, 5}, 10, "cpu")
	if w := lipgloss.Width(s); w != 10 {
		t.Errorf("sparkline width = %d, want 10", w)
	}
}

func TestLiveTableText(t *testing.T) {
	m := NewLive()
	out := m.TableText(100)
	if !strings.Contains(out, "PID") || !strings.Contains(out, "AGENT") {
		t.Errorf("table missing header:\n%s", out)
	}
	if !strings.Contains(out, "sessions") {
		t.Errorf("table missing summary:\n%s", out)
	}
}

func TestMeter(t *testing.T) {
	bar := meter(100, 6, true)
	if !strings.Contains(bar, "█") {
		t.Errorf("full meter should have blocks: %q", bar)
	}
	bar0 := meter(0, 6, true)
	if !strings.Contains(bar0, "░") {
		t.Errorf("empty meter should show only blanks: %q", bar0)
	}
	if meter(-1, 6, true) != meter(0, 6, true) {
		t.Errorf("negative values should clamp to zero")
	}
	if meter(150, 6, true) != meter(100, 6, true) {
		t.Errorf("values above 100 should clamp")
	}
}
