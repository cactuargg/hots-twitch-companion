# Optional Wails/WebView2 pilot. Does not replace or publish the stable package.
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    $tags = 'wails,production,wv2runtime.error'
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & go test -tags $tags ./...
    if ($LASTEXITCODE -ne 0) { throw 'Desktop Go tests failed' }
    & go vet -tags $tags ./...
    if ($LASTEXITCODE -ne 0) { throw 'Desktop Go vet failed' }
    Get-ChildItem web/assets -Filter *.js | ForEach-Object {
        & node --check $_.FullName
        if ($LASTEXITCODE -ne 0) { throw "JavaScript syntax check failed: $($_.Name)" }
    }
    & node --test
    if ($LASTEXITCODE -ne 0) { throw 'UI tests failed' }
    $ldflags = '-H=windowsgui -s -w'
    if ($env:TWITCH_CLIENT_ID) {
        if ($env:TWITCH_CLIENT_ID -notmatch '^[A-Za-z0-9]{10,80}$') { throw 'Invalid TWITCH_CLIENT_ID' }
        $ldflags += " -X main.defaultTwitchClientID=$env:TWITCH_CLIENT_ID"
    }
    $out = 'dist/desktop-pilot'
    New-Item -ItemType Directory -Force $out | Out-Null
    & go build -tags $tags -trimpath "-ldflags=$ldflags" -o "$out/HotS-Twitch-Companion-Desktop.exe" .
    if ($LASTEXITCODE -ne 0) { throw 'Desktop build failed' }
    Copy-Item LICENSE,THIRD_PARTY_NOTICES.txt,docs/desktop-shell.ru.md $out -Force
    $hash = (Get-FileHash "$out/HotS-Twitch-Companion-Desktop.exe" -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  HotS-Twitch-Companion-Desktop.exe" | Set-Content "$out/SHA256SUMS.txt" -Encoding utf8
    Compress-Archive -Path "$out/*" -DestinationPath HotS-Twitch-Companion-Desktop-Pilot-windows-x64.zip -Force
    Write-Host "Desktop Pilot built. SHA256: $hash"
} finally { Pop-Location }
