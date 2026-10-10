package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/12britz/rove/internal/detect"
	"github.com/12britz/rove/internal/registry"
)

type versionMsg struct {
	id      string
	version string
}

// bannerTick advances the animated banner gradient.
type bannerTick struct{}

// Actions a confirmed selection can request from the caller.
const (
	ActionLaunch    = "launch"
	ActionInstall   = "install"
	ActionUninstall = "uninstall"
)

// bannerMinHeight is the minimum terminal height to show the big ROVE
// wordmark header instead of the compact one-line header.
const bannerMinHeight = 15

// wideMinWidth is the minimum width at which the list and detail card are
// shown side by side.
const wideMinWidth = 100

// Model is the interactive agent picker.
type Model struct {
	Installs []detect.Install
	versions map[string]string
	chosen   *detect.Install
	action   string
	icons    IconMode
	animate  bool
	phase    int

	cursor     int
	offset     int
	filter     string
	filtering  bool
	help       bool
	quitting   bool
	confirming bool
	width      int
	height     int

	nameW int
	binW  int
	verW  int
}

// Option customises a picker model.
type Option func(*Model)

// WithIconMode selects the brand-mark style.
func WithIconMode(mode IconMode) Option { return func(m *Model) { m.icons = mode } }

// WithAnimate enables or disables the animated banner shimmer.
func WithAnimate(on bool) Option { return func(m *Model) { m.animate = on } }

// New builds the picker model for a set of installs.
func New(installs []detect.Install, opts ...Option) Model {
	m := Model{
		Installs: installs,
		versions: map[string]string{},
		verW:     14,
		icons:    ParseIconMode(""),
		animate:  true,
	}
	for _, in := range installs {
		if n := lipgloss.Width(in.Agent.Name); n > m.nameW {
			m.nameW = n
		}
		if n := lipgloss.Width(binOf(in)); n > m.binW {
			m.binW = n
		}
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

// Choice returns the selected install, if the user confirmed one.
func (m Model) Choice() (detect.Install, bool) {
	if m.chosen == nil {
		return detect.Install{}, false
	}
	return *m.chosen, true
}

// Action returns what to do with the confirmed choice: launch, install or
// uninstall the agent.
func (m Model) Action() string {
	if m.action == "" {
		return ActionLaunch
	}
	return m.action
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, in := range m.Installs {
		if !in.Installed() {
			continue
		}
		in := in
		cmds = append(cmds, func() tea.Msg {
			return versionMsg{id: in.Agent.ID, version: detect.ProbeVersion(in.Path)}
		})
	}
	if cmd := m.bannerCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m Model) bannerCmd() tea.Cmd {
	if !m.animate || NoColor() {
		return nil
	}
	return tea.Tick(110*time.Millisecond, func(time.Time) tea.Msg { return bannerTick{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil

	case bannerTick:
		m.phase++
		return m, m.bannerCmd()

	case versionMsg:
		m.versions[msg.id] = msg.version
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFiltering(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.clamp()
		m.ensureVisible()
	case "backspace":
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
		m.clamp()
		m.ensureVisible()
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		visible := m.visible()
		if len(visible) == 0 {
			m.filtering = false
			return m, nil
		}
		if m.cursor >= len(visible) {
			m.cursor = len(visible) - 1
		}
		return m.selectInstall(visible[m.cursor])
	default:
		if msg.Type == tea.KeyRunes && !msg.Alt {
			m.filter += string(msg.Runes)
			m.clamp()
			m.ensureVisible()
		}
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visible()
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		if m.confirming {
			m.confirming = false
			return m, nil
		}
		if m.filter != "" {
			m.filter = ""
			m.clamp()
			m.ensureVisible()
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.rowsAvailable())
	case "pgdown":
		m.move(m.rowsAvailable())
	case "g", "home":
		m.cursor = 0
		m.ensureVisible()
	case "G", "end":
		if len(visible) > 0 {
			m.cursor = len(visible) - 1
		}
		m.ensureVisible()
	case "/":
		m.filtering = true
	case "?":
		m.help = !m.help
	case "u":
		if len(visible) == 0 || m.help {
			return m, nil
		}
		sel := visible[m.cursor]
		if !sel.Installed() {
			return m, nil
		}
		if !m.confirming {
			m.confirming = true
			return m, nil
		}
		return m.selectInstall(sel, ActionUninstall)
	case "enter":
		if len(visible) == 0 {
			return m, nil
		}
		if m.cursor >= len(visible) {
			m.cursor = len(visible) - 1
		}
		return m.selectInstall(visible[m.cursor])
	}
	return m, nil
}

// selectInstall confirms a row and records the action to run: launch for an
// installed agent, install for a missing one, or uninstall when requested.
func (m Model) selectInstall(sel detect.Install, action ...string) (Model, tea.Cmd) {
	if len(action) > 0 {
		m.action = action[0]
	} else if sel.Installed() {
		m.action = ActionLaunch
	} else {
		m.action = ActionInstall
	}
	m.chosen = &sel
	m.quitting = true
	return m, tea.Quit
}

func (m *Model) move(d int) {
	n := len(m.visible())
	if n == 0 {
		return
	}
	m.confirming = false
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	m.ensureVisible()
}

func (m *Model) clamp() {
	n := len(m.visible())
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) ensureVisible() {
	avail := m.rowsAvailable()
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

// visible returns the rows after filtering, with installed agents grouped
// before missing ones (relative order preserved).
func (m Model) visible() []detect.Install {
	base := m.Installs
	if m.filter != "" {
		q := registry.Normalize(m.filter)
		base = nil
		for _, in := range m.Installs {
			hay := registry.Normalize(in.Agent.ID + in.Agent.Name + strings.Join(in.Agent.Binaries, ""))
			if strings.Contains(hay, q) {
				base = append(base, in)
			}
		}
	}
	out := make([]detect.Install, 0, len(base))
	for _, in := range base {
		if in.Installed() {
			out = append(out, in)
		}
	}
	for _, in := range base {
		if !in.Installed() {
			out = append(out, in)
		}
	}
	return out
}

func hasMissing(vis []detect.Install) bool {
	for _, in := range vis {
		if !in.Installed() {
			return true
		}
	}
	return false
}

func (m Model) headerLines() int {
	if m.height >= bannerMinHeight {
		return 6 // 5 wordmark lines + tagline
	}
	return 1
}

func (m Model) wideLayout() bool {
	return m.width >= wideMinWidth && m.height >= 18 && !m.help
}

func (m Model) rowsAvailable() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	chrome := m.headerLines() + 3 // footer + divider + first row frame
	if m.wideLayout() {
		chrome++ // list panel bottom border
	}
	avail := h - chrome
	if avail < 3 {
		avail = 3
	}
	return avail
}

func (m Model) contentWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

func (m Model) versionOf(in detect.Install) string {
	if !in.Installed() {
		return "not installed"
	}
	v, ok := m.versions[in.Agent.ID]
	if !ok {
		return "…"
	}
	if v == "" {
		return "—"
	}
	return truncate(v, m.verW-1)
}

func (m Model) versionCell(in detect.Install, w int) string {
	v := m.versionOf(in)
	if !in.Installed() {
		return faint.Render(padRight(v, w))
	}
	if v == "…" || v == "—" {
		return dim.Render(padRight(v, w))
	}
	return lipgloss.NewStyle().Foreground(Success).Render(padRight(v, w))
}

func (m Model) renderRow(in detect.Install, sel bool, width int) string {
	glyph := glyphFor(in.Agent, m.icons)
	col := lipgloss.Color(in.Agent.Color)
	name := padRight(in.Agent.Name, m.nameW)
	bin := padRight(binOf(in), m.binW)

	if !sel {
		g := lipgloss.NewStyle().Foreground(col).Render(glyph)
		if !in.Installed() {
			g = faint.Render(glyph)
		}
		line := "  " + g + "  " +
			lipgloss.NewStyle().Foreground(col).Render(name) + "  " +
			dim.Render(bin) + "  " + m.versionCell(in, m.verW)
		return fit(line, width)
	}

	fg := Contrast(in.Agent.Color)
	g := lipgloss.NewStyle().Foreground(fg).Render(glyph)
	text := "▐ " + g + "  " +
		lipgloss.NewStyle().Foreground(fg).Bold(true).Render(name) + "  " +
		lipgloss.NewStyle().Foreground(fg).Render(bin) + "  " +
		lipgloss.NewStyle().Foreground(fg).Render(m.versionOf(in))
	tail := fadeFill(in.Agent.Color, width, lipgloss.Width(text))
	return lipgloss.NewStyle().Background(col).Render(text) + tail
}

// fadeFill paints the trailing space of a selected row with a shimmering
// gradient that dissolves from the brand colour into darkness, instead of a
// solid block of colour.
func fadeFill(hex string, total, used int) string {
	if used >= total {
		return ""
	}
	n := total - used
	r, g, b := parseHex(hex)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		shade := 1 - 0.85*t + 0.05*math.Sin(float64(i)*0.55)
		if shade < 0 {
			shade = 0
		}
		bg := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
			int(float64(r)*shade), int(float64(g)*shade), int(float64(b)*shade)))
		sb.WriteString(lipgloss.NewStyle().Background(bg).Render(" "))
	}
	return sb.String()
}

func (m Model) headerView() string {
	if m.height >= bannerMinHeight {
		return m.bannerView()
	}
	return m.compactHeaderView()
}

func (m Model) bannerView() string {
	w := m.contentWidth()
	var sb strings.Builder
	for _, line := range bannerLinesPhase(m.phase) {
		gap := w - lipgloss.Width(line)
		if gap < 0 {
			gap = 0
		}
		sb.WriteString(strings.Repeat(" ", gap/2))
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	status := lipgloss.NewStyle().Foreground(Dim).Bold(true).Render("· pick your AI CLI")
	if m.filtering {
		status = lipgloss.NewStyle().Foreground(Accent).Bold(true).Render("/"+m.filter) +
			lipgloss.NewStyle().Foreground(Accent).Render("▌")
	} else if m.filter != "" {
		status = lipgloss.NewStyle().Foreground(Accent).Render("/" + m.filter)
	}
	gap := (w - lipgloss.Width(status)) / 2
	if gap < 0 {
		gap = 0
	}
	sb.WriteString(strings.Repeat(" ", gap))
	sb.WriteString(status)
	return sb.String()
}

func (m Model) compactHeaderView() string {
	w := m.contentWidth()
	bolt := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Render("⚡")
	title := Gradient("rove", "#D97757", "#A78BFA")
	left := bolt + " " + title + " " +
		lipgloss.NewStyle().Foreground(Dim).Render("· pick your AI CLI")

	var right string
	if m.filtering {
		right = lipgloss.NewStyle().Foreground(Accent).Bold(true).Render("/"+m.filter) +
			lipgloss.NewStyle().Foreground(Accent).Render("▌")
	} else if m.filter != "" {
		right = lipgloss.NewStyle().Foreground(Accent).Render("/"+m.filter) +
			lipgloss.NewStyle().Foreground(Faint).Render("  (esc clears)")
	}

	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) helpView() string {
	label := lipgloss.NewStyle().Foreground(Accent).Bold(true)
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render
	dim := lipgloss.NewStyle().Foreground(Dim).Render
	lines := []string{
		"  " + key("↑ ↓") + " / " + key("k j") + "   " + dim("move") +
			"      " + key("⏎") + "   " + dim("launch · ⏎ on missing = install") +
			"      " + key("/") + "   " + dim("filter"),
		"  " + key("u") + "                " + dim("uninstall selected (press twice)") +
			"      " + key("g G") + " " + dim("first / last") +
			"          " + key("?") + "   " + dim("close help"),
		"  " + key("q esc") + " " + dim("quit") +
			"                  " + key("⏎") + " " + dim("runs:") + " " + key("launch") +
			" · " + key("install") + " · " + key("uninstall"),
	}
	return "\n" + label.Render("  keyboard") + "\n" + strings.Join(lines, "\n") + "\n"
}

func (m Model) footerView() string {
	w := m.contentWidth()
	visible := m.visible()

	if m.confirming && len(visible) > 0 && m.cursor < len(visible) {
		in := visible[m.cursor]
		confirm := lipgloss.NewStyle().Foreground(Missing).Bold(true).Render(
			"press u again to uninstall " + in.Agent.Name + " · esc to cancel")
		keys := lipgloss.NewStyle().Foreground(Dim).Render("u confirm · esc cancel")
		gap := w - lipgloss.Width(confirm) - lipgloss.Width(keys)
		if gap < 2 {
			return keys
		}
		return confirm + strings.Repeat(" ", gap) + keys
	}

	installed, total := 0, len(m.Installs)
	for _, in := range m.Installs {
		if in.Installed() {
			installed++
		}
	}
	left := lipgloss.NewStyle().Foreground(Dim).Render(fmt.Sprintf(
		"● %d installed · %d total · %d matched", installed, total, len(visible)))
	keys := lipgloss.NewStyle().Foreground(Dim).Render(
		"↑↓ move · ⏎ run/install · u uninstall · / filter · ? help · q quit")

	gap := w - lipgloss.Width(left) - lipgloss.Width(keys)
	if gap < 2 {
		return keys
	}
	return left + strings.Repeat(" ", gap) + keys
}

// listLines renders the (optionally grouped) agent rows to the given width.
func (m Model) listLines(vis []detect.Install, width int) []string {
	group := m.filter == "" && len(vis) <= m.rowsAvailable() && hasMissing(vis)
	var lines []string
	if group {
		lines = append(lines, m.sectionLabel("INSTALLED", width))
	}
	shownAvail := false
	for i := m.offset; i < len(vis); i++ {
		in := vis[i]
		if group && !in.Installed() && !shownAvail {
			lines = append(lines, m.sectionLabel("AVAILABLE", width))
			shownAvail = true
		}
		lines = append(lines, m.renderRow(in, i == m.cursor, width))
	}
	return lines
}

func (m Model) sectionLabel(text string, width int) string {
	label := lipgloss.NewStyle().Foreground(Faint).Bold(true).Render("  " + text)
	return fit(label, width)
}

// detailCard renders the framed agent preview as a slice of lines.
func (m Model) detailCard(in detect.Install, cardW int) []string {
	inner := cardW - 2
	col := lipgloss.Color(in.Agent.Color)
	glyph := lipgloss.NewStyle().Foreground(col).Bold(true).Render(glyphFor(in.Agent, m.icons))
	body := []string{
		"  " + glyph,
		"  " + dim.Render(truncate(in.Agent.Description, inner-2)),
	}
	if in.Agent.VariantOf != "" {
		body = append(body, "  "+faint.Render("variant of "+in.Agent.VariantOf))
	}
	body = append(body, "")
	body = append(body, "  "+dim.Render(padRight("status", 9))+statusPill(in.Installed()))
	body = append(body, m.cardField("version", m.versionOf(in), inner))
	body = append(body, m.cardField("binary", binOf(in), inner))
	switch {
	case in.Installed():
		body = append(body, m.cardField("path", in.Path, inner))
	case in.Agent.Install != "":
		body = append(body, m.cardField("install", in.Agent.Install, inner))
	default:
		body = append(body, "  "+dim.Render(padRight("install", 9))+
			lipgloss.NewStyle().Foreground(Missing).Render("no managed installer"))
	}
	return panel(strings.ToUpper(in.Agent.Name), body, cardW, col)
}

func (m Model) cardField(label, value string, inner int) string {
	const lw = 9
	budget := inner - 2 - lw
	if budget < 4 {
		budget = 4
	}
	lead := "  " + dim.Render(padRight(label, lw))
	return lead + lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render(truncate(value, budget))
}

// condensedDetail is the narrow-layout one-line detail for the selected agent.
func (m Model) condensedDetail(in detect.Install, w int) string {
	g := lipgloss.NewStyle().Foreground(lipgloss.Color(in.Agent.Color)).Render(glyphFor(in.Agent, m.icons))
	name := lipgloss.NewStyle().Foreground(lipgloss.Color(in.Agent.Color)).Bold(true).Render(in.Agent.Name)
	prefix := "  " + g + "  " + name + "  "
	budget := w - lipgloss.Width(prefix) - 2
	if budget < 6 {
		budget = 6
	}
	var tail string
	if in.Installed() {
		tail = dim.Render(m.versionOf(in)) + "  " + faint.Render(truncate(in.Path, budget))
	} else if in.Agent.Install != "" {
		tail = faint.Render(truncate("⏎ installs: "+in.Agent.Install, budget))
	} else {
		tail = faint.Render("not installed")
	}
	return fit(prefix+tail, w)
}

// View renders the full screen.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	w := m.contentWidth()
	divider := lipgloss.NewStyle().Foreground(Faint).Render(strings.Repeat("─", w))

	if m.help {
		return m.headerView() + "\n" + m.helpView() + divider + "\n" + m.footerView()
	}

	vis := m.visible()

	if m.wideLayout() {
		cardW := clampInt(m.contentWidth()/3, 28, 44)
		listW := w - cardW - 3
		listInner := listW - 2
		var body []string
		if len(vis) == 0 {
			body = []string{faint.Render("  no agents match “" + m.filter + "”")}
		} else {
			body = m.listLines(vis, listInner)
		}
		listPanel := panel("AGENTS", body, listW, Accent)
		var card []string
		if len(vis) > 0 {
			card = m.detailCard(vis[m.cursor], cardW)
		} else {
			card = panel("AGENT", []string{""}, cardW, Accent)
		}
		band := lipgloss.JoinHorizontal(lipgloss.Top,
			strings.Join(listPanel, "\n"), "  ", strings.Join(card, "\n"))
		return m.headerView() + "\n" + band + "\n" + divider + "\n" + m.footerView()
	}

	var body string
	if len(vis) == 0 {
		body = "\n  " + faint.Render("no agents match “"+m.filter+"”") + "\n"
	} else {
		lines := m.listLines(vis, w)
		body = "\n" + strings.Join(lines, "\n") + "\n"
	}
	detail := ""
	if len(vis) > 0 && m.cursor < len(vis) {
		detail = m.condensedDetail(vis[m.cursor], w) + "\n"
	}
	return m.headerView() + "\n" + body + detail + divider + "\n" + m.footerView()
}
