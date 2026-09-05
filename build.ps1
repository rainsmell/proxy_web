param(
    [string]$OutDir = "release"
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
go build -trimpath -ldflags "-s -w" -o "$OutDir\windows\v2ray-web.exe" .

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o "$OutDir\linux\v2ray-web" .

Copy-Item -Recurse -Force "v2ray-windows-64" "$OutDir\windows\v2ray-windows-64"
Copy-Item -Recurse -Force "v2ray-linux-64" "$OutDir\linux\v2ray-linux-64"
Copy-Item -Force "README.md" "$OutDir\windows\README.md"
Copy-Item -Force "README.md" "$OutDir\linux\README.md"

@"
Windows:
  v2ray-web.exe

Linux:
  ./v2ray-web

The Go toolchain is not required on the target machine.
"@ | Set-Content -Encoding UTF8 "$OutDir\README.txt"

Write-Host "发布文件已生成到 $OutDir"
