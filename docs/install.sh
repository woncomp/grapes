#!/bin/sh

set -eu

OWNER="woncomp"
REPO="grapes"
API_URL="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
BIN_NAME="grapes"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL=0
INSTALL_DIR="${GRAPES_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
TMP_DIR=""

say() {
  printf '%s\n' "$*"
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

usage() {
  printf '%s\n' "Usage: install.sh [--user]"
  printf '%s\n' "  --user    Install to \$HOME/.local/bin instead of $DEFAULT_INSTALL_DIR"
}

cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}

have() {
  command -v "$1" >/dev/null 2>&1
}

install_failed() {
  if [ "$USER_INSTALL" -eq 0 ] && [ "$INSTALL_DIR" = "$DEFAULT_INSTALL_DIR" ]; then
    fail "could not install to $DEFAULT_INSTALL_DIR; rerun with sudo or pass --user to install into \$HOME/.local/bin"
  fi

  fail "could not install to $INSTALL_DIR"
}

fetch_text() {
  if have curl; then
    curl -fsSL "$1"
    return
  fi

  if have wget; then
    wget -qO- "$1"
    return
  fi

  fail "curl or wget is required"
}

download_file() {
  if have curl; then
    curl -fsSL -o "$2" "$1"
    return
  fi

  if have wget; then
    wget -qO "$2" "$1"
    return
  fi

  fail "curl or wget is required"
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *)
      fail "unsupported operating system: $(uname -s)"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *)
      fail "unsupported architecture: $(uname -m)"
      ;;
  esac
}

verify_sha256() {
  expected_hash="$1"
  file_path="$2"

  if have sha256sum; then
    actual_hash="$(sha256sum "$file_path" | awk '{print $1}')"
  elif have shasum; then
    actual_hash="$(shasum -a 256 "$file_path" | awk '{print $1}')"
  elif have openssl; then
    actual_hash="$(openssl dgst -sha256 "$file_path" | awk '{print $NF}')"
  else
    say "warning: no SHA-256 tool found; skipping checksum verification"
    return
  fi

  if [ "$actual_hash" != "$expected_hash" ]; then
    fail "checksum verification failed for $file_path"
  fi
}

trap cleanup EXIT INT TERM

while [ "$#" -gt 0 ]; do
  case "$1" in
    --user)
      USER_INSTALL=1
      INSTALL_DIR="$HOME/.local/bin"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
  shift
done

OS="$(detect_os)"
ARCH="$(detect_arch)"
ARCHIVE_EXT="tar.gz"
ARCHIVE_NAME_SUFFIX="_${OS}_${ARCH}.${ARCHIVE_EXT}"

TMP_DIR="$(mktemp -d)"
RELEASE_JSON="$(fetch_text "$API_URL")"
DOWNLOAD_URLS="$(
  printf '%s' "$RELEASE_JSON" \
    | grep -o '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*"' \
    | sed 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/'
)"

ASSET_URL=""
CHECKSUM_URL=""

OLD_IFS="$IFS"
IFS='
'
for url in $DOWNLOAD_URLS; do
  name=${url##*/}
  case "$name" in
    grapes_*"$ARCHIVE_NAME_SUFFIX")
      ASSET_URL="$url"
      ;;
    grapes_*_checksums.txt)
      CHECKSUM_URL="$url"
      ;;
  esac
done
IFS="$OLD_IFS"

[ -n "$ASSET_URL" ] || fail "could not find a release asset for ${OS}/${ARCH}"
[ -n "$CHECKSUM_URL" ] || fail "could not find the release checksum file"

ASSET_NAME=${ASSET_URL##*/}
ARCHIVE_PATH="$TMP_DIR/$ASSET_NAME"
CHECKSUM_PATH="$TMP_DIR/${CHECKSUM_URL##*/}"

download_file "$ASSET_URL" "$ARCHIVE_PATH"
download_file "$CHECKSUM_URL" "$CHECKSUM_PATH"

EXPECTED_HASH="$(awk -v file="$ASSET_NAME" '$2 == file { print $1 }' "$CHECKSUM_PATH")"
[ -n "$EXPECTED_HASH" ] || fail "could not find checksum entry for $ASSET_NAME"

verify_sha256 "$EXPECTED_HASH" "$ARCHIVE_PATH"

EXTRACT_DIR="$TMP_DIR/extract"
mkdir -p "$EXTRACT_DIR"

have tar || fail "tar is required"
tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"

BINARY_PATH="$(find "$EXTRACT_DIR" -type f -name "$BIN_NAME" | head -n 1)"
[ -n "$BINARY_PATH" ] || fail "could not find $BIN_NAME in the extracted archive"

mkdir -p "$INSTALL_DIR" || install_failed
RESOLVED_INSTALL_DIR="$(cd "$INSTALL_DIR" && pwd)"
DESTINATION_PATH="$RESOLVED_INSTALL_DIR/$BIN_NAME"

if have install; then
  install -m 0755 "$BINARY_PATH" "$DESTINATION_PATH" || install_failed
else
  cp "$BINARY_PATH" "$DESTINATION_PATH" || install_failed
  chmod 0755 "$DESTINATION_PATH" || install_failed
fi

say "Installed $BIN_NAME to $DESTINATION_PATH"

if [ "$USER_INSTALL" -eq 1 ] && ! have "$BIN_NAME"; then
  say "Add $HOME/.local/bin to PATH to run $BIN_NAME directly."
fi
