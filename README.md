# rove

One command to launch any installed AI CLI agent. Lists everything on your
machine with a colour for each brand, lets you pick one, and hands the terminal
straight to it.

```
  ⚡ rove · pick your AI CLI
    ▸  Claude Code              claude       2.1.296
         Codex CLI              codex        0.144.5
         OpenCode               opencode     1.18.31
         Crush                  crush        0.56.0
    ─────────────────────────────────────────────
      ● 4 installed · 23 total · 4 matched
      ↑↓ move · ⏎ run/install · u uninstall · / filter · ? help · q quit
```

On wide terminals the picker shows an agent detail card next to the list:

```
  ╭─ AGENTS ─────────────────────────╮  ╭─ CLAUDE CODE ──────────────────╮
  │  ▐  Claude Code  claude   2.1.296│  │                                │
  │     Codex CLI    codex    0.144.5│  │  status   ● installed          │
  │     Gemini CLI   gemini   …      │  │  version  2.1.296              │
  ╰──────────────────────────────────╯  ╰────────────────────────────────╯
```

## Features

- Interactive colour picker with a gradient, animated `ROVE` banner, per-brand
  colours, and async `--version` probes so rows light up as versions resolve.
- Brand glyphs for every agent, with Nerd Font, ASCII and plain markers.
- Side-by-side detail card (status, version, binary, path/install command) on
  wide terminals; condensed one-line detail on narrow ones.
- Direct launch: `rove claude --resume`, `rove codex exec "fix the test"`.
- Managed install/uninstall: run each agent's own installer, or
  `u` twice in the picker to uninstall.
- Live stats dashboard: `rove stats` shows CPU/mem/agent charts for running
  agents.
- Fuzzy name matching: `rove clo`, `rove code`, `rove kil`.
- Non-interactive modes: `rove -l` (colourised table), `rove --json`,
  `rove -a` to include known-but-not-installed agents.
- Auto-detection across `$PATH` plus `~/.local/bin`, `~/go/bin`,
  `~/.cargo/bin`, `~/.bun/bin`, `~/Library/pnpm`, `/opt/homebrew/bin`,
  `/usr/local/bin`.

## Install

```sh
brew install go          # if not already installed
cd rove
make install            # copies `rove` to ~/.local/bin
```

## Usage

```
  rove                     interactive picker
  rove <name> [args...]    launch an agent directly
  rove -- <name> [args]    same, for args that start with a dash
  rove stats, rove ps      live stats dashboard of running agents
  rove install <name>      run the managed installer for an agent
  rove uninstall <name>    run the managed uninstaller for an agent
  rove -l, --list          colourised table
  rove --json              machine-readable JSON
  rove -a, --all           include missing agents
  rove --icons <mode>      brand icons: nerd (default) | ascii | none
  rove --no-animate        disable the animated banner shimmer
  rove -v, --version       print version
```

Environment: `ROVE_ICONS` sets the default icon mode; `NO_COLOR` disables
colour and animation.

Picker keys: `↑ ↓` / `k j` move · `g G` first/last · `⏎` launch (install if
missing) · `u u` uninstall · `/` filter · `?` help · `q` / `esc` quit.

## Detected rove

claude · codex · gemini · opencode · crush · kilocode · aider · cursor-agent ·
goose · amp · copilot · qwen · droid · cline · openhands · mistral · hermes ·
muse · feynman · claude-omniroute · claude-free · cursor-omniroute ·
pi-omniroute