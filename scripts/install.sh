#!/usr/bin/env sh
set -eu

REPO_OWNER="0xJohnnyboy"
REPO_NAME="voyage"
REPO_HOST="github.com"
DEFAULT_INSTALL_DIR="/usr/local/bin"
FALLBACK_INSTALL_DIR="$HOME/.local/bin"

VERSION=""
INSTALL_DIR=""

usage() {
  cat <<USAGE
Install Voyage

Usage:
  install.sh [--version vX.Y.Z] [--install-dir DIR]

Options:
  --version     Install a specific release tag (default: latest)
  --install-dir Install target directory (default: /usr/local/bin, fallback: ~/.local/bin)
  -h, --help    Show this help
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "Missing value for --version" >&2; exit 1; }
      VERSION="$2"
      shift 2
      ;;
    --install-dir)
      [ "$#" -ge 2 ] || { echo "Missing value for --install-dir" >&2; exit 1; }
      INSTALL_DIR="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

[ -n "$INSTALL_DIR" ] || INSTALL_DIR="$DEFAULT_INSTALL_DIR"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

ARTIFACT="vo-$os-$arch"
BASE="https://$REPO_HOST/$REPO_OWNER/$REPO_NAME/releases"
if [ -n "$VERSION" ]; then
  RELEASE_PATH="download/$VERSION"
else
  RELEASE_PATH="latest/download"
fi
ARTIFACT_URL="$BASE/$RELEASE_PATH/$ARTIFACT"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
ARTIFACT_PATH="$TMP_DIR/$ARTIFACT"

curl -fsSL "$ARTIFACT_URL" -o "$ARTIFACT_PATH"

if [ ! -d "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR" 2>/dev/null || true
fi
if [ ! -w "$INSTALL_DIR" ]; then
  if [ "$INSTALL_DIR" = "$DEFAULT_INSTALL_DIR" ]; then
    INSTALL_DIR="$FALLBACK_INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
  else
    echo "Install directory is not writable: $INSTALL_DIR" >&2
    exit 1
  fi
fi

TARGET="$INSTALL_DIR/vo"
install -m 0755 "$ARTIFACT_PATH" "$TARGET"
printf 'Installed: %s\n' "$TARGET"
"$TARGET" -v || true

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    printf "\n'%s' is not in your PATH.\n" "$INSTALL_DIR"
    printf 'Add this line to your shell profile (e.g. ~/.zshrc or ~/.bashrc):\n'
    printf 'export PATH="%s:$PATH"\n' "$INSTALL_DIR"
    ;;
esac
