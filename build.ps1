# Build HotS Twitch Companion v0.4.1 from source on Windows.
# Requires Go 1.23+ on PATH. No third-party dependencies.
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    Write-Host 'Running tests...'
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }

    New-Item -ItemType Directory -Force -Path 'dist' | Out-Null
    Write-Host 'Building Windows GUI EXE...'
    & go build -trimpath '-ldflags=-H=windowsgui -s -w' -o 'dist/HotS-Twitch-Companion.exe' .
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }

    Copy-Item 'README.md' 'dist/README.md' -Force
    Copy-Item 'README.ru.md' 'dist/README.ru.md' -Force
    Copy-Item 'LICENSE' 'dist/LICENSE' -Force
    $hash = (Get-FileHash 'dist/HotS-Twitch-Companion.exe' -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  HotS-Twitch-Companion.exe" | Set-Content 'dist/SHA256SUMS.txt' -Encoding utf8
    Compress-Archive -Path 'dist/HotS-Twitch-Companion.exe', 'dist/README.md', 'dist/README.ru.md', 'dist/LICENSE', 'dist/SHA256SUMS.txt' `
        -DestinationPath 'HotS-Twitch-Companion-windows-x64.zip' -Force

    Write-Host ''
    Write-Host 'Build successful:'
    Write-Host '  dist/HotS-Twitch-Companion.exe'
    Write-Host '  HotS-Twitch-Companion-windows-x64.zip'
    Write-Host "  SHA256: $hash"
} finally {
    Pop-Location
}
