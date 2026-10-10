package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/12britz/rove/internal/detect"
	"github.com/12britz/rove/internal/launch"
	"github.com/12britz/rove/internal/registry"
	"github.com/12britz/rove/internal/ui"
)

var version = "dev"

func main() {
	var (
		listFlag      bool
		jsonFlag      bool
		allFlag       bool
		verFlag       bool
		iconsFlag     string
		animateFlag   bool
		noAnimateFlag bool
	)
	flag.BoolVar(&listFlag, "l", false, "")
	flag.BoolVar(&listFlag, "list", false, "")
	flag.BoolVar(&jsonFlag, "json", false, "")
	flag.BoolVar(&allFlag, "a", false, "")
	flag.BoolVar(&allFlag, "all", false, "")
	flag.BoolVar(&verFlag, "v", false, "")
	flag.BoolVar(&verFlag, "version", false, "")
	flag.StringVar(&iconsFlag, "icons", "", "")
	flag.BoolVar(&animateFlag, "animate", false, "")
	flag.BoolVar(&noAnimateFlag, "no-animate", false, "")
	flag.Usage = usage
	flag.Parse()

	iconMode := ui.ParseIconMode(iconsFlag)
	animate := !noAnimateFlag
	if animateFlag {
		animate = true
	}
	if noAnimateFlag {
		animate = false
	}

	if verFlag {
		fmt.Printf("rove %s\n", version)
		return
	}

	args := flag.Args()
	known := registry.Known()

	switch {
	case jsonFlag:
		printJSON(known, allFlag)
		return
	case listFlag:
		installs := detect.Scan(known, allFlag)
		fmt.Println(ui.Table(installs, detect.ProbeVersions(installs), screenWidth()))
		return
	}

	if len(args) > 0 {
		switch args[0] {
		case "stats", "ps":
			runStats()
			return
		case "install":
			if len(args) < 2 {
				fail(fmt.Errorf("usage: rove install <agent>"))
			}
			inst, err := resolveList(detect.Scan(registry.Known(), true), args[1])
			if err != nil {
				fail(err)
			}
			runAgentCommand(inst.Agent, ui.ActionInstall)
			return
		case "uninstall":
			if len(args) < 2 {
				fail(fmt.Errorf("usage: rove uninstall <agent>"))
			}
			inst, err := resolveList(detect.Scan(registry.Known(), false), args[1])
			if err != nil {
				fail(err)
			}
			runAgentCommand(inst.Agent, ui.ActionUninstall)
			return
		}
		inst, err := resolve(args[0])
		if err != nil {
			fail(err)
		}
		if err := launch.Exec(inst.Path, args[1:]); err != nil {
			fail(err)
		}
		return
	}

	installs := detect.Scan(known, allFlag)
	if len(detect.Scan(known, false)) == 0 && !allFlag {
		fail(fmt.Errorf("no AI CLI agents found on PATH"))
	}
	if !isTTY() {
		fmt.Println(ui.Table(installs, detect.ProbeVersions(installs), screenWidth()))
		return
	}

	p := tea.NewProgram(ui.New(installs, ui.WithIconMode(iconMode), ui.WithAnimate(animate)), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		fail(fmt.Errorf("picker failed: %w", err))
	}
	model, ok := final.(ui.Model)
	if !ok {
		return
	}
	inst, ok := model.Choice()
	if !ok {
		return // user quit without choosing
	}
	switch model.Action() {
	case ui.ActionInstall, ui.ActionUninstall:
		runAgentCommand(inst.Agent, model.Action())
	default:
		if err := launch.Exec(inst.Path, nil); err != nil {
			fail(err)
		}
	}
}

// runAgentCommand runs the managed install or uninstall command for an agent.
func runAgentCommand(a registry.Agent, action string) {
	var cmd, label string
	switch action {
	case ui.ActionInstall:
		if a.Install == "" {
			fail(fmt.Errorf("no managed installer for %s; install it manually", a.Name))
		}
		cmd, label = a.Install, "installing "+a.Name
	case ui.ActionUninstall:
		if a.UninstallCmd() == "" {
			fail(fmt.Errorf("no managed uninstaller for %s; remove it manually", a.Name))
		}
		cmd, label = a.UninstallCmd(), "uninstalling "+a.Name
	default:
		return
	}
	fmt.Printf("\033[2m%s\033[0m\n$ %s\n", label, cmd)
	c := exec.Command("bash", "-c", cmd)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		fail(fmt.Errorf("%s failed: %v", label, err))
	}
}

// runStats shows a live stats dashboard of running AI agent processes.
func runStats() {
	m := ui.NewLive()
	if !isTTY() {
		model := &m
		fmt.Println(model.TableText(screenWidth()))
		return
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fail(fmt.Errorf("stats view failed: %w", err))
	}
}

// resolve maps a user-typed name to an installed agent, with helpful errors.
func resolve(query string) (detect.Install, error) {
	return resolveList(detect.Scan(registry.Known(), false), query)
}

// resolveList finds the install matching query in a set of installs.
func resolveList(installs []detect.Install, query string) (detect.Install, error) {
	agents := make([]registry.Agent, len(installs))
	for i, in := range installs {
		agents[i] = in.Agent
	}

	matches := registry.Match(agents, query)
	switch len(matches) {
	case 1:
		for _, in := range installs {
			if in.Agent.ID == matches[0].ID {
				return in, nil
			}
		}
	case 0:
		if known := registry.Match(registry.Known(), query); len(known) > 0 {
			names := make([]string, len(known))
			for i, a := range known {
				names[i] = a.Binaries[0]
			}
			return detect.Install{}, fmt.Errorf("%q is known but not installed\n  install: rove install %s\n  see all: rove -l -a",
				query, names[0])
		}
		return detect.Install{}, fmt.Errorf("no agent matches %q\n  run `rove` to pick from what is installed", query)
	default:
		names := make([]string, len(matches))
		for i, a := range matches {
			names[i] = a.Binaries[0]
		}
		return detect.Install{}, fmt.Errorf("%q is ambiguous: %s", query, strings.Join(names, ", "))
	}
	return detect.Install{}, fmt.Errorf("could not resolve %q", query)
}

type jsonEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Binary    string `json:"binary"`
	Color     string `json:"color"`
	VariantOf string `json:"variant_of,omitempty"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}

func printJSON(known []registry.Agent, all bool) {
	installs := detect.Scan(known, true)
	versions := detect.ProbeVersions(installs)

	entries := make([]jsonEntry, 0, len(installs))
	for _, in := range installs {
		if !in.Installed() && !all {
			continue
		}
		e := jsonEntry{
			ID:        in.Agent.ID,
			Name:      in.Agent.Name,
			Binary:    in.Agent.Binaries[0],
			Color:     in.Agent.Color,
			VariantOf: in.Agent.VariantOf,
			Installed: in.Installed(),
			Path:      in.Path,
		}
		if in.Installed() {
			e.Version = versions[in.Agent.ID]
		}
		entries = append(entries, e)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(entries); err != nil {
		fail(err)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `rove — launch any installed AI CLI agent from one command

Usage:
  rove                       interactive colour picker (TUI)
  rove <agent> [args...]     launch an agent directly
  rove -- <agent> [args]     same, for args beginning with a dash
  rove stats, rove ps        live stats dashboard of running AI agents
  rove install <agent>       run the managed installer for an agent
  rove uninstall <agent>     run the managed uninstaller for an agent
  rove -l, --list            colourised table of detected agents
  rove --json                machine-readable JSON list
  rove -a, --all             also show known-but-not-installed agents
  rove --icons <mode>        brand icons: nerd (default) | ascii | none
  rove --no-animate          disable the animated banner shimmer
  rove -v, --version         print version

Environment:
  ROVE_ICONS                 default icon mode (nerd | ascii | none)
  NO_COLOR                   disable colour and animation

Examples:
  rove                       # pick from a menu
  rove claude --resume
  rove codex exec "fix the flaky test"

Keys (interactive):
  ↑ ↓ / k j   move        ⏎   launch / install    u (×2)  uninstall
  g G         first/last  ?   help               q       quit
`)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "rove: "+err.Error())
	os.Exit(1)
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func screenWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 100
}
