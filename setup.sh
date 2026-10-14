#!/usr/bin/env bash
set -euo pipefail

# rove installer
#
#   curl -fsSL https://raw.githubusercontent.com/12britz/rove/main/setup.sh | bash
#
# Environment overrides:
#   ROVE_VERSION     install a specific tag (e.g. v0.1.0). Default: latest release.
#   ROVE_INSTALL_DIR install directory. Default: $HOME/.local/bin
#   ROVE_REPO        source repo. Default: 12britz/rove

REPO="${ROVE_REPO:-12britz/rove}"
BIN="rove"
INSTALL_DIR="${ROVE_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${ROVE_VERSION:-}"

info() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1; }

need curl || err "curl is required"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin | linux) ;;
  *) err "unsupported OS: $os (rove supports macOS and Linux)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

if [ -z "$VERSION" ]; then
  info "resolving latest ${BIN} release..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
    | grep -o '"tag_name": *"[^"]*"' | head -1 | sed 's/.*"tag_name": *"//; s/"$//' || true)"
fi

tag="${VERSION:-}"
asset=""
if [ -n "$tag" ]; then
  asset="${BIN}_${tag#v}_${os}_${arch}.tar.gz"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

install_binary() {
  mkdir -p "$INSTALL_DIR"
  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$1" "$INSTALL_DIR/$BIN"
  else
    cp "$1" "$INSTALL_DIR/$BIN"
    chmod 0755 "$INSTALL_DIR/$BIN"
  fi
  info "installed ${BIN} to ${INSTALL_DIR}/${BIN}"
}

install_from_release() {
  [ -n "$asset" ] || return 1
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
  info "downloading ${url}"
  curl -fsSL "$url" -o "$tmp/$asset" || return 1
  tar -xzf "$tmp/$asset" -C "$tmp" || return 1
  [ -f "$tmp/$BIN" ] || return 1
  install_binary "$tmp/$BIN"
}

install_from_go() {
  need go || return 1
  ref="${VERSION:-latest}"
  info "no release binary available; building from source (go install ${REPO}@${ref})..."
  GOBIN="$tmp" GOFLAGS="-trimpath" go install "${REPO}@${ref}" || return 1
  [ -f "$tmp/$BIN" ] || return 1
  install_binary "$tmp/$BIN"
}

if ! install_from_release; then
  install_from_go || err "installation failed (no usable release and no Go toolchain found)"
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    warn "${INSTALL_DIR} is not on your PATH. Add this to your shell profile:"
    printf '    export PATH="%s:$PATH"\n' "$INSTALL_DIR"
    ;;
esac

info "done. Run '${BIN}' to get started."
