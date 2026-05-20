#!/bin/sh
# Mio installer — downloads the latest release for your platform.
# Usage: curl -fsSL https://raw.githubusercontent.com/Yarov/mio/main/install.sh | sh

set -e

REPO="Yarov/mio"
INSTALL_DIR="${MIO_INSTALL_DIR:-}"
BINARY="mio"

# Default to ~/.local/bin if INSTALL_DIR not set and /usr/local/bin not writable
if [ -z "$INSTALL_DIR" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { printf "${CYAN}[mio]${NC} %s\n" "$1"; }
ok()    { printf "${GREEN}[mio]${NC} %s\n" "$1"; }
warn()  { printf "${YELLOW}[mio]${NC} %s\n" "$1"; }
error() { printf "${RED}[mio]${NC} %s\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
  case "$(uname -s)" in
    Darwin)  echo "darwin" ;;
    Linux)   echo "linux" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) error "Unsupported OS: $(uname -s)" ;;
  esac
}

# Detect architecture
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "amd64" ;;
    aarch64|arm64)   echo "arm64" ;;
    *) error "Unsupported architecture: $(uname -m)" ;;
  esac
}

# Get latest version from GitHub
get_latest_version() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/'
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/'
  else
    error "curl or wget required"
  fi
}

# Download file
download() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  fi
}

main() {
  info "Installing Mio — persistent memory for AI agents"
  echo

  OS=$(detect_os)
  ARCH=$(detect_arch)
  info "Detected: ${OS}/${ARCH}"

  VERSION=$(get_latest_version)
  if [ -z "$VERSION" ]; then
    error "Could not determine latest version. Check https://github.com/${REPO}/releases"
  fi
  info "Latest version: v${VERSION}"

  # Build download URL
  EXT="tar.gz"
  if [ "$OS" = "windows" ]; then
    EXT="zip"
  fi
  FILENAME="mio_${VERSION}_${OS}_${ARCH}.${EXT}"
  URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

  # Download
  TMPDIR=$(mktemp -d)
  trap 'rm -rf "$TMPDIR"' EXIT

  info "Downloading ${FILENAME}..."
  download "$URL" "${TMPDIR}/${FILENAME}"

  # Extract
  info "Extracting..."
  if [ "$EXT" = "tar.gz" ]; then
    tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"
  else
    unzip -q "${TMPDIR}/${FILENAME}" -d "$TMPDIR"
  fi

  # Find the binary (tar extracts to a subdirectory)
  info "Looking for binary..."
  BINARY_PATH="$TMPDIR/mio"
  if [ ! -f "$BINARY_PATH" ]; then
    BINARY_PATH=$(find "$TMPDIR" -name "$BINARY" -type f 2>/dev/null | head -1)
  fi
  if [ -z "$BINARY_PATH" ] || [ ! -f "$BINARY_PATH" ]; then
    info "TMPDIR contents:"
    ls -laR "$TMPDIR" 2>/dev/null || true
    error "Could not find $BINARY in extracted files"
  fi
  info "Found binary at: $BINARY_PATH"

  # Install
  mkdir -p "$INSTALL_DIR"
  cp "$BINARY_PATH" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"

  # Add to PATH if needed
  if [ "$INSTALL_DIR" = "${HOME}/.local/bin" ]; then
    SHELL_RC="${HOME}/.zshrc"
    if [ -f "${HOME}/.bashrc" ]; then
      SHELL_RC="${HOME}/.bashrc"
    fi
    if ! grep -q '.local/bin' "$SHELL_RC" 2>/dev/null; then
      echo '' >> "$SHELL_RC"
      echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$SHELL_RC"
      info "Added ~/.local/bin to PATH in $SHELL_RC"
      info "Run 'source $SHELL_RC' or restart your terminal"
    fi
  fi

  # macOS: ad-hoc codesign
  if [ "$OS" = "darwin" ]; then
    codesign -f -s - "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
  fi

  # Verify
  if "${INSTALL_DIR}/${BINARY}" version >/dev/null 2>&1; then
    echo
    ok "Mio v${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
    echo
    info "Next steps:"
    echo "  1. Run 'mio setup' to configure your AI agent"
    echo "  2. Restart your agent (Claude Code, Cursor, etc.)"
    echo "  3. Open http://localhost:7438 for the dashboard"
    echo
  else
    error "Installation failed. Binary not working."
  fi
}

main "$@"
