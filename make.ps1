<#
.SYNOPSIS
    Cross-platform build script for backend-go (Windows PowerShell / pwsh).
.DESCRIPTION
    Usage:  .\make.ps1 <target>
    Targets: setup, deps, web, generate, build, dev, run, clean, help
#>
param(
    [Parameter(Position=0)]
    [ValidateSet("setup","deps","web","generate","build","dev","run","clean","help")]
    [string]$Target = "help"
)

$ErrorActionPreference = "Stop"

$OutDir  = "dist"
$Binary  = "$OutDir\backend-go.exe"

try   { $Commit = (git rev-parse --short=12 HEAD 2>$null) } catch { $Commit = "none" }
$Version = if ($env:VERSION) { $env:VERSION } else { "dev" }
$Build   = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LDFlags = "-s -w -X main.Version=$Version -X main.Commit=$Commit -X main.Build=$Build"

function Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }

function Do-Deps {
    Step "Downloading Go modules"
    go mod download;  if ($LASTEXITCODE) { throw "go mod download failed" }
    go mod tidy;      if ($LASTEXITCODE) { throw "go mod tidy failed" }
}

function Do-Web {
    Step "Building web SPA"
    Push-Location web
    try {
        pnpm install --no-frozen-lockfile; if ($LASTEXITCODE) { throw "pnpm install failed" }
        pnpm build;                        if ($LASTEXITCODE) { throw "pnpm build failed" }
    } finally { Pop-Location }
}

function Do-Generate {
    Step "Generating module imports"
    go generate ./internal/bootstrap/mod; if ($LASTEXITCODE) { throw "go generate failed" }
}

function Do-Setup {
    Do-Deps
    Do-Web
    Do-Generate
    Step "Setup complete"
}

function Do-Build {
    Do-Setup
    Step "Building binary"
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags $LDFlags -o $Binary ./cmd/server
    if ($LASTEXITCODE) { throw "go build failed" }
    Step "Built $Binary"
}

function Do-Dev {
    Do-Deps
    Do-Generate
    Step "Building binary (dev)"
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    go build -ldflags $LDFlags -o $Binary ./cmd/server
    if ($LASTEXITCODE) { throw "go build failed" }
    Step "Built $Binary"
}

function Do-Run {
    Do-Dev
    Step "Running"
    & $Binary
}

function Do-Clean {
    Step "Cleaning"
    if (Test-Path $OutDir)  { Remove-Item -Recurse -Force $OutDir }
    $auto = "internal\bootstrap\mod\autogen_imports.go"
    if (Test-Path $auto)    { Remove-Item -Force $auto }
}

function Do-Help {
    Write-Host ""
    Write-Host "Usage: .\make.ps1 <target>" -ForegroundColor White
    Write-Host ""
    Write-Host "  setup       Full init: deps + web SPA + codegen"
    Write-Host "  deps        Download and tidy Go modules"
    Write-Host "  web         Install web deps and build admin SPA"
    Write-Host "  generate    Generate module import file (autogen_imports.go)"
    Write-Host "  build       Full build: setup + compile binary"
    Write-Host "  dev         Quick dev build (skip web)"
    Write-Host "  run         Build and run"
    Write-Host "  clean       Remove build artifacts"
    Write-Host "  help        Show this help"
    Write-Host ""
}

switch ($Target) {
    "setup"    { Do-Setup }
    "deps"     { Do-Deps }
    "web"      { Do-Web }
    "generate" { Do-Generate }
    "build"    { Do-Build }
    "dev"      { Do-Dev }
    "run"      { Do-Run }
    "clean"    { Do-Clean }
    "help"     { Do-Help }
}
