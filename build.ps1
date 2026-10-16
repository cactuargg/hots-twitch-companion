# Build HotS Twitch Companion v0.8.0 from source on Windows.
# Requires Go 1.25+ and Node.js 18+ on PATH for tests; app users do not need Node.
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    Write-Host 'Running tests...'
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
    Get-ChildItem 'web/assets' -Filter *.js | ForEach-Object {
        & node --check $_.FullName
        if ($LASTEXITCODE -ne 0) { throw "JavaScript syntax check failed: $($_.Name)" }
    }
    & node --test
    if ($LASTEXITCODE -ne 0) { throw 'UI tests failed' }

    New-Item -ItemType Directory -Force -Path 'dist' | Out-Null
    Write-Host 'Building Windows GUI EXE...'
    $ldflags = '-H=windowsgui -s -w'
    $publicId = $env:TWITCH_CLIENT_ID
    if ($publicId) {
        if ($publicId -notmatch '^[A-Za-z0-9]+$' -or $publicId.Length -lt 10 -or $publicId.Length -gt 80) {
            throw 'Invalid TWITCH_CLIENT_ID environment variable'
        }
        $ldflags += " -X main.defaultTwitchClientID=$publicId"
    }
    & go build -trimpath "-ldflags=$ldflags" -o 'dist/HotS-Twitch-Companion.exe' .
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }

    Copy-Item 'README.md' 'dist/README.md' -Force
    Copy-Item 'README.ru.md' 'dist/README.ru.md' -Force
    Copy-Item 'LICENSE' 'dist/LICENSE' -Force
    $notes = (Get-Content '.github/RELEASE_NOTES_TEMPLATE.md' -Raw) -replace '__TAG__', 'v0.8.0'
    $hash = (Get-FileHash 'dist/HotS-Twitch-Companion.exe' -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  HotS-Twitch-Companion.exe" | Set-Content 'dist/SHA256SUMS.txt' -Encoding utf8
    ($notes -replace '__HASH__', $hash) | Set-Content 'dist/RELEASE_NOTES.md' -Encoding utf8
    Compress-Archive -Path 'dist/HotS-Twitch-Companion.exe', 'dist/README.md', 'dist/README.ru.md', 'dist/LICENSE', 'dist/SHA256SUMS.txt', 'dist/RELEASE_NOTES.md' `
        -DestinationPath 'HotS-Twitch-Companion-windows-x64.zip' -Force

    Write-Host ''
    Write-Host 'Build successful:'
    Write-Host '  dist/HotS-Twitch-Companion.exe'
    Write-Host '  HotS-Twitch-Companion-windows-x64.zip'
    Write-Host "  SHA256: $hash"
} finally {
    Pop-Location
}
