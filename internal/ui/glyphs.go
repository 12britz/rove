package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/12britz/rove/internal/registry"
)

// IconMode selects how agent brand marks are drawn.
type IconMode int

const (
	IconsNerd  IconMode = iota // Nerd Font glyphs (default)
	IconsASCII                 // portable ASCII fallback
	IconsNone                  // plain bullet
)

// ParseIconMode maps a user string ("nerd", "ascii", "none") to an IconMode,
// honouring the ROVE_ICONS environment variable and NO_COLOR.
func ParseIconMode(s string) IconMode {
	if s == "" {
		s = os.Getenv("ROVE_ICONS")
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ascii":
		return IconsASCII
	case "none", "off":
		return IconsNone
	case "nerd":
		return IconsNerd
	}
	return IconsNerd
}

// asciiGlyph gives every known agent a distinct, width-1 ASCII mark for
// terminals without a Nerd Font.
var asciiGlyph = map[string]string{
	"claude":           "*",
	"codex":            ">",
	"gemini":           "G",
	"opencode":         "O",
	"crush":            "+",
	"kilocode":         "K",
	"aider":            "A",
	"cursor":           "C",
	"goose":            "g",
	"amp":              "&",
	"copilot":          "H",
	"qwen":             "Q",
	"droid":            "D",
	"cline":            "L",
	"openhands":        "N",
	"mistral":          "M",
	"hermes":           "E",
	"muse":             "m",
	"feynman":          "F",
	"claude-omniroute": "*",
	"claude-free":      "*",
	"cursor-omniroute": "C",
	"pi-omniroute":     "p",
}

// glyphFor returns the display mark for an agent under the given icon mode.
// Nerd glyphs wider than one cell fall back to ASCII so columns never drift.
func glyphFor(a registry.Agent, mode IconMode) string {
	switch mode {
	case IconsNone:
		return "●"
	case IconsASCII:
		return asciiFor(a.ID)
	}
	if a.Glyph != "" && lipgloss.Width(a.Glyph) == 1 {
		return a.Glyph
	}
	return asciiFor(a.ID)
}

func asciiFor(id string) string {
	if g, ok := asciiGlyph[id]; ok {
		return g
	}
	return "•"
}
