package proc

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	raw := "1234 567 1.5 2.3 456789 ttys003 Sat Oct 10 10:00:00 2026 /usr/local/bin/claude --resume abc"
	infos := Parse(raw)
	if len(infos) != 1 {
		t.Fatalf("expected 1 row, got %d", len(infos))
	}
	in := infos[0]
	if in.PID != 1234 || in.PPID != 567 {
		t.Errorf("pid/ppid = %d/%d", in.PID, in.PPID)
	}
	if in.CPU != 1.5 || in.Mem != 2.3 {
		t.Errorf("cpu/mem = %v/%v", in.CPU, in.Mem)
	}
	if in.RSS != 456789 {
		t.Errorf("rss = %d", in.RSS)
	}
	if in.TTY != "ttys003" {
		t.Errorf("tty = %q", in.TTY)
	}
	if in.Start != "Sat Oct 10 10:00:00 2026" {
		t.Errorf("start = %q", in.Start)
	}
	if in.Command != "claude" || BinOf(in) != "claude" {
		t.Errorf("command/bin = %q/%q", in.Command, BinOf(in))
	}
}

func TestParseSkipsJunk(t *testing.T) {
	infos := Parse("not a ps line\n")
	if len(infos) != 0 {
		t.Fatalf("junk should be skipped, got %d", len(infos))
	}
}

func TestSince(t *testing.T) {
	now := time.Now().Format("Mon Jan 2 15:04:05 2006")
	cases := []struct {
		raw  string
		want string
	}{
		{"bogus", ""},
		{now, "0s"},
	}
	for _, c := range cases {
		if got := Since(c.raw); got != c.want {
			t.Errorf("Since(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestRSSHuman(t *testing.T) {
	cases := []struct {
		kb   int64
		want string
	}{
		{512, "512K"},
		{2048, "2M"},
		{1024 * 1024, "1G"},
		{600 * 1024, "600M"},
	}
	for _, c := range cases {
		if got := RSSHuman(c.kb); got != c.want {
			t.Errorf("RSSHuman(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}

func TestRoots(t *testing.T) {
	infos := []Info{
		{PID: 100, PPID: 1, Command: "claude", Args: "claude --resume"}, // root
		{PID: 101, PPID: 100, Command: "bash", Args: "bash"},            // child
		{PID: 102, PPID: 100, Command: "node", Args: "node tool.js"},    // child
		{PID: 103, PPID: 1, Command: "codex", Args: "codex"},            // second root
		{PID: 104, PPID: 103, Command: "node", Args: "node child.js"},   // child of codex
		{PID: 105, PPID: 1, Command: "unknown", Args: "unknown"},        // unknown, not a root
		{PID: 106, PPID: 1, Command: "ps", Args: "ps -axo"},             // transient ps, excluded
	}
	known := map[string]bool{"claude": true, "codex": true}
	roots := Roots(infos, known, 999)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	if roots[0].Info.PID != 100 || roots[1].Info.PID != 103 {
		t.Errorf("roots not pid-ordered: first=%d second=%d", roots[0].Info.PID, roots[1].Info.PID)
	}
	kids := roots[0].Kids
	if len(kids) != 2 {
		t.Fatalf("claude root should have 2 kids, got %d", len(kids))
	}
	if kids[0].Info.PID != 101 || kids[1].Info.PID != 102 {
		t.Errorf("claude kids = %d,%d", kids[0].Info.PID, kids[1].Info.PID)
	}
	if len(roots[1].Kids) != 1 || roots[1].Kids[0].Info.PID != 104 {
		t.Errorf("codex kids wrong: %d", len(roots[1].Kids))
	}
}

func TestRootsDedupesPIDs(t *testing.T) {
	// macOS ps may emit the same PID twice; the row matching a known bin wins.
	infos := []Info{
		{PID: 100, PPID: 1, Command: "opencode", Args: "opencode"},
		{PID: 100, PPID: 1, Command: "OpenCode", Args: "OpenCode"},
		{PID: 101, PPID: 100, Command: "bash", Args: "bash"},
	}
	known := map[string]bool{"opencode": true, "claude": true}
	roots := Roots(infos, known, 999)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root after dedupe, got %d", len(roots))
	}
	if BinOf(roots[0].Info) != "opencode" {
		t.Errorf("duplicate row should prefer known bin, got %q", BinOf(roots[0].Info))
	}
	if len(roots[0].Kids) != 1 {
		t.Errorf("root should keep its child, got %d", len(roots[0].Kids))
	}
}

func TestRootsIgnoresSelfParent(t *testing.T) {
	infos := []Info{
		{PID: 100, PPID: 100, Command: "claude", Args: "claude"}, // self-parenting
		{PID: 101, PPID: 100, Command: "bash", Args: "bash"},     // would be child
	}
	known := map[string]bool{"claude": true}
	roots := Roots(infos, known, 999)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if len(roots[0].Kids) != 1 {
		t.Fatalf("self-parenting root should keep a real child (101), got %d kids", len(roots[0].Kids))
	}
	if roots[0].Kids[0].Info.PID == roots[0].Info.PID {
		t.Errorf("root must not list itself as a child")
	}
}

func TestRootsExcludesSelfAndPS(t *testing.T) {
	infos := []Info{
		{PID: 999, PPID: 1, Command: "claude", Args: "claude"}, // rove itself
		{PID: 555, PPID: 1, Command: "ps", Args: "ps -axo"},    // transient helper
	}
	known := map[string]bool{"claude": true}
	if roots := Roots(infos, known, 999); len(roots) != 0 {
		t.Errorf("self and ps processes must not appear, got %d", len(roots))
	}
}

func TestSessionSums(t *testing.T) {
	n := &Node{Info: Info{PID: 1, CPU: 10, Mem: 1},
		Kids: []*Node{
			{Info: Info{PID: 2, CPU: 20, Mem: 2}},
			{Info: Info{PID: 3, CPU: 30, Mem: 3}},
		}}
	if got := SessionCPU(n); got != 60 {
		t.Errorf("SessionCPU = %v, want 60", got)
	}
	if got := SessionMem(n); got != 6 {
		t.Errorf("SessionMem = %v, want 6", got)
	}
}
