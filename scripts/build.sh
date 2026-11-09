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
# Match the release workflow. GOAMD64 does not reach gcc; runtime/cgo needs
# -march=x86-64 or a newer GCC can put BMI2 in x_cgo_sigaction.
if [ "$arch" = "amd64" ]; then
  if [ -z "${GOAMD64:-}" ]; then
    export GOAMD64=v1
  fi
  for var in CGO_CFLAGS CGO_CXXFLAGS; do
    eval "current=\${$var:-}"
    case " $current " in
      *" -march="*) ;;
      *)
        if [ -z "$current" ]; then
          export "$var=-O2 -g -march=x86-64"
        else
          export "$var=-O2 -g -march=x86-64 $current"
        fi
        ;;
    esac
  done
fi
case "$(uname -s)" in
  Darwin) EXT=dylib ;;
  *) EXT=so ;;
esac
OUT="$OUT_DIR/${NAME}.${EXT}"
echo "Building $OUT (CGO_ENABLED=1, GOAMD64=${GOAMD64:-}, CGO_CFLAGS=${CGO_CFLAGS:-}, buildmode=c-shared)"
go build -buildmode=c-shared -o "$OUT" .
if [ "$arch" = "amd64" ] && [ "$(uname -s)" = "Linux" ]; then
  section="$(objdump -d --no-show-raw-insn "$OUT" | sed -n '/<x_cgo_sigaction>:/,/^$/p')"
  test -n "$section"
  if printf '%s\n' "$section" | grep -qE '\b(shlx|mulx|andn|bzhi|bextr|sarx|pdep|pext)\b'; then
    echo "BMI2 leaked into x_cgo_sigaction" >&2
    exit 1
  fi
fi
echo "OK $OUT"