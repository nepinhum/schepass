#!/usr/bin/env sh
set -eu

BIN_NAME="schepass"
SRC_BIN="${SRC_BIN:-./schepass}"
PREFIX="${PREFIX:-/usr/local}"
DEST_DIR="${DEST_DIR:-$PREFIX/bin}"

if [ "${1:-}" = "--user" ]; then
  DEST_DIR="${DEST_DIR:-$HOME/.local/bin}"
  shift
fi

if [ "${1:-}" = "--prefix" ] && [ -n "${2:-}" ]; then
  DEST_DIR="$2/bin"
  shift 2
fi

if [ ! -f "$SRC_BIN" ]; then
  echo "error: build the binary first (expected $SRC_BIN)" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"
install -m 0755 "$SRC_BIN" "$DEST_DIR/$BIN_NAME"
echo "installed to $DEST_DIR/$BIN_NAME"
