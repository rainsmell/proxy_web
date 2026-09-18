param(
    [string]$OutDir = "dist\fnos",
    [string]$AppName = "myapp",
    [string]$Version = "0.1.0",
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

function Write-Utf8NoBom([string]$Path, [string]$Text) {
    $enc = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $Text, $enc)
}

function Resolve-Go() {
    $cmd = Get-Command go -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $default = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $default) { return $default }
    throw "Go was not found. Install Go or add go.exe to PATH."
}

function Resolve-Fnpack() {
    if ($FnpackPath) {
        if (Test-Path $FnpackPath) { return (Resolve-Path $FnpackPath).Path }
        $joined = Join-Path $Root $FnpackPath
        if (Test-Path $joined) { return (Resolve-Path $joined).Path }
        throw "fnpack not found: $FnpackPath"
    }
    foreach ($candidate in @(
        (Join-Path $Root "fnos-utils\fnpack.exe"),
        (Join-Path $Root "fnos-utils\fnpack-*-windows-amd64"),
        (Join-Path $Root "fnos-utils\fnpack")
    )) {
        $hit = Get-Item $candidate -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($hit) { return $hit.FullName }
    }
    throw "fnpack not found. Put it under fnos-utils or pass -FnpackPath."
}

Assert-InRoot $OutFull
Assert-InRoot $StageDir
if (Test-Path $StageDir) { Remove-Item -LiteralPath $StageDir -Recurse -Force }
New-Item -ItemType Directory -Force $OutFull, $StageDir | Out-Null
Copy-Item -Path (Join-Path $TemplateDir "*") -Destination $StageDir -Recurse -Force
New-Item -ItemType Directory -Force (Join-Path $StageDir "wizard") | Out-Null

$manifest = Join-Path $StageDir "manifest"
$manifestText = (Get-Content $manifest -Raw) -replace '(?m)^version\s*=.*$', "version               = $Version"
Write-Utf8NoBom $manifest $manifestText

$appDir = Join-Path $StageDir "app"
New-Item -ItemType Directory -Force $appDir | Out-Null

$go = Resolve-Go
$oldGOOS = $env:GOOS; $oldGOARCH = $env:GOARCH; $oldCGO = $env:CGO_ENABLED
try {
    $env:GOOS = "linux"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
    & $go build -trimpath -ldflags "-s -w" -o (Join-Path $appDir $AppName) $Root
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
}
finally {
    $env:GOOS = $oldGOOS; $env:GOARCH = $oldGOARCH; $env:CGO_ENABLED = $oldCGO
}

# Copy required runtime assets here, e.g.:
# Copy-Item -Path (Join-Path $Root "runtime-linux-amd64") -Destination (Join-Path $appDir "runtime-linux-amd64") -Recurse -Force

$fnpack = Resolve-Fnpack
Push-Location $OutFull
try {
    $packOutput = & $fnpack build -d $StageDir 2>&1
    $packOutput | ForEach-Object { Write-Host $_ }
    if ($LASTEXITCODE -ne 0) { throw "fnpack build failed with exit code $LASTEXITCODE" }
    if (($packOutput -join "`n") -match "(?i)packing failed|build failed|error") { throw "fnpack reported failure" }
}
finally { Pop-Location }

$fpk = Get-ChildItem -Path $OutFull -Filter "*.fpk" -File | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $fpk) { throw "No .fpk was generated" }
Write-Host "FPK generated: $($fpk.FullName)"
