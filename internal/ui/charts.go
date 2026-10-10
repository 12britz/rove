package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// blockLv maps a 0–8 intensity to a block glyph (space = empty).
var blockLv = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// fit truncates s to w display cells and right-pads it with spaces.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return padRight(truncate(s, w), w)
}

// resample maps vals onto exactly w buckets, right-aligned (newest on the
// right). When down-sampling, each bucket keeps the maximum value.
func resample(vals []float64, w int) []float64 {
	out := make([]float64, w)
	if w <= 0 || len(vals) == 0 {
		return out
	}
	if len(vals) <= w {
		copy(out[w-len(vals):], vals)
		return out
	}
	step := float64(len(vals)) / float64(w)
	for i := 0; i < w; i++ {
		lo := int(float64(i) * step)
		hi := int(float64(i+1) * step)
		if hi <= lo {
			hi = lo + 1
		}
		if lo > len(vals)-1 {
			lo = len(vals) - 1
		}
		if hi > len(vals) {
			hi = len(vals)
		}
		mx := 0.0
		for j := lo; j < hi; j++ {
			if vals[j] > mx {
				mx = vals[j]
			}
		}
		out[i] = mx
	}
	return out
}

// kindColor picks a bar colour for a 0–100 reading; memory uses a fixed hue.
func kindColor(kind string, v float64) string {
	if kind == "mem" {
		return "#A78BFA"
	}
	switch {
	case v >= 70:
		return "#EF4444"
	case v >= 40:
		return "#F59E0B"
	default:
		return "#22C55E"
	}
}

// gauge renders a w-cell horizontal bar filled to pct (0–100).
func gauge(pct float64, w int, col string) string {
	if w < 1 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct/100*float64(w) + 0.5)
	var sb strings.Builder
	for i := 0; i < w; i++ {
		if i < filled {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render("█"))
		} else {
			sb.WriteString(faint.Render("░"))
		}
	}
	return sb.String()
}

// sparkline renders vals as a single-row block history, newest on the right.
func sparkline(vals []float64, w int, kind string) string {
	if w < 1 {
		return ""
	}
	bs := resample(vals, w)
	mx := 0.0
	for _, v := range bs {
		if v > mx {
			mx = v
		}
	}
	if mx <= 0 {
		mx = 1
	}
	var sb strings.Builder
	for _, v := range bs {
		if v <= 0 {
			sb.WriteString(faint.Render("▁"))
			continue
		}
		idx := int(v/mx*8 + 0.5)
		if idx < 1 {
			idx = 1
		} else if idx > 8 {
			idx = 8
		}
		col := kindColor(kind, v/mx*100)
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(string(blockLv[idx])))
	}
	return sb.String()
}

// vgraph renders vals as a w-wide, h-tall vertical bar chart (task-manager
// style), newest on the right. maxv sets the top of the scale.
func vgraph(vals []float64, maxv float64, w, h int, kind string) []string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if maxv <= 0 {
		maxv = 100
	}
	bs := resample(vals, w)
	lines := make([]string, h)
	for r := 0; r < h; r++ {
		lvl := h - r // distance from the bottom, 1-based
		var sb strings.Builder
		for _, v := range bs {
			rows := v / maxv * float64(h)
			full := int(rows)
			frac := rows - float64(full)
			var ch rune = ' '
			switch {
			case full >= lvl:
				ch = '█'
			case full == lvl-1:
				if idx := int(frac*8 + 0.5); idx > 0 {
					if idx > 8 {
						idx = 8
					}
					ch = blockLv[idx]
				}
			}
			if ch == ' ' {
				sb.WriteString(" ")
			} else {
				sb.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color(kindColor(kind, v))).Render(string(ch)))
			}
		}
		lines[r] = sb.String()
	}
	return lines
}

// panel draws a titled, rounded box of the given outer width around body
// lines. Body lines are truncated/padded to the inner width.
func panel(title string, body []string, width int, accent lipgloss.Color) []string {
	if width < 12 {
		width = 12
	}
	inner := width - 2
	titleS := " " + bold.Foreground(accent).Render(title) + " "
	fill := inner - 1 - lipgloss.Width(titleS)
	if fill < 0 {
		titleS = " " + bold.Foreground(accent).Render(truncate(title, inner-4)) + " "
		if fill = inner - 1 - lipgloss.Width(titleS); fill < 0 {
			fill = 0
		}
	}
	out := make([]string, 0, len(body)+2)
	out = append(out, "╭─"+titleS+strings.Repeat("─", fill)+"╮")
	for _, l := range body {
		out = append(out, "│"+fit(l, inner)+"│")
	}
	out = append(out, "╰"+strings.Repeat("─", inner)+"╯")
	return out
}
