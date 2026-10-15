# HotS Twitch Companion __TAG__

A portable Windows x64 app that automatically runs Twitch Predictions based on Heroes of the Storm matches.

## What's new in v0.6

### Redesigned, minimalist desktop interface
- Replaced game-themed decorations, large banners, hexagons and glows with a clean dark navy and violet interface.
- Three clear sections: **Home**, **History** and **Settings**.
- Home now prioritizes one monitoring status and one Start/Stop action, with compact Twitch, Prediction and latest-match summaries.
- A smaller **1024×800** initial desktop window and responsive layouts for narrower windows and 125–150% Windows display scaling.
- Clearer Start vs. Stop styling, pending-action feedback, keyboard navigation, and reduced-motion support.

### Easier setup and settings
- Simplified first-run wizard: **choose your HotS player → optionally connect Twitch → start a safe simulation**.
- Select your player from an existing replay to fill nickname and Toon Handle automatically, or enter a nickname manually.
- Preview Prediction title, outcomes and betting-window duration without creating anything on Twitch.
- Settings now expose three everyday areas; technical fields stay available under **Advanced settings**.
- English/Russian switching, improved Client ID guidance, and a copy button for Twitch Device Code.
- The simulation remains the default: **real viewer channel points are never used without explicitly enabling live mode**.

### Match history and troubleshooting
- Search recent replays by map, filename or player; filter wins, losses and unknown results.
- Separate unreadable replays from successfully parsed replays where the selected player was not found.
- Structured event levels and problem-specific guidance for Twitch authorization, Prediction creation/settlement, and replay issues.
- Diagnostic logs remain in memory for the current run (up to 150 entries); they are not a permanent audit trail.

### Reliability and maintainability
- Split the frontend into focused JavaScript and CSS files while keeping everything embedded in a **single Go executable** — no Electron or Node.js needed by end users.
- Added automated Go/JavaScript UI tests and Windows builds in GitHub Actions.
- Improved button busy-state handling to prevent conflicting actions and restore controls on failure.

## Download

| File | Description |
|------|-------------|
| `HotS-Twitch-Companion.exe` | Portable Windows x64 application |
| `HotS-Twitch-Companion-windows-x64.zip` | EXE, documentation, license, and checksum |
| `SHA256SUMS.txt` | SHA-256 checksum of the executable |

The ZIP contains the EXE, `README.md`, `README.ru.md`, `LICENSE`, and `SHA256SUMS.txt`.

## SHA-256

```text
__HASH__  HotS-Twitch-Companion.exe
```

## Quick start

1. Download and run `HotS-Twitch-Companion.exe` (no installation; the UI opens in Microsoft Edge app mode, or in the default browser).
2. Choose your HotS player from a replay or enter the nickname.
3. Optionally connect Twitch using the **Public** app Client ID and Device Code authorization.
4. Start a **simulation** and confirm replay detection works before enabling live Predictions in Settings.

**Important:** Real Predictions can use viewers' Twitch channel points. Check the selected player and Twitch authorization before enabling live mode. Windows UI at high DPI and live Twitch interaction should also be checked manually; automated tests do not cover them.

---

*Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch. Heroes of the Storm is a trademark of Blizzard Entertainment, Inc.*
