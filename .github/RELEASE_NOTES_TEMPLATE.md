# HotS Twitch Companion __TAG__

Portable Windows x64 app for Twitch Predictions based on Heroes of the Storm replays.

## What's new in v0.8.0

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

The interface uses Microsoft Edge app mode, with default-browser fallback. No Node.js or Go installation is needed to run the EXE. Twitch tokens stay local under Windows DPAPI.

Automated tests include a real BattleLobby/StormReplay pair, mocked Twitch transitions and the real local HTTP server startup/shutdown. Visible Edge-window behavior and live Twitch payouts have not been manually smoke-tested. Complete that validation before treating the release as verified for a live channel. The EXE is unsigned.

Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch.
