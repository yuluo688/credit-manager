# PowerShell build for Windows native plugin DLL.
param(
    [string]$OutDir = "dist",
    [string]$Name = "credit-manager"
)

$ErrorActionPreference = "Stop"
if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    Write-Error "CGO requires a C compiler on PATH (e.g. MinGW-w64 gcc). Install a toolchain matching windows/amd64, then retry."
}
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$env:CGO_ENABLED = "1"
$out = Join-Path $OutDir ($Name + ".dll")
Write-Host "Building $out (CGO_ENABLED=1, buildmode=c-shared)"
go build -buildmode=c-shared -o $out .
Write-Host "OK $out"