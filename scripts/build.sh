#!/usr/bin/env bash
set -euo pipefail
OUT_DIR="${1:-dist}"
NAME="${2:-credit-manager}"
if ! command -v gcc >/dev/null 2>&1 && ! command -v clang >/dev/null 2>&1; then
  echo "CGO requires a C compiler (gcc or clang) matching the target OS/ARCH" >&2
  exit 1
fi
mkdir -p "$OUT_DIR"
export CGO_ENABLED=1
case "$(uname -s)" in
  Darwin) EXT=dylib ;;
  *) EXT=so ;;
esac
OUT="$OUT_DIR/${NAME}.${EXT}"
echo "Building $OUT (CGO_ENABLED=1, buildmode=c-shared)"
go build -buildmode=c-shared -o "$OUT" .
echo "OK $OUT"