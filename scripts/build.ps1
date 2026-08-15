# PowerShell build for Windows native plugin DLL.
param(
    [string]$OutDir = "dist",
    [string]$Name = "credit-manager"
)

$ErrorActionPreference = "Stop"
$compiler = Get-Command gcc -ErrorAction SilentlyContinue
if ($compiler) {
    $env:CC = $compiler.Source
} else {
    $zig = Get-Command zig -ErrorAction SilentlyContinue
    $zigPath = if ($zig) { $zig.Source } else { $null }
    if (-not $zig) {
        # Winget adds this alias to future shells only. Check it explicitly so
        # deployment also works from a PowerShell session opened before install.
        $wingetZig = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Links\zig.exe"
        if (Test-Path -LiteralPath $wingetZig) {
            $zigPath = $wingetZig
        }
    }
    if (-not $zigPath) {
        Write-Error "CGO requires a C compiler on PATH (e.g. MinGW-w64 gcc or Zig). Install a windows/amd64 toolchain, then retry."
    }
    # Zig provides a self-contained Windows C compiler suitable for CGO.
    $zigDir = Split-Path -Parent $zigPath
    if (($env:PATH -split ';') -notcontains $zigDir) {
        $env:PATH = "$zigDir;$env:PATH"
    }
    $env:CC = 'zig cc'
}
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$env:CGO_ENABLED = "1"
$out = Join-Path $OutDir ($Name + ".dll")
Write-Host "Building $out (CGO_ENABLED=1, CC=$env:CC, buildmode=c-shared)"
go build -buildmode=c-shared -o $out .
Write-Host "OK $out"
