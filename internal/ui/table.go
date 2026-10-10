package ui

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/12britz/rove/internal/detect"
)

// Table renders a bordered, colourised listing of detected agents.
func Table(installs []detect.Install, versions map[string]string, termWidth int) string {
	if termWidth <= 0 {
		termWidth = 100
	}

	n := len(installs)
	names := make([]string, n)
	bins := make([]string, n)
	vers := make([]string, n)
	paths := make([]string, n)
	installed := make([]bool, n)

	for i, in := range installs {
		dot := "○"
		if in.Installed() {
			dot = "●"
		}
		names[i] = dot + " " + in.Agent.Name
		bins[i] = binOf(in)
		paths[i] = in.Path
		installed[i] = in.Installed()
		if !in.Installed() {
			vers[i] = "not installed"
		} else if v := versions[in.Agent.ID]; v != "" {
			vers[i] = v
		} else {
			vers[i] = "—"
		}
	}

	// content widths (headers + longest cell)
	c1 := maxStringWidth(names, "AGENT")
	c2 := maxStringWidth(bins, "BINARY")
	c3 := maxStringWidth(vers, "VERSION")
	c4 := maxStringWidth(paths, "PATH")

	// fit inside termWidth; drop PATH first when the total runs wide
	showPath := true
	if c1+c2+c3+10 > termWidth { // 10 = borders/padding/separators for 3 cols
		return plainFallback(installs, versions, n, names, bins, vers, paths, installed, c1, c2, c3)
	}
	if c1+c2+c3+c4+13 > termWidth { // 13 = same for 4 cols
		over := c1 + c2 + c3 + c4 + 13 - termWidth
		const minPath = 8
		if c4-over >= minPath {
			c4 -= over
		} else {
			showPath = false
		}
	}

	header := []string{"AGENT", "BINARY", "VERSION"}
	rowFns := []func(i int) string{
		func(i int) string { return coloredName(names[i], installs[i].Agent.Color) },
		func(i int) string { return dim.Render(bins[i]) },
		func(i int) string { return dim.Render(vers[i]) },
	}
	cells := []int{c1, c2, c3}
	if showPath {
		header = append(header, "PATH")
		rowFns = append(rowFns, func(i int) string { return faint.Render(paths[i]) })
		cells = append(cells, c4)
	}

	var sb strings.Builder
	sb.WriteString(tableTop(cells))
	sb.WriteString(tableRow(header, cells, true))
	sb.WriteString(tableDivider(cells))
	for i := 0; i < n; i++ {
		cells2 := make([]string, len(cells))
		for j, fn := range rowFns {
			cells2[j] = fn(i)
		}
		sb.WriteString(tableRow(cells2, cells, false))
	}
	sb.WriteString(tableBottom(cells))

	shown, inst := 0, 0
	for _, ok := range installed {
		if ok {
			inst++
		}
		shown++
	}
	legend := lipgloss.NewStyle().Foreground(Success).Render("●") + " " +
		dim.Render(strconv.Itoa(inst)+" installed") + "   ·   " +
		dim.Render(strconv.Itoa(shown)+" shown")
	return strings.TrimRight(sb.String(), "\n") + "\n\n" + legend
}

var (
	dim   = lipgloss.NewStyle().Foreground(Dim)
	faint = lipgloss.NewStyle().Foreground(Faint)
	bold  = lipgloss.NewStyle().Bold(true)
	head  = bold.Foreground(Accent)
)

func coloredName(s string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)
}

func maxStringWidth(ss []string, hdr string) int {
	w := lipgloss.Width(hdr)
	for _, s := range ss {
		if l := lipgloss.Width(s); l > w {
			w = l
		}
	}
	return w
}

func cellWidth(w int) int { return w + 2 }

func padCell(s string, w int) string {
	if l := lipgloss.Width(s); l <= w {
		return " " + s + strings.Repeat(" ", w-l) + " "
	}
	return " " + truncate(s, w) + " "
}

func tableTop(cells []int) string {
	var sb strings.Builder
	sb.WriteString("╭")
	for i, w := range cells {
		sb.WriteString(strings.Repeat("─", cellWidth(w)))
		if i < len(cells)-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("╮\n")
	return sb.String()
}

func tableDivider(cells []int) string {
	var sb strings.Builder
	sb.WriteString("├")
	for i, w := range cells {
		sb.WriteString(strings.Repeat("─", cellWidth(w)))
		if i < len(cells)-1 {
			sb.WriteString("┼")
		}
	}
	sb.WriteString("┤\n")
	return sb.String()
}

func tableBottom(cells []int) string {
	var sb strings.Builder
	sb.WriteString("╰")
	for i, w := range cells {
		sb.WriteString(strings.Repeat("─", cellWidth(w)))
		if i < len(cells)-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("╯\n")
	return sb.String()
}

func tableRow(cells []string, widths []int, isHeader bool) string {
	var sb strings.Builder
	sb.WriteString("│")
	for i, content := range cells {
		w := widths[i]
		if isHeader {
			content = head.Render(content)
		}
		sb.WriteString(padCell(content, w))
		sb.WriteString("│")
	}
	sb.WriteString("\n")
	return sb.String()
}

// plainFallback prints a minimal listing for terminals too narrow for the
// bordered table to fit.
func plainFallback(installs []detect.Install, versions map[string]string, n int,
	names, bins, vers, paths []string, installed []bool, c1, c2, c3 int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		line := padRight(names[i], c1+2) +
			padRight(dim.Render(bins[i]), c2+2) +
			padRight(dim.Render(vers[i]), c3+2)
		if installed[i] {
			line += faint.Render(paths[i])
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func binOf(in detect.Install) string {
	if in.Installed() {
		return filepath.Base(in.Path)
	}
	if len(in.Agent.Binaries) > 0 {
		return in.Agent.Binaries[0]
	}
	return in.Agent.ID
}
