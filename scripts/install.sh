#!/usr/bin/env sh
set -eu

REPO="zhravan/vecna"
VERSION="${VERSION:-latest}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "tar is required." >&2; exit 1; }

if [ "$VERSION" = "latest" ]; then
  TAG="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$TAG" ] || { echo "Unable to determine the latest Vecna release." >&2; exit 1; }
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

echo "Downloading Vecna v${TAG} (${OS}/${ARCH})..."
curl -fsSL "$ARCHIVE_URL" -o "$ARCHIVE"
curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUMS"

EXPECTED="$(awk -v file="$ASSET" '$2 == file { print $1; exit }' "$CHECKSUMS")"
[ -n "$EXPECTED" ] || { echo "No checksum found for $ASSET." >&2; exit 1; }

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
else
  echo "A SHA-256 utility (sha256sum or shasum) is required." >&2
  exit 1
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum verification failed for $ASSET." >&2
  echo "Expected: $EXPECTED" >&2
  echo "Actual:   $ACTUAL" >&2
  exit 1
fi

mkdir -p "$BIN_DIR"
tar -xzf "$ARCHIVE" -C "$TMP_DIR" vecna
[ -f "$EXTRACTED" ] || { echo "Install failed: vecna binary was not found in the release archive." >&2; exit 1; }
chmod +x "$EXTRACTED"
install -m 0755 "$EXTRACTED" "$BIN_DIR/vecna"
[ -x "$BIN_DIR/vecna" ] || { echo "Install failed: $BIN_DIR/vecna is not executable." >&2; exit 1; }

echo "Installed Vecna v${TAG} to $BIN_DIR/vecna"
case ":${PATH:-}:" in
  *":$BIN_DIR:"*) ;;
  *) echo; echo "Note: $BIN_DIR is not currently in PATH."; echo "Add it to your shell profile, then run: vecna version" ;;
esac
