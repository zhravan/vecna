#!/usr/bin/env sh
set -eu

REPO="zhravan/vecna"
VERSION="${VERSION:-latest}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

info() {
  printf '[vecna] %s\n' "$1"
}

success() {
  printf '[vecna] ✓ %s\n' "$1"
}

fail() {
  printf '[vecna] ✗ %s\n' "$1" >&2
  exit 1
}

case "$(uname -s)" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) fail "Unsupported OS: $(uname -s)" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) fail "Unsupported architecture: $(uname -m)" ;;
esac

command -v curl >/dev/null 2>&1 || fail "curl is required."
command -v tar >/dev/null 2>&1 || fail "tar is required."

info "Starting Vecna installation"
info "Platform: ${OS}/${ARCH}"

if [ "$VERSION" = "latest" ]; then
  info "Checking latest release..."
  TAG="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$TAG" ] || fail "Unable to determine the latest Vecna release."
else
  TAG="${VERSION#v}"
fi

ASSET="vecna_${TAG}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/$REPO/releases/download/v${TAG}"
ARCHIVE_URL="$BASE_URL/$ASSET"
CHECKSUM_URL="$BASE_URL/SHA256SUMS.txt"

TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t vecna)"
ARCHIVE="$TMP_DIR/$ASSET"
CHECKSUMS="$TMP_DIR/SHA256SUMS.txt"
EXTRACTED="$TMP_DIR/vecna"
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT INT TERM

info "Release: v${TAG}"
info "Downloading ${ASSET}..."
curl -fL --progress-bar "$ARCHIVE_URL" -o "$ARCHIVE"
success "Binary archive downloaded"

info "Downloading SHA-256 checksums..."
curl -fL --progress-bar "$CHECKSUM_URL" -o "$CHECKSUMS"
success "Checksums downloaded"

info "Verifying SHA-256 checksum..."
EXPECTED="$(awk -v file="$ASSET" '$2 == file { print $1; exit }' "$CHECKSUMS")"
[ -n "$EXPECTED" ] || fail "No checksum found for $ASSET."

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
else
  fail "A SHA-256 utility (sha256sum or shasum) is required."
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  printf '[vecna] ✗ Checksum verification failed for %s.\n' "$ASSET" >&2
  printf '[vecna]   Expected: %s\n' "$EXPECTED" >&2
  printf '[vecna]   Actual:   %s\n' "$ACTUAL" >&2
  exit 1
fi
success "Checksum verified"

info "Extracting Vecna..."
tar -xzf "$ARCHIVE" -C "$TMP_DIR" vecna
[ -f "$EXTRACTED" ] || fail "Install failed: vecna binary was not found in the release archive."
chmod +x "$EXTRACTED"
success "Archive extracted"

info "Installing to $BIN_DIR/vecna..."
mkdir -p "$BIN_DIR"
install -m 0755 "$EXTRACTED" "$BIN_DIR/vecna"
[ -x "$BIN_DIR/vecna" ] || fail "Install failed: $BIN_DIR/vecna is not executable."
success "Vecna v${TAG} installed successfully"

case ":${PATH:-}:" in
  *":$BIN_DIR:"*) ;;
  *)
    printf '\n'
    info "Note: $BIN_DIR is not currently in PATH."
    info "Add it to your shell profile, then run: vecna version"
    ;;
esac
