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
$goarch = (& go env GOARCH).Trim()
if ($goarch -eq "amd64") {
    # Match the release workflow. GOAMD64 does not reach gcc; runtime/cgo needs
    # -march=x86-64 or a newer GCC can put BMI2 in x_cgo_sigaction.
    if (-not $env:GOAMD64) {
        $env:GOAMD64 = "v1"
    }
    foreach ($flag in @("CGO_CFLAGS", "CGO_CXXFLAGS")) {
        $current = [Environment]::GetEnvironmentVariable($flag, "Process")
        if ($current -notmatch '(^|\s)-march=') {
            if ([string]::IsNullOrWhiteSpace($current)) {
                Set-Item -Path "Env:$flag" -Value "-O2 -g -march=x86-64"
            } else {
                Set-Item -Path "Env:$flag" -Value "-O2 -g -march=x86-64 $current"
            }
        }
    }
}
$out = Join-Path $OutDir ($Name + ".dll")
Write-Host "Building $out (CGO_ENABLED=1, CC=$env:CC, GOAMD64=$env:GOAMD64, CGO_CFLAGS=$env:CGO_CFLAGS, buildmode=c-shared)"
go build -buildmode=c-shared -o $out .
Write-Host "OK $out"
