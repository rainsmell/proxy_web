#!/usr/bin/env bash
set -euo pipefail

# Download mihomo release binaries into the bundled core directories.
#
# Usage: scripts/fetch-core.sh [all|linux|windows|host]
#
# Environment:
#   MIHOMO_VERSION  release tag, default v1.19.31
#   MIHOMO_VARIANT  compatible (default) or default
#                   - compatible: GOAMD64=v1, runs on older NAS CPUs that
#                     lack x86-64-v3 (AVX2). Starts with "This program can
#                     only be run on AMD64 processors with v3 microarchitecture
#                     support." -> use this.
#                   - default: upstream amd64 build, requires x86-64-v3.
#   MIHOMO_MIRROR   optional URL prefix tried first, e.g. https://gh-proxy.com/

VERSION="${MIHOMO_VERSION:-v1.19.31}"
TARGET="${1:-all}"
VARIANT="${MIHOMO_VARIANT:-compatible}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIRROR="${MIHOMO_MIRROR:-}"

case "$VARIANT" in
  compatible|c) SUFFIX="-compatible" ;;
  ""|default|v3|normal) SUFFIX="" ;;
  *) echo "unknown MIHOMO_VARIANT: $VARIANT (use compatible or default)" >&2; exit 1 ;;
esac

if [ "$TARGET" = "host" ]; then
  case "$(uname -s)-$(uname -m)" in
    Linux-x86_64|Linux-amd64) TARGET=linux ;;
    *) echo "unsupported host: $(uname -s)-$(uname -m)" >&2; exit 1 ;;
  esac
fi

MIRRORS=()
[ -n "$MIRROR" ] && MIRRORS+=("$MIRROR")
MIRRORS+=("" "https://gh-proxy.com/" "https://ghfast.top/")

fetch() {
  local url="$1" dest="$2" base
  for base in "${MIRRORS[@]}"; do
    local u="${base}${url}"
    echo "downloading $u"
    if command -v curl >/dev/null 2>&1; then
      curl -fL --retry 2 --connect-timeout 15 -o "$dest" "$u" && return 0
    elif command -v wget >/dev/null 2>&1; then
      wget -O "$dest" "$u" && return 0
    else
      echo "curl or wget is required" >&2
      exit 1
    fi
    echo "  failed, trying next mirror..."
  done
  echo "all download attempts failed for $url" >&2
  return 1
}

fetch_linux() {
  local base="https://github.com/MetaCubeX/mihomo/releases/download/${VERSION}"
  local tmp
  tmp="$(mktemp -d)"
  mkdir -p "$ROOT/mihomo-linux-64"
  fetch "${base}/mihomo-linux-amd64${SUFFIX}-${VERSION}.gz" "$tmp/mihomo.gz"
  gzip -dc "$tmp/mihomo.gz" > "$ROOT/mihomo-linux-64/mihomo"
  chmod +x "$ROOT/mihomo-linux-64/mihomo"
  rm -rf "$tmp"
  echo "installed $ROOT/mihomo-linux-64/mihomo (variant: ${VARIANT})"
}

fetch_windows() {
  local base="https://github.com/MetaCubeX/mihomo/releases/download/${VERSION}"
  local tmp
  tmp="$(mktemp -d)"
  mkdir -p "$ROOT/mihomo-windows-64"
  fetch "${base}/mihomo-windows-amd64${SUFFIX}-${VERSION}.zip" "$tmp/mihomo.zip"
  unzip -o -q "$tmp/mihomo.zip" -d "$tmp/extract"
  local exe
  exe="$(find "$tmp/extract" -iname 'mihomo*.exe' | head -n1)"
  [ -n "$exe" ] || { echo "mihomo.exe not found in archive" >&2; exit 1; }
  cp "$exe" "$ROOT/mihomo-windows-64/mihomo.exe"
  rm -rf "$tmp"
  echo "installed $ROOT/mihomo-windows-64/mihomo.exe (variant: ${VARIANT})"
}

case "$TARGET" in
  all) fetch_linux; fetch_windows ;;
  linux) fetch_linux ;;
  windows) fetch_windows ;;
  *) echo "usage: $0 [all|linux|windows|host]" >&2; exit 1 ;;
esac
