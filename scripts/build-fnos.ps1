param(
    [string]$OutDir = "dist\fnos",
    [string]$AppName = "v2rayweb",
    [string]$Version = "0.1.6",
    [string]$FnpackPath = ""
)

$ErrorActionPreference = "Stop"

$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$TemplateDir = Join-Path $Root "packaging\fnos"
$OutFull = [System.IO.Path]::GetFullPath((Join-Path $Root $OutDir))
$StageDir = Join-Path $OutFull $AppName
$RootFull = [System.IO.Path]::GetFullPath($Root)

function Assert-InRoot([string]$Path) {
    $full = [System.IO.Path]::GetFullPath($Path)
    if (-not $full.StartsWith($RootFull, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to touch path outside workspace: $full"
    }
}

function Resolve-Go() {
    $cmd = Get-Command go -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $default = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $default) { return $default }
    throw "Go was not found. Install Go or add go.exe to PATH."
}

function Write-Utf8NoBom([string]$Path, [string]$Text) {
    $enc = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $Text, $enc)
}

function Resolve-Fnpack() {
    if ($FnpackPath) {
        $p = [System.IO.Path]::GetFullPath((Join-Path $Root $FnpackPath))
        if (Test-Path $p) { return $p }
        if (Test-Path $FnpackPath) { return (Resolve-Path $FnpackPath).Path }
        throw "fnpack not found: $FnpackPath"
    }
    foreach ($candidate in @(
        (Join-Path $Root "fnos-utils\fnpack.exe"),
        (Join-Path $Root "fnos-utils\fnpack-1.2.3-windows-amd64"),
        (Join-Path $Root "fnos-utils\fnpack")
    )) {
        if (Test-Path $candidate) {
            $resolved = (Resolve-Path $candidate).Path
            if ([System.IO.Path]::GetExtension($resolved) -eq "") {
                $shim = Join-Path $OutFull "fnpack.exe"
                Copy-Item -Force $resolved $shim
                return $shim
            }
            return $resolved
        }
    }
    throw "fnpack not found. Put it under fnos-utils or pass -FnpackPath."
}

if (-not (Test-Path $TemplateDir)) { throw "Missing template dir: $TemplateDir" }
if (-not (Test-Path (Join-Path $Root "v2ray-linux-64\v2ray"))) { throw "Missing Linux V2Ray core: v2ray-linux-64\v2ray" }

Assert-InRoot $OutFull
Assert-InRoot $StageDir
if (Test-Path $StageDir) {
    Remove-Item -LiteralPath $StageDir -Recurse -Force
}
New-Item -ItemType Directory -Force $OutFull | Out-Null
New-Item -ItemType Directory -Force $StageDir | Out-Null

Copy-Item -Path (Join-Path $TemplateDir "*") -Destination $StageDir -Recurse -Force
New-Item -ItemType Directory -Force (Join-Path $StageDir "wizard") | Out-Null

$manifest = Join-Path $StageDir "manifest"
$manifestText = (Get-Content $manifest -Raw) -replace '(?m)^version\s*=.*$', "version               = $Version"
Write-Utf8NoBom $manifest $manifestText

$appDir = Join-Path $StageDir "app"
New-Item -ItemType Directory -Force $appDir | Out-Null

$go = Resolve-Go
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$oldCGO = $env:CGO_ENABLED
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    & $go build -trimpath -ldflags "-s -w -X main.appVersion=$Version" -o (Join-Path $appDir "v2ray-web") $Root
    if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }
}
finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    $env:CGO_ENABLED = $oldCGO
}

Copy-Item -Path (Join-Path $Root "v2ray-linux-64") -Destination (Join-Path $appDir "v2ray-linux-64") -Recurse -Force

$fnpack = Resolve-Fnpack
Push-Location $OutFull
try {
    $packOutput = & $fnpack build -d $StageDir 2>&1
    $packOutput | ForEach-Object { Write-Host $_ }
    if ($LASTEXITCODE -ne 0) { throw "fnpack build failed with exit code $LASTEXITCODE" }
    if (($packOutput -join "`n") -match "(?i)packing failed|build failed|error") { throw "fnpack reported failure" }
}
finally {
    Pop-Location
}

$fpk = Get-ChildItem -Path $OutFull -Filter "*.fpk" -File | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $fpk) {
    throw "fnpack finished but no .fpk was found under $OutFull"
}

Write-Host "FPK generated: $($fpk.FullName)"
