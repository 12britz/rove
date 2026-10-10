package proc

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Info is one snapshot row from `ps`.
type Info struct {
	PID     int
	PPID    int
	CPU     float64 // percent CPU
	Mem     float64 // percent physical memory
	RSS     int64   // resident set size (KB)
	TTY     string
	Start   string // raw lstart string
	Command string // first token of the command line
	Args    string // full command line
}

var psLine = regexp.MustCompile(`^(\d+)\s+(\d+)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+)\s+(\S+)\s+(\S+\s+\S+\s+\d+\s+\d\d:\d\d:\d\d\s+\d{4})\s+(.*)$`)

// Snapshot collects process info via ps.
func Snapshot() []Info {
	out, err := exec.Command("ps", "-axo",
		"pid=,ppid=,pcpu=,pmem=,rss=,tty=,lstart=,command=").Output()
	if err != nil {
		return nil
	}
	return Parse(string(out))
}

// Parse splits raw ps -axo output into rows.
func Parse(s string) []Info {
	var out []Info
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		m := psLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pid, _ := strconv.Atoi(m[1])
		ppid, _ := strconv.Atoi(m[2])
		cpu, _ := strconv.ParseFloat(m[3], 64)
		mem, _ := strconv.ParseFloat(m[4], 64)
		rss, _ := strconv.ParseInt(m[5], 10, 64)
		args := m[8]
		out = append(out, Info{
			PID: pid, PPID: ppid,
			CPU: cpu, Mem: mem, RSS: rss,
			TTY:     m[6],
			Start:   m[7],
			Command: firstWord(args),
			Args:    args,
		})
	}
	return out
}

// BinOf returns the executable basename for a process.
func BinOf(i Info) string {
	if i.Command != "" {
		return i.Command
	}
	f := strings.Fields(i.Args)
	if len(f) == 0 {
		return ""
	}
	return baseName(f[0])
}

func baseName(p string) string {
	if idx := strings.LastIndexAny(p, "/\\"); idx >= 0 {
		return p[idx+1:]
	}
	return p
}

func firstWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return baseName(f[0])
}

// Since renders a human "ses ago" duration from a ps lstart string.
func Since(raw string) string {
	t, err := time.Parse("Mon Jan 2 15:04:05 2006", raw)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h" + strconv.Itoa(int(d.Minutes())%60) + "m"
	default:
		return strconv.Itoa(int(d.Hours())/24) + "d" + strconv.Itoa(int(d.Hours())%24) + "h"
	}
}

// RSSHuman formats a KB value readably.
func RSSHuman(kb int64) string {
	switch {
	case kb >= 1024*1024:
		return trim(float64(kb)/1024/1024) + "G"
	case kb >= 1024:
		return trim(float64(kb)/1024) + "M"
	case kb >= 0:
		return strconv.FormatInt(kb, 10) + "K"
	}
	return "—"
}

func trim(f float64) string {
	s := strconv.FormatFloat(f, 'f', 1, 64)
	return strings.TrimSuffix(strings.TrimSuffix(s, "0"), ".")
}
