# Build (optional) and copy credit-manager.dll into CLIProxyAPI plugins dir.
param(
    [string]$DestDir = "D:\CLIProxyAPI\plugins\windows\amd64",
    [string]$OutDir = "dist",
    [string]$Name = "credit-manager",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$dllName = "$Name.dll"
$src = Join-Path $Root (Join-Path $OutDir $dllName)

if (-not $SkipBuild) {
    Write-Host "Building plugin..."
    & powershell -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "build.ps1") -OutDir $OutDir -Name $Name
    if ($LASTEXITCODE -ne 0) {
        throw "build.ps1 failed with exit code $LASTEXITCODE"
    }
}

if (-not (Test-Path -LiteralPath $src)) {
    throw "Source DLL not found: $src (run without -SkipBuild, or build first)"
}

New-Item -ItemType Directory -Force -Path $DestDir | Out-Null
$dest = Join-Path $DestDir $dllName

try {
    Copy-Item -LiteralPath $src -Destination $dest -Force
} catch {
    throw @"
Failed to copy to $dest
If CLIProxyAPI is running, stop it first (Windows often locks loaded DLLs), then retry.
$($_.Exception.Message)
"@
}

$info = Get-Item -LiteralPath $dest
Write-Host "OK copied -> $($info.FullName)"
Write-Host "Size: $($info.Length) bytes  Time: $($info.LastWriteTime)"
Write-Host "Restart CLIProxyAPI, then enable the plugin in 插件管理."