package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// bannerArt is the "ROVE" wordmark in a compact lean style (5px tall).
var bannerArt = map[rune][]string{
	'R': {
		"█████",
		"█   █",
		"████ ",
		"█  █ ",
		"█   █",
	},
	'O': {
		"█████",
		"█   █",
		"█   █",
		"█   █",
		"█████",
	},
	'V': {
		"█   █",
		"█   █",
		"█   █",
		" █ █ ",
		"  █  ",
	},
	'E': {
		"█████",
		"█    ",
		"████ ",
		"█    ",
		"█████",
	},
}

// bannerColors gives each letter its own creative gradient.
var bannerColors = map[rune][2]string{
	'R': {"#FF5C5C", "#FFB25C"},
	'O': {"#FFC24D", "#61E08C"},
	'V': {"#5CC8FF", "#8E7BFF"},
	'E': {"#B96BFF", "#FF6BC6"},
}

// bannerLines renders the ROVE wordmark, painting each letter with its own
// horizontal gradient so the word reads as four creative colour blocks
// rather than one solid slab.
func bannerLines() []string {
	return bannerLinesShift(0)
}

// bannerLinesPhase renders the wordmark with a phase-dependent gradient shift,
// giving the letters a gentle drifting shimmer. Phase 0 is identical to the
// static bannerLines output.
func bannerLinesPhase(phase int) []string {
	shift := math.Sin(float64(phase)*0.18) * 0.25
	return bannerLinesShift(shift)
}

// bannerLinesShift interpolates each letter's gradient pair with a phase
// offset (reflected to stay within the gradient's range).
func bannerLinesShift(shift float64) []string {
	const gap = "  "
	text := "ROVE"
	height := len(bannerArt['R'])

	out := make([]string, height)
	for l := 0; l < height; l++ {
		var sb strings.Builder
		for i := 0; i < len(text); i++ {
			ch := rune(text[i])
			if i > 0 {
				sb.WriteString(gap)
			}
			pair := bannerColors[ch]
			r1, g1, b1 := parseHex(pair[0])
			r2, g2, b2 := parseHex(pair[1])
			row := []rune(bannerArt[ch][l])
			n := len(row)
			for j, rn := range row {
				t := 0.0
				if n > 1 {
					t = float64(j) / float64(n-1)
				}
				t = reflect01(t + shift)
				col := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X",
					lerp(r1, r2, t), lerp(g1, g2, t), lerp(b1, b2, t)))
				sb.WriteString(lipgloss.NewStyle().Foreground(col).Render(string(rn)))
			}
		}
		out[l] = sb.String()
	}
	return out
}

// reflect01 folds v back into [0,1] so a shifted gradient parameter never
// overshoots the colour pair.
func reflect01(v float64) float64 {
	for v < 0 || v > 1 {
		if v < 0 {
			v = -v
		}
		if v > 1 {
			v = 2 - v
		}
	}
	return v
}
