# Build HotS Twitch Companion from source on Windows.
# Requires Go 1.25+ and Node.js 18+ on PATH for tests; app users do not need Node.
param([switch]$BrowserFallback)
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    . "$PSScriptRoot/build-version.ps1"
    if ($env:GITHUB_REF_TYPE -eq 'tag' -and $env:GITHUB_REF_NAME -ne $buildVersion) {
        throw "Release tag must match appVersion: $buildVersion"
    }
    Write-Host 'Running tests...'
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
    & go test -tags 'wails,production,wv2runtime.error' ./...
    if ($LASTEXITCODE -ne 0) { throw 'Desktop Go tests failed' }
    & go vet -tags 'wails,production,wv2runtime.error' ./...
    if ($LASTEXITCODE -ne 0) { throw 'Desktop Go vet failed' }
    Get-ChildItem 'internal/dashboard/web/assets' -Filter *.js | ForEach-Object {
        & node --check $_.FullName
        if ($LASTEXITCODE -ne 0) { throw "JavaScript syntax check failed: $($_.Name)" }
    }
    & node --test
    if ($LASTEXITCODE -ne 0) { throw 'UI tests failed' }

    $out = if ($BrowserFallback) { 'dist/browser-fallback' } else { 'dist' }
    New-Item -ItemType Directory -Force -Path $out | Out-Null
    Write-Host 'Building Windows GUI EXE...'
    $ldflags = '-H=windowsgui -s -w'
    $publicId = $env:TWITCH_CLIENT_ID
    if ($publicId) {
        if ($publicId -notmatch '^[A-Za-z0-9]+$' -or $publicId.Length -lt 10 -or $publicId.Length -gt 80) {
            throw 'Invalid TWITCH_CLIENT_ID environment variable'
        }
        $ldflags += " -X main.defaultTwitchClientID=$publicId"
    }
    $exeName = if ($BrowserFallback) { "HotS-Twitch-Companion-Browser-$buildVersion.exe" } else { "HotS-Twitch-Companion-$buildVersion.exe" }
    $zipName = if ($BrowserFallback) { "HotS-Twitch-Companion-Browser-$buildVersion.zip" } else { "HotS-Twitch-Companion-$buildVersion-windows-x64.zip" }
    $buildTags = if ($BrowserFallback) { @() } else { @('-tags', 'wails,production,wv2runtime.error') }
    & go build @buildTags -trimpath "-ldflags=$ldflags" -o "$out/$exeName" ./cmd/companion
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }

    Copy-Item 'README.md','README.ru.md','LICENSE','THIRD_PARTY_NOTICES.txt','docs/compact-desktop.ru.md','docs/profile-reset.ru.md','docs/live-predictions.ru.md','docs/sessions-obs.ru.md','docs/wide-ui.ru.md','docs/game-profiles.ru.md','docs/local-statistics.ru.md' $out -Force
    $notes = (Get-Content '.github/RELEASE_NOTES_TEMPLATE.md' -Raw) -replace '__TAG__', $buildVersion
    if ($BrowserFallback) {
        $notes = $notes.Replace('The main EXE uses Microsoft WebView2 Evergreen Runtime.', 'This Browser edition EXE uses Microsoft Edge app mode or the default browser; it has no native title bar or tray and does not require WebView2 Runtime.')
    }
    $hash = (Get-FileHash "$out/$exeName" -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $exeName" | Set-Content "$out/SHA256SUMS.txt" -Encoding utf8
    ($notes -replace '__HASH__', $hash -replace 'HotS-Twitch-Companion.exe', $exeName -replace 'HotS-Twitch-Companion-windows-x64.zip', $zipName) | Set-Content "$out/RELEASE_NOTES.md" -Encoding utf8
    Compress-Archive -Path "$out/$exeName", "$out/README.md", "$out/README.ru.md", "$out/LICENSE", "$out/THIRD_PARTY_NOTICES.txt", "$out/compact-desktop.ru.md", "$out/profile-reset.ru.md", "$out/live-predictions.ru.md", "$out/sessions-obs.ru.md", "$out/wide-ui.ru.md", "$out/game-profiles.ru.md", "$out/local-statistics.ru.md", "$out/SHA256SUMS.txt", "$out/RELEASE_NOTES.md" `
        -DestinationPath $zipName -Force

    Write-Host ''
    Write-Host 'Build successful:'
    Write-Host "  $out/$exeName"
    Write-Host "  $zipName"
    Write-Host "  SHA256: $hash"
} finally {
    Pop-Location
}
