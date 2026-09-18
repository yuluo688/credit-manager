# Build (optional), replace credit-manager.dll, and restart CLIProxyAPI.
param(
    [string]$DestDir = "D:\CLIProxyAPI\plugins\windows\amd64",
    [string]$OutDir = "dist",
    [string]$Name = "credit-manager",
    [switch]$SkipBuild,
    [string]$CPAExe
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

if ([string]::IsNullOrWhiteSpace($CPAExe)) {
    $cpaRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $DestDir))
    $CPAExe = Join-Path $cpaRoot "cli-proxy-api.exe"
}

if (-not (Test-Path -LiteralPath $CPAExe -PathType Leaf)) {
    throw "CLIProxyAPI executable not found: $CPAExe"
}

$CPAExe = (Resolve-Path -LiteralPath $CPAExe).Path
$cpaRoot = Split-Path -Parent $CPAExe
$cpaProcessName = [System.IO.Path]::GetFileNameWithoutExtension($CPAExe)
$runningCPA = @(Get-Process -Name $cpaProcessName -ErrorAction SilentlyContinue | Where-Object {
    -not $_.Path -or $_.Path -ieq $CPAExe
})

New-Item -ItemType Directory -Force -Path $DestDir | Out-Null
$dest = Join-Path $DestDir $dllName

if ($runningCPA.Count -gt 0) {
    Write-Host "Stopping CLIProxyAPI..."
    $runningCPA | Stop-Process -Force

    $deadline = (Get-Date).AddSeconds(15)
    do {
        Start-Sleep -Milliseconds 250
        $remainingCPA = @($runningCPA | Where-Object { -not $_.HasExited })
    } while ($remainingCPA.Count -gt 0 -and (Get-Date) -lt $deadline)

    if ($remainingCPA.Count -gt 0) {
        throw "CLIProxyAPI did not stop within 15 seconds."
    }
}

try {
    Copy-Item -LiteralPath $src -Destination $dest -Force
} catch {
    throw @"
Failed to copy to $dest
CLIProxyAPI was stopped before replacement, but Windows may still be holding the DLL.
$($_.Exception.Message)
"@
}

$info = Get-Item -LiteralPath $dest
Write-Host "OK copied -> $($info.FullName)"
Write-Host "Size: $($info.Length) bytes  Time: $($info.LastWriteTime)"

Write-Host "Starting CLIProxyAPI..."
$cpaProcess = Start-Process -FilePath $CPAExe -WorkingDirectory $cpaRoot -PassThru
Start-Sleep -Milliseconds 500
if ($cpaProcess.HasExited) {
    throw "CLIProxyAPI exited immediately after restart. Check its logs."
}

Write-Host "OK restarted CLIProxyAPI (PID $($cpaProcess.Id))."
Write-Host "Enable the plugin in 插件管理 if it is not already enabled."
