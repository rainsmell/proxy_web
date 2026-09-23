param(
    [string]$OutDir = "release",
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "未找到 Go。请在构建机安装 Go 1.22+；最终用户不需要安装 Go。"
}

if (Test-Path $OutDir) {
    Remove-Item -LiteralPath $OutDir -Recurse -Force
}
New-Item -ItemType Directory -Path "$OutDir\windows" -Force | Out-Null
New-Item -ItemType Directory -Path "$OutDir\linux" -Force | Out-Null

$env:CGO_ENABLED = "0"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w -X main.appVersion=$Version" -o "$OutDir\windows\v2ray-web.exe" .

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w -X main.appVersion=$Version" -o "$OutDir\linux\v2ray-web" .

if (-not (Test-Path "mihomo-linux-64\mihomo") -or -not (Test-Path "mihomo-windows-64\mihomo.exe")) {
    Write-Host "未找到 mihomo 内核，正在尝试下载..."
    & (Join-Path $PSScriptRoot "scripts\fetch-core.ps1") -Target all
}
if (-not (Test-Path "mihomo-linux-64\mihomo") -or -not (Test-Path "mihomo-windows-64\mihomo.exe")) {
    throw "缺少 mihomo 内核。请运行 scripts\fetch-core.ps1 或手动放置 mihomo-linux-64\mihomo 与 mihomo-windows-64\mihomo.exe。"
}

Copy-Item -Recurse -Force "mihomo-windows-64" "$OutDir\windows\mihomo-windows-64"
Copy-Item -Recurse -Force "mihomo-linux-64" "$OutDir\linux\mihomo-linux-64"
Copy-Item -Force "README.md" "$OutDir\windows\README.md"
Copy-Item -Force "README.md" "$OutDir\linux\README.md"

@"
Windows:
  v2ray-web.exe

Linux:
  ./v2ray-web

Proxy core:
  mihomo (Clash.Meta)

The Go toolchain is not required on the target machine.
"@ | Set-Content -Encoding UTF8 "$OutDir\README.txt"

Write-Host "发布文件已生成到 $OutDir"
