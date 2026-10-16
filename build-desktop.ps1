# Compatibility entry point: Desktop is the default Windows build in v0.9.0.
$ErrorActionPreference = 'Stop'
& "$PSScriptRoot/build.ps1"
if ($LASTEXITCODE -ne 0) { throw 'Desktop build failed' }
