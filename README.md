# rove

One command to launch any installed AI CLI agent. Lists everything on your
machine with a colour for each brand, lets you pick one, and hands the terminal
straight to it.

```sh
curl -fsSL https://raw.githubusercontent.com/12britz/rove/main/setup.sh | bash
```

<img width="757" height="374" alt="image" src="https://github.com/user-attachments/assets/3c631936-6ac8-471c-95ff-6da7b02b269c" />


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

One-liner (macOS / Linux):

```sh
curl -fsSL https://raw.githubusercontent.com/12britz/rove/main/setup.sh | bash
```

This downloads the latest prebuilt binary to `~/.local/bin`. Override with
`ROVE_VERSION=v0.1.0` or `ROVE_INSTALL_DIR=/usr/local/bin`.

rove can also install the AI CLI agents themselves, not just the ones already on
your machine: run `rove -a` to include known-but-missing agents, then press
`⏎` on one to install it, or run `rove install <name>` directly.

From source:

```sh
brew install go          # if not already installed
cd rove
make install            # copies `rove` to ~/.local/bin
```

## Usage

Commands:

| Command | Description |
| --- | --- |
| `rove` | interactive colour picker (TUI) |
| `rove <agent> [args...]` | launch an agent directly |
| `rove -- <agent> [args]` | same, for args beginning with a dash |
| `rove stats` / `rove ps` | live stats dashboard of running agents |
| `rove install <agent>` | run the managed installer for an agent |
| `rove uninstall <agent>` | run the managed uninstaller for an agent |

Options:

| Option | Description |
| --- | --- |
| `-l`, `--list` | colourised table of detected agents |
| `--json` | machine-readable JSON list |
| `-a`, `--all` | also show known-but-not-installed agents |
| `--icons <mode>` | brand icons: `nerd` (default) \| `ascii` \| `none` |
| `--animate` | force the animated banner shimmer on |
| `--no-animate` | disable the animated banner shimmer |
| `-v`, `--version` | print version |

Environment:

| Variable | Description |
| --- | --- |
| `ROVE_ICONS` | default icon mode (`nerd` \| `ascii` \| `none`) |
| `NO_COLOR` | disable colour and animation |
| `ROVE_VERSION` | version to install via `setup.sh` (default: latest) |
| `ROVE_INSTALL_DIR` | where `setup.sh` installs the binary (default: `~/.local/bin`) |
| `ROVE_REPO` | source repo used by `setup.sh` (default: `12britz/rove`) |

Picker keys: `↑ ↓` / `k j` move · `g G` first/last · `⏎` launch (install if
missing) · `u u` uninstall · `/` filter · `?` help · `q` / `esc` quit.

## Detected rove

claude · codex · gemini · opencode · crush · kilocode · aider · cursor-agent ·
goose · amp · copilot · qwen · droid · cline · openhands · mistral · hermes ·
muse · feynman · claude-omniroute · claude-free · cursor-omniroute ·
pi-omniroute
