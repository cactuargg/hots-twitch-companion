# HotS Twitch Companion __TAG__

Portable Windows x64 app for Twitch Predictions based on Heroes of the Storm replays.

## What's new in v0.9.5

- Recognizes HotS client build 98348 (October 2026 patch). The embedded BattleLobby of a real match on that build parses with the existing bit-level layout and is committed as fixture `IndustrialDistrict1_98348`, so matchmaking games on the current patch correlate and settle automatically again. Games on older builds are unaffected.
- Reminder: matches that are not ten-human 5v5 matchmaking lobbies (custom games, games with AI participants, observers) intentionally require manual Win/Loss/Cancel decisions.

## Browser edition from v0.9.4

- The Browser edition is now attached to the release itself as `HotS-Twitch-Companion-Browser-v__TAG__.zip` instead of staying a CI-only artifact, and its SHA-256 is included in `SHA256SUMS.txt`. Use it when WebView2 Runtime is unavailable: it runs the interface in Microsoft Edge app mode or the default browser, without the native title bar and tray.

## Update checks from v0.9.3

- The app checks public GitHub release metadata once at startup (and daily while running) and shows an "Update available" link in the footer when a newer version exists. The link opens the release page in your browser; nothing is downloaded or installed automatically.
- The check is a single read-only request to `api.github.com` that sends no account or match data, and it fails silently offline.

## Twitch identity fix from v0.9.2

- The verified Twitch connection now survives saving settings. Previously, finishing setup or pressing Launch right after authorization could report Twitch as disconnected and bounce back to the setup step, even though the token was stored. The identity is dropped only when the Client ID actually changes.

## Live launch from v0.9.1

- First-run setup now connects Twitch and launches real Predictions directly, with explicit confirmation. Simulation and the mode switch are removed from the user flow.
- Saving settings or finishing setup without starting never enables monitoring. Legacy simulated Predictions are cleared during profile migration; real active Predictions and recovery evidence are preserved.

## Desktop features from v0.9.0

- EXE/ZIP filenames include the app version, also shown in the footer. Desktop uses the title-bar close button instead of a duplicate footer Exit.
- Settings offers confirmed profile reset: after the app closes, next launch clears local settings/tokens/state/WebView2 cache and restarts setup. Monitoring must be stopped and no Prediction may be active. HotS replays stay untouched.

- Wails/WebView2 is now the default Windows desktop shell, starting at 520×740 with width constrained to 480–560 logical pixels.
- Custom draggable title bar with minimize, hide to tray, and close. The tray offers Open and Exit; hiding continues monitoring, while closing retains the existing safety confirmation and saved Prediction.
- H application icon, Windows version resources and PerMonitorV2 manifest; compact Home, History, Settings and setup wizard using the existing interface and Go engine.
- The Browser edition is published alongside the Desktop package. Desktop requires WebView2 Evergreen Runtime; no automatic installation occurs.
- Both configurations run Go tests/vet and all JavaScript tests before packaging. Dependencies' license notices are included in the ZIP.

## Match safety carried forward from v0.8.0

- Public Twitch Client ID can be embedded by the release workflow; connect using Device Code without creating your own app.
- BattleLobby parser with five real fixture builds: 89566, 89754, 90670, 93640 and 98348.
- Each Prediction keeps its own lobby and player binding. Automatic settlement requires matching embedded BattleLobby bytes, consistent participants/results, a fresh game clock and a unique replay. Seed or nickname alone is insufficient.
- Ambiguous matches stay blocked after restart. Manual Win/Loss/Cancel and Sync remain available.
- A Win/Loss/Cancel decision is saved before Twitch is called. Lost responses and restart resume that decision; a different outcome is blocked until Twitch is reconciled.
- Twitch status is read before PATCH, and terminal ID/status/winner are verified. External finalization is recorded without another mutation. Rate limits retain the decision and delay retries.
- The next fresh lobby is queued while the previous Prediction completes. The two-minute cooldown is removed; duplicate lobby contents do not create another Prediction.
- A lobby event is saved as consumed before POST. If creation cannot be confirmed, inspect Twitch: the app does not automatically recreate or adopt that Prediction by title.
- Local and CI builds run Go tests, Go vet, JavaScript syntax checks and UI tests. Local packages now include these release notes and SHA-256.

## Automatic support and recovery

Choose your account from a replay and save the ToonHandle **before** the match. Automatic outcomes currently support ten human participants in 5v5 Quick Match, Unranked Draft, Storm League and ARAM on the fixture builds above. AI/custom games, unsupported builds, missing evidence, legacy active Predictions without a binding and clock mismatches require manual decisions.

Connect Twitch and confirm the monitoring launch. When verification is incomplete, check the diagnosis on Home. If a selected outcome is awaiting confirmation, retry the same outcome or Sync with Twitch; restarting does not authorize a different decision. Win/Loss distributes viewer points, Cancel refunds them. Check the match before confirming.

## Download

| File | Description |
| --- | --- |
| `HotS-Twitch-Companion.exe` | Portable Windows x64 GUI executable |
| `HotS-Twitch-Companion-windows-x64.zip` | EXE, READMEs, MIT notices, release notes and checksum |
| `HotS-Twitch-Companion-Browser-v__TAG__.zip` | Browser edition for systems without WebView2 Runtime |
| `SHA256SUMS.txt` | SHA-256 for the EXE and the Browser edition ZIP |

```text
__HASH__  HotS-Twitch-Companion.exe
```

The main EXE uses Microsoft WebView2 Evergreen Runtime. No Node.js or Go installation is needed to run the EXE. Twitch tokens stay local under Windows DPAPI. Close any old v0.8.0 instance before launching; old builds do not honor the new profile lock.

Automated tests include real BattleLobby/StormReplay data, mocked Twitch transitions, local server startup/shutdown, guarded tray requests, live-launch consent and profile migration. Real Twitch payouts are not performed by these tests. The EXE is unsigned.

Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch.
