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
arch="${GOARCH:-$(go env GOARCH)}"
# Match the release workflow: v1 loads on CPUs without AVX2/BMI2.
if [ "$arch" = "amd64" ] && [ -z "${GOAMD64:-}" ]; then
  export GOAMD64=v1
fi
case "$(uname -s)" in
  Darwin) EXT=dylib ;;
  *) EXT=so ;;
esac
OUT="$OUT_DIR/${NAME}.${EXT}"
echo "Building $OUT (CGO_ENABLED=1, GOAMD64=${GOAMD64:-}, buildmode=c-shared)"
go build -buildmode=c-shared -o "$OUT" .
echo "OK $OUT"