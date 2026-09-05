#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-release}"

if ! command -v go >/dev/null 2>&1; then
  echo "未找到 Go。请在构建机安装 Go 1.22+；最终用户不需要安装 Go。" >&2
  exit 1
fi

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR/windows" "$OUT_DIR/linux"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$OUT_DIR/windows/v2ray-web.exe" .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$OUT_DIR/linux/v2ray-web" .

cp -R v2ray-windows-64 "$OUT_DIR/windows/"
cp -R v2ray-linux-64 "$OUT_DIR/linux/"
cp README.md "$OUT_DIR/windows/README.md"
cp README.md "$OUT_DIR/linux/README.md"
cat > "$OUT_DIR/README.txt" <<'EOF'
Windows: v2ray-web.exe
Linux: ./v2ray-web

The Go toolchain is not required on the target machine.
EOF

echo "发布文件已生成到 $OUT_DIR"
