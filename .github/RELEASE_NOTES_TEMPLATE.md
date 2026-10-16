# HotS Twitch Companion __TAG__

Portable Windows x64 app for Twitch Predictions based on Heroes of the Storm replays.

## What's new in v0.9.0

- Wails/WebView2 is now the default Windows desktop shell, starting at 520×740 with width constrained to 480–560 logical pixels.
- Custom draggable title bar with minimize, hide to tray, and close. The tray offers Open and Exit; hiding continues monitoring, while closing retains the existing safety confirmation and saved Prediction.
- H application icon, Windows version resources and PerMonitorV2 manifest; compact Home, History, Settings and setup wizard using the existing interface and Go engine.
- BrowserFallback remains a separate build and CI artifact. Desktop requires WebView2 Evergreen Runtime; no automatic installation occurs.
- Both configurations run Go tests/vet and all JavaScript tests before packaging. Dependencies' license notices are included in the ZIP.

## Match safety carried forward from v0.8.0

- Public Twitch Client ID can be embedded by the release workflow; connect using Device Code without creating your own app.
- BattleLobby parser with four real fixture builds: 89566, 89754, 90670 and 93640.
- Each Prediction keeps its own lobby and player binding. Automatic settlement requires matching embedded BattleLobby bytes, consistent participants/results, a fresh game clock and a unique replay. Seed or nickname alone is insufficient.
- Ambiguous matches stay blocked after restart. Manual Win/Loss/Cancel and Sync remain available.
- A Win/Loss/Cancel decision is saved before Twitch is called. Lost responses and restart resume that decision; a different outcome is blocked until Twitch is reconciled.
- Twitch status is read before PATCH, and terminal ID/status/winner are verified. External finalization is recorded without another mutation. Rate limits retain the decision and delay retries.
- The next fresh lobby is queued while the previous Prediction completes. The two-minute cooldown is removed; duplicate lobby contents do not create another Prediction.
- A lobby event is saved as consumed before POST. If creation cannot be confirmed, inspect Twitch: the app does not automatically recreate or adopt that Prediction by title.
- Local and CI builds run Go tests, Go vet, JavaScript syntax checks and UI tests. Local packages now include these release notes and SHA-256.

## Automatic support and recovery

Choose your account from a replay and save the ToonHandle **before** the match. Automatic outcomes currently support ten human participants in 5v5 Quick Match, Unranked Draft, Storm League and ARAM on the fixture builds above. AI/custom games, unsupported builds, missing evidence, legacy active Predictions without a binding and clock mismatches require manual decisions.

Start with simulation. When verification is incomplete, check the diagnosis on Home. If a selected outcome is awaiting confirmation, retry the same outcome or Sync with Twitch; restarting does not authorize a different decision. Live Win/Loss distributes viewer points, Cancel refunds them. Check the match before confirming.

## Download

| File | Description |
| --- | --- |
| `HotS-Twitch-Companion.exe` | Portable Windows x64 GUI executable |
| `HotS-Twitch-Companion-windows-x64.zip` | EXE, READMEs, MIT notices, release notes and checksum |
| `SHA256SUMS.txt` | SHA-256 for the EXE |

```text
__HASH__  HotS-Twitch-Companion.exe
```

The main EXE uses Microsoft WebView2 Evergreen Runtime. No Node.js or Go installation is needed to run the EXE. Twitch tokens stay local under Windows DPAPI. Close any old v0.8.0 instance before launching; old builds do not honor the new profile lock.

Automated tests include real BattleLobby/StormReplay data, mocked Twitch transitions, local server startup/shutdown, guarded tray requests and native window controls. Live Twitch payouts are not part of UI smoke-tests. Validate simulation and tray behavior on your Windows/DPI setup before a live channel. The EXE is unsigned.

Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch.
