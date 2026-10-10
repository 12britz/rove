package ui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/12britz/rove/internal/proc"
	"github.com/12britz/rove/internal/registry"
)

type liveMsg struct{ roots []*proc.Node }

type metric string

const (
	byCPU  metric = "cpu"
	byMem  metric = "mem"
	byName metric = "name"
	byPID  metric = "pid"
)

// LiveModel is the live agent-process monitor.
type LiveModel struct {
	rows     []drow
	roots    []*proc.Node
	sortBy   metric
	interval int // refresh ms
	cursor   int
	offset   int
	help     bool
	charts   bool
	quitting bool
	width    int
	height   int
	cpuTotal float64
	memTotal float64
	cpuHist  []float64
	memHist  []float64
	lastTick string
}

type drow struct {
	node  *proc.Node
	depth int
}

// histLen is how many samples the sparkline history keeps per metric.
const histLen = 90

// agentAgg is a per-agent roll-up of the running sessions.
type agentAgg struct {
	name  string
	color string
	cpu   float64
	mem   float64
	rss   int64
	procs int
}

// NewLive builds the live monitor model.
func NewLive() LiveModel {
	return LiveModel{sortBy: byCPU, interval: 2000, charts: true, width: 100, height: 24}
}

func (m LiveModel) Init() tea.Cmd { return m.refreshCmd() }

func (m LiveModel) refreshCmd() tea.Cmd {
	bins := registry.BinIndex(registry.Known())
	known := make(map[string]bool, len(bins))
	for b := range bins {
		known[b] = true
	}
	return func() tea.Msg {
		roots := proc.Roots(proc.Snapshot(), known, os.Getpid())
		return liveMsg{roots: roots}
	}
}

func (m LiveModel) tickAfter() tea.Cmd {
	return tea.Tick(time.Duration(m.interval)*time.Millisecond, func(time.Time) tea.Msg {
		return refreshSignal{}
	})
}

type refreshSignal struct{}

func (m LiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case liveMsg:
		m.roots = msg.roots
		m.recompute()
		m.pushHist()
		m.lastTick = time.Now().Format("15:04:05")
		return m, m.tickAfter()
	case refreshSignal:
		return m, m.refreshCmd()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil
	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m LiveModel) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit
	case "?":
		m.help = !m.help
	case "d", "D":
		m.charts = !m.charts
		m.ensureVisible()
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "g":
		m.cursor = 0
		m.ensureVisible()
	case "G":
		m.cursor = len(m.rows) - 1
		m.ensureVisible()
	case "c", "C":
		m.setSort(byCPU)
	case "m", "M":
		m.setSort(byMem)
	case "n", "N":
		m.setSort(byName)
	case "p", "P":
		m.setSort(byPID)
	case "r":
		return m, m.refreshCmd()
	case "+", "=":
		m.interval = clampInt(m.interval+500, 500, 5000)
		return m, m.tickAfter()
	case "-", "_":
		m.interval = clampInt(m.interval-500, 500, 5000)
		return m, m.tickAfter()
	}
	return m, nil
}

func (m *LiveModel) setSort(s metric) {
	m.sortBy = s
	m.recompute()
	m.ensureVisible()
}

func (m *LiveModel) move(d int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	m.ensureVisible()
}

func (m *LiveModel) ensureVisible() {
	avail := m.rowsAvail()
	if m.cursor >= len(m.rows) {
		m.cursor = maxInt(0, len(m.rows)-1)
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+avail {
		m.offset = m.cursor - avail + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m LiveModel) rowsAvail() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	avail := h - 4 - m.panelHeights() // header + args + divider + footer
	if avail < 3 {
		avail = 3
	}
	return avail
}

// recompute flattens the process tree into display rows and sums resources.
func (m *LiveModel) recompute() {
	sort.Slice(m.roots, func(i, j int) bool {
		return metricVal(m.roots[i].Info, m.sortBy) > metricVal(m.roots[j].Info, m.sortBy)
	})
	rows := make([]drow, 0, len(m.roots)*2)
	m.cpuTotal, m.memTotal = 0, 0
	for _, r := range m.roots {
		rows = append(rows, drow{node: r, depth: 0})
		m.cpuTotal += proc.SessionCPU(r)
		m.memTotal += proc.SessionMem(r)
		sort.Slice(r.Kids, func(i, j int) bool {
			return metricVal(r.Kids[i].Info, m.sortBy) > metricVal(r.Kids[j].Info, m.sortBy)
		})
		for _, k := range r.Kids {
			rows = append(rows, drow{node: k, depth: minInt(2, 1)})
		}
	}
	m.rows = rows
	if m.cursor >= len(m.rows) {
		m.cursor = maxInt(0, len(m.rows)-1)
	}
}

// pushHist appends the latest totals to the sparkline histories.
func (m *LiveModel) pushHist() {
	m.cpuHist = append(m.cpuHist, m.cpuTotal)
	m.memHist = append(m.memHist, m.memTotal)
	if len(m.cpuHist) > histLen {
		m.cpuHist = m.cpuHist[len(m.cpuHist)-histLen:]
	}
	if len(m.memHist) > histLen {
		m.memHist = m.memHist[len(m.memHist)-histLen:]
	}
}

func (m LiveModel) hist(kind string) []float64 {
	if kind == "mem" {
		return m.memHist
	}
	return m.cpuHist
}

// showCharts reports whether the dashboard panels should be drawn, honouring
// the user toggle and collapsing automatically on short terminals.
func (m LiveModel) showCharts() bool {
	if !m.charts {
		return false
	}
	return m.height <= 0 || m.height >= 20
}

// agentRows rolls the root sessions up by agent, busiest first.
func (m LiveModel) agentRows() []agentAgg {
	by := map[string]*agentAgg{}
	var order []string
	for _, r := range m.roots {
		name, color := agentLabel(r.Info)
		a := by[name]
		if a == nil {
			a = &agentAgg{name: name, color: color}
			by[name] = a
			order = append(order, name)
		}
		a.cpu += proc.SessionCPU(r)
		a.mem += proc.SessionMem(r)
		a.rss += nodeRSS(r)
		a.procs += countProcs(r)
	}
	aggs := make([]agentAgg, 0, len(order))
	for _, n := range order {
		aggs = append(aggs, *by[n])
	}
	sort.Slice(aggs, func(i, j int) bool { return aggs[i].cpu > aggs[j].cpu })
	return aggs
}

func nodeRSS(n *proc.Node) int64 {
	t := n.Info.RSS
	for _, k := range n.Kids {
		t += nodeRSS(k)
	}
	return t
}

func countProcs(n *proc.Node) int {
	t := 1
	for _, k := range n.Kids {
		t += countProcs(k)
	}
	return t
}

// panelHeights sums the vertical space the dashboard panels occupy.
func (m LiveModel) panelHeights() int {
	if !m.showCharts() {
		return 0
	}
	return 4 + m.agentsPanelHeight()
}

func (m LiveModel) agentsPanelHeight() int {
	aggs := m.agentRows()
	if len(aggs) == 0 {
		return 0
	}
	return minInt(len(aggs), 4) + 2
}

// perfPanel renders the CPU/MEM gauges with sparkline history.
func (m LiveModel) perfPanel(w int) []string {
	inner := w - 2
	g := clampInt(inner/3, 12, 28)
	spark := inner - (15 + g)
	if spark < 6 {
		spark = 0
		if g = inner - 15; g < 8 {
			g = 8
		}
	}
	row := func(label string, pct float64, kind string) string {
		col := kindColor(kind, pct)
		lbl := lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Bold(true).
			Render(padRight(label, 4))
		pctS := lipgloss.NewStyle().Foreground(lipgloss.Color(col)).
			Render(fmt.Sprintf("%5.1f%%", pct))
		s := " " + lbl + pctS + "  " + gauge(pct, g, col)
		if spark > 0 {
			s += "  " + sparkline(m.hist(kind), spark, kind)
		}
		return s
	}
	body := []string{
		row("CPU", m.cpuTotal, "cpu"),
		row("MEM", m.memTotal, "mem"),
	}
	return panel("PERFORMANCE", body, w, Accent)
}

// agentsPanel renders a per-agent usage breakdown as horizontal bars.
func (m LiveModel) agentsPanel(w int) []string {
	aggs := m.agentRows()
	if len(aggs) == 0 {
		return nil
	}
	const maxRows = 4
	inner := w - 2
	nameW := clampInt(inner/4, 10, 20)
	barW := inner - nameW - 6 - 8 - 10 // name + pct + rss + procs + gaps
	if barW < 6 {
		barW = 6
	}
	maxCPU := 0.0
	for _, a := range aggs {
		if a.cpu > maxCPU {
			maxCPU = a.cpu
		}
	}
	if maxCPU < 1 {
		maxCPU = 1
	}
	body := make([]string, 0, maxRows+1)
	for i, a := range aggs {
		if i >= maxRows {
			body = append(body, dim.Render(fmt.Sprintf(" +%d more agent(s)", len(aggs)-maxRows)))
			break
		}
		nameS := lipgloss.NewStyle().Foreground(lipgloss.Color(a.color)).Bold(true).
			Render(fit(a.name, nameW))
		bar := gauge(a.cpu/maxCPU*100, barW, kindColor("cpu", a.cpu))
		pctS := dim.Render(fmt.Sprintf("%5.1f%%", a.cpu))
		rssS := faint.Render(padLeft(proc.RSSHuman(a.rss), 7))
		procS := faint.Render(fmt.Sprintf("%2dp", a.procs))
		body = append(body, nameS+" "+bar+" "+pctS+" "+rssS+" "+procS)
	}
	return panel("AGENTS", body, w, Success)
}

func metricVal(in proc.Info, m metric) float64 {
	switch m {
	case byMem:
		return in.Mem
	case byPID:
		return float64(-in.PID)
	case byName:
		return 0
	default:
		return in.CPU
	}
}

func (m LiveModel) View() string {
	if m.quitting {
		return ""
	}
	w := maxInt(m.width, 60)
	var sb strings.Builder

	bolt := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Render("⚡")
	title := Gradient("rove", "#FF6B6B", "#8E7BFF")
	meta := lipgloss.NewStyle().Foreground(Dim).Render(
		fmt.Sprintf("· stats · %d running · sort %s", len(m.roots), m.sortBy))
	refresh := lipgloss.NewStyle().Foreground(Faint).Render(
		"refresh " + refSecs(m.interval) + " · " + m.lastTick)
	left := bolt + " " + title + " " + meta
	gap := w - lipgloss.Width(left) - lipgloss.Width(refresh)
	if gap < 1 {
		gap = 1
	}
	sb.WriteString(left + strings.Repeat(" ", gap) + refresh)
	sb.WriteString("\n")

	if m.help {
		sb.WriteString(m.liveHelp())
		sb.WriteString(lipgloss.NewStyle().Foreground(Faint).Render(strings.Repeat("─", w)) + "\n")
		sb.WriteString(m.footer(w))
		return sb.String()
	}

	if m.showCharts() {
		for _, l := range m.perfPanel(w) {
			sb.WriteString(l + "\n")
		}
		for _, l := range m.agentsPanel(w) {
			sb.WriteString(l + "\n")
		}
	}

	avail := m.rowsAvail()
	end := minInt(m.offset+avail, len(m.rows))
	for i := m.offset; i < end; i++ {
		sb.WriteString(m.renderRow(m.rows[i], i == m.cursor, w))
		sb.WriteString("\n")
	}
	if len(m.rows) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(Faint).Render(
			"  no AI CLI agents running\n"))
	}

	if len(m.rows) > 0 && m.cursor < len(m.rows) {
		args := m.rows[m.cursor].node.Info.Args
		sb.WriteString(lipgloss.NewStyle().Foreground(Faint).Render(
			truncate(args, maxInt(20, w-2))) + "\n")
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(Faint).Render(strings.Repeat("─", w)) + "\n")
	sb.WriteString(m.footer(w))
	return sb.String()
}

func refSecs(ms int) string {
	return strconv.FormatFloat(float64(ms)/1000, 'f', 1, 64) + "s"
}

func (m LiveModel) liveHelp() string {
	s := lipgloss.NewStyle().Foreground(Accent).Bold(true).Render
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render
	d := lipgloss.NewStyle().Foreground(Dim).Render
	return "\n  " + s("stats keys") + "\n" +
		"  " + k("j k ↑ ↓") + " " + d("move") + "     " + k("c m n p") + " " + d("sort by cpu·mem·name·pid") + "\n" +
		"  " + k("d") + " " + d("toggle charts") + "   " + k("+ −") + " " + d("refresh speed") + "   " + k("r") + " " + d("refresh now") + "    " + k("q esc") + " " + d("quit") + "\n\n"
}

func (m LiveModel) footer(w int) string {
	peak := 0.0
	for _, v := range m.cpuHist {
		if v > peak {
			peak = v
		}
	}
	left := lipgloss.NewStyle().Foreground(Dim).Render(fmt.Sprintf(
		"cpu %5.1f%%  mem %5.1f%%  peak cpu %5.1f%%", m.cpuTotal, m.memTotal, peak))
	keys := lipgloss.NewStyle().Foreground(Faint).Render(
		"d charts   c cpu m mem n name p pid   + − speed   r refresh   ? help   q quit")
	gap := w - lipgloss.Width(left) - lipgloss.Width(keys)
	if gap < 2 {
		return keys
	}
	return left + strings.Repeat(" ", gap) + keys
}

func (m LiveModel) renderRow(r drow, sel bool, w int) string {
	in := r.node.Info
	name, color := agentLabel(in)

	pid := padLeft(strconv.Itoa(in.PID), 6)
	cpu := padLeft(num(in.CPU), 5)
	mem := padLeft(num(in.Mem), 5)
	rss := padLeft(proc.RSSHuman(in.RSS), 7)

	indent := strings.Repeat(" ", r.depth*2)
	mark := ""
	if r.depth > 0 {
		mark = lipgloss.NewStyle().Foreground(Faint).Render("└ ")
	}

	var nameS string
	if r.depth == 0 {
		nameS = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(name)
	} else {
		nameS = lipgloss.NewStyle().Foreground(Faint).Render(procBin(in))
	}
	nameS = padRight(nameS, maxInt(18, m.width/5))

	cpuMeter := meter(in.CPU, 6, true)
	memMeter := meter(in.Mem, 6, false)

	line := fmt.Sprintf("%s%s%s %s %s%s %s%s %s  %s  %s",
		indent, mark, pid, cpu, cpuMeter, mem, memMeter, rss, nameS,
		lipgloss.NewStyle().Foreground(Dim).Render(padRight(in.TTY, 8)),
		lipgloss.NewStyle().Foreground(Faint).Render(padLeft(proc.Since(in.Start), 6)))
	if lipgloss.Width(line) > w {
		line = truncate(line, w)
	}
	return line
}

var binIndexOnce = registry.BinIndex(registry.Known())

func agentLabel(in proc.Info) (string, string) {
	if a, ok := binIndexOnce[procBin(in)]; ok {
		return a.Name, a.Color
	}
	return title(procBin(in)), "#7C86FF"
}

func procBin(in proc.Info) string { return proc.BinOf(in) }

// TableText renders a one-shot snapshot for non-TTY usage.
func (m *LiveModel) TableText(width int) string {
	bins := registry.BinIndex(registry.Known())
	known := make(map[string]bool, len(bins))
	for b := range bins {
		known[b] = true
	}
	m.roots = proc.Roots(proc.Snapshot(), known, os.Getpid())
	m.width, m.height = width, 32
	m.lastTick = time.Now().Format("15:04:05")
	m.recompute()
	var sb strings.Builder
	for _, l := range bannerLines() {
		sb.WriteString(strings.Repeat(" ", maxInt(0, (width-lipgloss.Width(l))/2)) + l + "\n")
	}
	left := lipgloss.NewStyle().Foreground(Dim).Render(fmt.Sprintf(
		"cpu %5.1f%% · mem %5.1f%% · %d sessions\n", m.cpuTotal, m.memTotal, len(m.roots)))
	sb.WriteString(left)
	sb.WriteString(m.liveTable(width))
	return strings.TrimRight(sb.String(), "\n")
}

// liveTable renders the session rows as a bordered table.
func (m *LiveModel) liveTable(width int) string {
	rows := m.rows
	if width <= 60 {
		// too narrow for bordered cells: fall back to plain rows
		return m.plainRows(rows, width)
	}

	pidW, cpuW, memW, rssW, nameW, ttyW, upW := 3, 3, 3, 4, 5, 3, 2
	cells := make([][]string, len(rows))

	for i, r := range rows {
		in := r.node.Info
		name, _ := agentLabel(in)
		if r.depth > 0 {
			name = "└ " + procBin(in)
		}
		cells[i] = []string{
			strconv.Itoa(in.PID),
			num(in.CPU) + " " + meter(in.CPU, 6, true),
			num(in.Mem) + " " + meter(in.Mem, 6, false),
			proc.RSSHuman(in.RSS),
			name,
			in.TTY,
			proc.Since(in.Start),
		}
		if in.CPU < 0 {
			_ = cells[i]
		}
	}

	for _, c := range cells {
		if l := lipgloss.Width(c[0]); l > pidW {
			pidW = l
		}
		if l := lipgloss.Width(c[1]); l > cpuW {
			cpuW = l
		}
		if l := lipgloss.Width(c[2]); l > memW {
			memW = l
		}
		if l := lipgloss.Width(c[3]); l > rssW {
			rssW = l
		}
		if l := lipgloss.Width(c[4]); l > nameW {
			nameW = l
		}
		if l := lipgloss.Width(c[5]); l > ttyW {
			ttyW = l
		}
		if l := lipgloss.Width(c[6]); l > upW {
			upW = l
		}
	}

	// fit to width: shrink agent name, then drop the trailing columns
	full := pidW + cpuW + memW + rssW + nameW + ttyW + upW + 22 // 22 = borders/padding/separators
	if full > width {
		over := full - width
		if over <= nameW-4 {
			nameW -= over
		} else {
			nameW = 4
			drop := over - (nameW - 4)
			if drop >= ttyW+1 {
				ttyW = 0
				upW = 0
			} else if drop >= upW+1 {
				upW = 0
			}
		}
	}

	keep := []string{"PID", "CPU", "MEM", "RSS", "AGENT"}
	widths := []int{pidW, cpuW, memW, rssW, nameW}
	if ttyW > 0 {
		keep = append(keep, "TTY")
		widths = append(widths, ttyW)
	}
	if upW > 0 {
		keep = append(keep, "UP")
		widths = append(widths, upW)
	}

	var sb strings.Builder
	sb.WriteString(tableTop(widths))
	sb.WriteString(tableRow(keep, widths, true))
	sb.WriteString(tableDivider(widths))
	for i, r := range rows {
		vals := make([]string, len(widths))
		c := cells[i]
		vals[0] = padLeft(c[0], pidW)
		vals[1] = c[1]
		vals[2] = c[2]
		vals[3] = padLeft(c[3], rssW)
		vals[4] = colName(r)
		if len(vals) > 5 {
			vals[5] = c[5]
		}
		if len(vals) > 6 {
			vals[6] = padLeft(c[6], upW)
		}
		sb.WriteString(tableRow(vals, widths, false))
	}
	sb.WriteString(tableBottom(widths))
	return sb.String()
}

// colName returns the coloured name cell for a tree row.
func colName(r drow) string {
	name, color := agentLabel(r.node.Info)
	if r.depth > 0 {
		return lipgloss.NewStyle().Foreground(Faint).Render(name)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(name)
}

// plainRows is the narrow-terminal fallback for a live snapshot.
func (m *LiveModel) plainRows(rows []drow, width int) string {
	if len(rows) == 0 {
		return "  no AI CLI agents running"
	}
	var sb strings.Builder
	avail := m.rowsAvail()
	end := minInt(avail, len(rows))
	for i := 0; i < end; i++ {
		sb.WriteString(m.renderRow(rows[i], false, width))
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func padLeft(s string, w int) string {
	d := w - lipgloss.Width(s)
	if d <= 0 {
		return s
	}
	return strings.Repeat(" ", d) + s
}

func num(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// meter renders a w-cell horizontal bar for a 0–100 value.
func meter(v float64, w int, cpu bool) string {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	full := v / 100 * float64(w)
	var sb strings.Builder
	for i := 0; i < w; i++ {
		if !cpu {
			sb.WriteString(barCell(i < int(full), "#A78BFA"))
			continue
		}
		var col string
		switch {
		case v >= 70:
			col = "#EF4444"
		case v >= 40:
			col = "#F59E0B"
		default:
			col = "#22C55E"
		}
		sb.WriteString(barCell(i < int(full), col))
	}
	return sb.String()
}

func barCell(on bool, col string) string {
	if on {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render("█")
	}
	return lipgloss.NewStyle().Foreground(Faint).Render("░")
}

func title(s string) string {
	if s == "" {
		return "agent"
	}
	runes := []rune(s)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] = runes[0] - 'a' + 'A'
	}
	return string(runes)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
