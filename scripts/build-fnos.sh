#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT/dist/fnos}"
APP_NAME="${APP_NAME:-v2rayweb}"
VERSION="${VERSION:-0.1.4}"
FNPACK_BIN="${FNPACK_BIN:-$ROOT/fnos-utils/fnpack}"
if [[ ! -x "$FNPACK_BIN" && -x "$ROOT/fnos-utils/fnpack-1.2.3-linux-amd64" ]]; then
  FNPACK_BIN="$ROOT/fnos-utils/fnpack-1.2.3-linux-amd64"
fi
STAGE="$OUT_DIR/$APP_NAME"

command -v go >/dev/null 2>&1 || { echo "go not found" >&2; exit 1; }
[[ -x "$FNPACK_BIN" ]] || { echo "fnpack not found/executable: $FNPACK_BIN" >&2; exit 1; }
[[ -f "$ROOT/v2ray-linux-64/v2ray" ]] || { echo "missing v2ray-linux-64/v2ray" >&2; exit 1; }

rm -rf "$STAGE"
mkdir -p "$OUT_DIR" "$STAGE"
cp -R "$ROOT/packaging/fnos/"* "$STAGE/"
mkdir -p "$STAGE/wizard"
sed -i -E "s/^version[[:space:]]*=.*$/version               = $VERSION/" "$STAGE/manifest"

mkdir -p "$STAGE/app"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.appVersion=$VERSION" -o "$STAGE/app/v2ray-web" "$ROOT"
cp -R "$ROOT/v2ray-linux-64" "$STAGE/app/v2ray-linux-64"
chmod +x "$STAGE/app/v2ray-web" "$STAGE/app/v2ray-linux-64/v2ray" "$STAGE/cmd/"*

(cd "$OUT_DIR" && "$FNPACK_BIN" build -d "$STAGE")
FPK="$(ls -t "$OUT_DIR"/*.fpk | head -n 1)"
ls -lh "$FPK"
