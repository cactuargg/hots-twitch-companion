# HotS Twitch Companion __TAG__

Portable Windows x64 app for automating Twitch Predictions based on Heroes of the Storm replays.

## What's new in v0.7.0

### Manual Prediction controls

- New **Win**, **Loss**, and **Cancel / refund** actions on Home while a Prediction is active.
- Live outcome actions require confirmation: Win/Loss distribute real viewer channel points, while Cancel returns points to viewers. Simulation never makes Twitch Prediction calls.
- These controls remain available when automated monitoring is stopped.
- **Sync with Twitch** reconciles a tracked Prediction that was already resolved/canceled elsewhere. It never silently discards a Prediction that Twitch still reports as ACTIVE or LOCKED.
- Safeguards check the exact Prediction ID and start time, and serialize manual actions with automatic replay processing to prevent duplicate or stale settlement.

### Faster automatic match resolution

- Removed the previous **75-second mandatory wait** before checking a finished replay.
- Once HotS writes a complete replay, the app polls every ~4 seconds and waits approximately 8 seconds for file stability before parsing its result.
- Twitch Predictions may be resolved before the full betting-window duration has elapsed when the final result is known.
- The app still gets match results **from completed replay files**, not game memory or screen capture, so resolution depends on HotS saving the replay.

### From v0.6.0

The minimalist navy/purple Windows interface continues to use Home, History, Settings, a guided first-run wizard and read-only replay diagnostics. No Electron or Node.js installation is required: the frontend is embedded in one Go executable.

## Download

| File | Description |
|------|-------------|
| `HotS-Twitch-Companion.exe` | Portable Windows x64 executable |
| `HotS-Twitch-Companion-windows-x64.zip` | EXE, README files, license and checksum |
| `SHA256SUMS.txt` | SHA-256 checksum for the executable |

## SHA-256

```text
__HASH__  HotS-Twitch-Companion.exe
```

## Usage

1. Download `HotS-Twitch-Companion.exe` and launch it (Microsoft Edge app-mode window, with fallback to the default browser).
2. Choose the correct Heroes of the Storm player from a replay, or enter your player name.
3. Connect Twitch using the Public app Client ID and Device Code flow, if using real Predictions.
4. Start in **simulation** and verify that a finished match is detected correctly.
5. On Home, use Win/Loss/Cancel if automatic settlement cannot be completed. **Double-check the result before confirming a live outcome**, since distributed viewer channel points cannot be recovered by undoing the click.

**Important:** The manual actions manage only a Prediction tracked by this instance of the app. If Twitch has already finished it, use Sync with Twitch. Windows GUI rendering and real Twitch payout behavior have not been smoke-tested manually in this release workflow; automated tests alone are insufficient for that assurance.

---

*Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch. Heroes of the Storm is a trademark of Blizzard Entertainment, Inc.*
