param(
    [string]$Version = "v1.19.31",
    [ValidateSet("compatible", "default")]
    [string]$Variant = "compatible",
    [ValidateSet("all", "linux", "windows")]
    [string]$Target = "all",
    [string]$Mirror = ""
)

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$base = "https://github.com/MetaCubeX/mihomo/releases/download/$Version"
$suffix = if ($Variant -eq "compatible") { "-compatible" } else { "" }

$mirrors = @()
if ($Mirror) { $mirrors += $Mirror }
$mirrors += @("", "https://gh-proxy.com/", "https://ghfast.top/")

function Get-RemoteFile([string]$url, [string]$dest) {
    foreach ($prefix in $mirrors) {
        $u = "$prefix$url"
        Write-Host "downloading $u"
        try {
            Invoke-WebRequest -Uri $u -OutFile $dest -UseBasicParsing
            return
        } catch {
            Write-Host "  failed, trying next mirror..."
        }
    }
    throw "all download attempts failed for $url"
}

if ($Target -in @("all", "linux")) {
    $dir = Join-Path $Root "mihomo-linux-64"
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $gz = Join-Path $env:TEMP "mihomo-linux-amd64$suffix-$Version.gz"
    Get-RemoteFile "$base/mihomo-linux-amd64$suffix-$Version.gz" $gz
    $in = [System.IO.File]::OpenRead($gz)
    try {
        $gzs = New-Object System.IO.Compression.GZipStream($in, [System.IO.Compression.CompressionMode]::Decompress)
        $out = [System.IO.File]::Create((Join-Path $dir "mihomo"))
        try { $gzs.CopyTo($out) } finally { $out.Close(); $gzs.Close() }
    } finally { $in.Close() }
    Write-Host "installed $dir\mihomo (variant: $Variant)"
}

if ($Target -in @("all", "windows")) {
    $dir = Join-Path $Root "mihomo-windows-64"
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $zip = Join-Path $env:TEMP "mihomo-windows-amd64$suffix-$Version.zip"
    Get-RemoteFile "$base/mihomo-windows-amd64$suffix-$Version.zip" $zip
    $extract = Join-Path $env:TEMP "mihomo-win-extract"
    if (Test-Path $extract) { Remove-Item -Recurse -Force $extract }
    Expand-Archive -Path $zip -DestinationPath $extract -Force
    $exe = Get-ChildItem -Path $extract -Recurse -Filter "mihomo*.exe" | Select-Object -First 1
    if (-not $exe) { throw "mihomo.exe not found in archive" }
    Copy-Item $exe.FullName (Join-Path $dir "mihomo.exe") -Force
    Write-Host "installed $dir\mihomo.exe (variant: $Variant)"
}
