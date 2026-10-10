package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Shared palette.
var (
	Dim     = lipgloss.AdaptiveColor{Light: "#8C8C8C", Dark: "#6B7280"}
	Faint   = lipgloss.AdaptiveColor{Light: "#B0B0B0", Dark: "#4B5563"}
	Accent  = lipgloss.Color("#A78BFA")
	Success = lipgloss.Color("#22C55E")
	Warn    = lipgloss.Color("#F59E0B")
	Missing = lipgloss.Color("#4B5563")
)

// NoColor reports whether colour output has been disabled via NO_COLOR.
func NoColor() bool { return os.Getenv("NO_COLOR") != "" }

// pill renders a compact inline badge (coloured text on a dim background).
func pill(text string, fg, bg lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(fg).Background(bg).Padding(0, 1).Render(text)
}

// statusPill renders the installed / missing badge used by the detail card.
func statusPill(installed bool) string {
	if installed {
		return pill("● installed", lipgloss.Color("#0B1F12"), Success)
	}
	return pill("○ missing", lipgloss.Color("#E5E7EB"), Missing)
}

// Gradient renders text with a per-rune colour interpolation from → to.
func Gradient(text, from, to string) string {
	r1, g1, b1 := parseHex(from)
	r2, g2, b2 := parseHex(to)
	runes := []rune(text)
	n := len(runes)
	var sb strings.Builder
	for i, rn := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		col := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
			lerp(r1, r2, t), lerp(g1, g2, t), lerp(b1, b2, t)))
		sb.WriteString(lipgloss.NewStyle().Foreground(col).Render(string(rn)))
	}
	return sb.String()
}

// Contrast picks a readable foreground (near-black or white) for a hex background.
func Contrast(hex string) lipgloss.Color {
	r, g, b := parseHex(hex)
	lum := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
	if lum > 0.55 {
		return lipgloss.Color("#141414")
	}
	return lipgloss.Color("#FFFFFF")
}

func parseHex(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	var v [3]int
	for i := 0; i < 3; i++ {
		var n int
		for _, c := range hex[i*2 : i*2+2] {
			n *= 16
			switch {
			case c >= '0' && c <= '9':
				n += int(c - '0')
			case c >= 'a' && c <= 'f':
				n += int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				n += int(c-'A') + 10
			}
		}
		v[i] = n
	}
	return v[0], v[1], v[2]
}

func lerp(a, b int, t float64) int {
	return int(float64(a) + (float64(b)-float64(a))*t + 0.5)
}

func padRight(s string, w int) string {
	d := w - lipgloss.Width(s)
	if d <= 0 {
		return s
	}
	return s + strings.Repeat(" ", d)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}
