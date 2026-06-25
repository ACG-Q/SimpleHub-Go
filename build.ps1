param(
    [string]$Target = "",
    [string]$OutputDir = ""
)

$ErrorActionPreference = "Stop"
$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Step($msg) {
    Write-Host "==> $msg" -ForegroundColor Cyan
}

function Write-Error($msg) {
    Write-Host "ERROR: $msg" -ForegroundColor Red
    exit 1
}

# --- Build frontend ---
Write-Step "Building frontend..."
Push-Location "$RootDir\web"
try {
    if (-not (Test-Path "node_modules")) {
        npm ci
    }
    npm run build
    if ($LASTEXITCODE -ne 0) { Write-Error "Frontend build failed" }
} finally {
    Pop-Location
}

# --- Copy dist to embed location ---
Write-Step "Copying frontend dist to embed location..."
if (Test-Path "$RootDir\cmd\server\dist") {
    Remove-Item -Recurse -Force "$RootDir\cmd\server\dist"
}
Copy-Item -Recurse "$RootDir\web\dist" "$RootDir\cmd\server\dist"

# --- Determine build flags ---
$env:CGO_ENABLED = "0"

if ($Target) {
    $parts = $Target -split "/"
    $os = $parts[0]
    $arch = $parts[1]
    $env:GOOS = $os
    $env:GOARCH = $arch
} else {
    $env:GOOS = ""
    $env:GOARCH = ""
}

if (-not $OutputDir) {
    $OutputDir = "$RootDir\bin"
}
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

$outputName = "server"
$isWin = $env:OS -match "Windows" -or [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT
if ($env:GOOS -eq "windows" -or (-not $env:GOOS -and $isWin)) {
    $outputName = "server.exe"
}
$outputPath = "$OutputDir\$outputName"

# --- Build Go binary ---
Write-Step "Building Go binary ($outputPath)..."
Push-Location $RootDir
try {
    if ($Target) {
        Write-Host "  Target: $Target"
    }
    go build -o $outputPath -ldflags="-s -w" ./cmd/server
    if ($LASTEXITCODE -ne 0) { Write-Error "Go build failed" }
} finally {
    Pop-Location
}

# --- Restore embed placeholder ---
Remove-Item -Recurse -Force "$RootDir\cmd\server\dist"
New-Item -ItemType Directory -Path "$RootDir\cmd\server\dist" -Force | Out-Null
Set-Content -Path "$RootDir\cmd\server\dist\index.html" -Value '<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"/><title>SimpleHub</title></head><body><div id="root">Placeholder</div></body></html>' -Encoding UTF8

$fileSize = (Get-Item $outputPath).Length / 1MB
Write-Host ""
Write-Host "Done! Binary: $outputPath ($([math]::Round($fileSize, 1)) MB)" -ForegroundColor Green
Write-Host "Run: $outputPath" -ForegroundColor Green
