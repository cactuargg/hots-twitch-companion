# HotS Twitch Companion __TAG__

Portable Windows x64 app for Twitch Predictions based on Heroes of the Storm replays.

## What's new in v0.9.9

- Clearer Russian and English setup, settings and recovery messages. The interface explains saved Twitch authorization, the player ID required for automatic results and the difference between a betting window and match duration.
- Manual result confirmations identify the selected HotS player and use readable paragraphs. Replay preview and Twitch-result checks explain what they do without implying they resolve a Prediction.
- Stopping monitoring is distinguished from cancelling a Twitch Prediction, and losing the connection to the window is distinguished from stopping background monitoring. Startup diagnostics no longer incorrectly promise simulation mode.
- Failed creation checks are visible on Home, while uncertain remote creation retains its separate warning and recovery action. Saved Predictions show recovery guidance when monitoring is stopped.
- Localized visible labels, accessibility names and setup result badges. Added checks for translation coverage and placeholders; all 84 UI tests, Go tests/vet, Desktop, Browser and race checks gate publication.
- Existing profiles, saved Twitch authorization and prediction recovery rules remain compatible.

## Reliability fixes from v0.9.8

- Prediction titles, outcomes and betting-window settings can be changed while monitoring runs. An active prediction retains its original betting-window duration; new settings apply to future predictions.
- Edits made during an in-flight settings save remain marked as unsaved. Start is blocked until the latest draft is saved; overlapping settings and language writes are guarded.
- Twitch connection checks recover automatically after temporary connectivity failures. Connection loss is visible in the interface, and stale status cannot authorize prediction actions.
- Refreshed Twitch credentials survive temporary disk-save errors in memory. The app retries saving before allowing API operations and explains storage failures in English and Russian. Already saved authorization is preserved between launches.
- A slow token refresh no longer blocks connection-status reads. Successful exit or profile reset stops status/replay polling, cancels outstanding requests and ignores late responses.
- Added regression coverage for credential persistence, concurrent refresh/status access, settings drafts, Go-to-JavaScript status compatibility and interface shutdown. Desktop, Browser and race checks gate publication.

## Architecture from v0.9.7

- Completes the architecture refactor: Engine, Twitch, Dashboard, configuration, storage and HotS correlation have separate package boundaries. The Windows command now contains launchers and dependency composition.
- Dashboard reads independent state snapshots and invokes application operations. HTTP transport, time, persistence and game-source dependencies can be isolated in tests; Twitch owns synchronized token refresh and credential commits.
- Existing profiles, DPAPI tokens, API routes and prediction recovery rules remain compatible. Creation uncertainty and saved settlement intent continue to prevent duplicate wagers or conflicting payouts.
- Preserves all 94 legacy test/fuzz entry points, adds boundary and token-rotation checks, and documents the architecture. Desktop, Browser and race checks must all pass before release publication.

## Custom games from v0.9.6

- Custom 5v5 games with AI participants now settle automatically. Every human lobby slot is still verified by a unique ToonHandle; AI entries carry no verifiable identity, so only their team and outcome are checked (verified on the real `IndustrialDistrict1_98348` match). An unaccounted Hero handle, inconsistent team outcomes or a binding outside the roster still block automatic settlement.
- Matchmaking games keep the strict ten-human 5v5 verification.

## Client build 98348 from v0.9.5

- Recognizes HotS client build 98348 (October 2026 patch). The embedded BattleLobby of a real match on that build parses with the existing bit-level layout and is committed as fixture `IndustrialDistrict1_98348`, so matchmaking games on the current patch correlate and settle automatically again. Games on older builds are unaffected.

## Browser edition from v0.9.4

- The Browser edition is attached to the release itself as `HotS-Twitch-Companion-Browser-__TAG__.zip`, and its SHA-256 is included in `SHA256SUMS.txt`. Use it when WebView2 Runtime is unavailable: it runs the interface in Microsoft Edge app mode or the default browser, without the native title bar and tray.

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

Choose your account from a replay and save the ToonHandle **before** the match. Automatic outcomes support 5v5 lobbies on the fixture builds above: matchmaking games with ten human participants, and custom games where AI participants are accepted with verified team/outcome data but no identity. Name-only identities, missing evidence, legacy active Predictions without a binding and clock mismatches require manual decisions.

Connect Twitch and confirm the monitoring launch. When verification is incomplete, check the diagnosis on Home. If a selected outcome is awaiting confirmation, retry the same outcome or Sync with Twitch; restarting does not authorize a different decision. Win/Loss distributes viewer points, Cancel refunds them. Check the match before confirming.

## Download

| File | Description |
| --- | --- |
| `HotS-Twitch-Companion.exe` | Portable Windows x64 GUI executable |
| `HotS-Twitch-Companion-windows-x64.zip` | EXE, READMEs, MIT notices, release notes and checksum |
| `HotS-Twitch-Companion-Browser-__TAG__.zip` | Browser edition for systems without WebView2 Runtime |
| `SHA256SUMS.txt` | SHA-256 for the EXE and the Browser edition ZIP |

```text
__HASH__  HotS-Twitch-Companion.exe
```

The main EXE uses Microsoft WebView2 Evergreen Runtime. No Node.js or Go installation is needed to run the EXE. Twitch tokens stay local under Windows DPAPI. Close any old v0.8.0 instance before launching; old builds do not honor the new profile lock.

Automated tests include real BattleLobby/StormReplay data, mocked Twitch transitions, local server startup/shutdown, guarded tray requests, live-launch consent and profile migration. Real Twitch payouts are not performed by these tests. The EXE is unsigned.

Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch.
