# HotS Twitch Companion __TAG__

Portable Windows x64 app that automatically runs Twitch Predictions based on your Heroes of the Storm match results.

## What's new

- **English and Russian interface** — switch any time via the `EN / RU` control in the top bar or in Settings; backend messages follow the selected language.
- **Configurable Prediction duration** — betting window of 1–30 minutes selected in Settings, like the duration picker in the Twitch Predictions interface (default: 2 minutes).
- English `README.md` added.
- CI: releases are now published correctly and include a direct `.exe` download.

## Downloads

| File | Description |
|------|-------------|
| `HotS-Twitch-Companion.exe` | Portable executable — just download and run |
| `HotS-Twitch-Companion-windows-x64.zip` | Full package (contents below) |
| `SHA256SUMS.txt` | Checksums |

## Archive contents

`HotS-Twitch-Companion-windows-x64.zip` contains:

```
HotS-Twitch-Companion.exe    the app
README.md                    documentation (English)
README.ru.md                 documentation (Russian)
LICENSE                      MIT license
SHA256SUMS.txt               checksums
```

## SHA256

```
__HASH__  HotS-Twitch-Companion.exe
```

## Quick start

1. Run `HotS-Twitch-Companion.exe` — no installation needed (the UI opens in Edge app mode or your default browser).
2. **Settings** → set your HotS player name, save.
3. **Matches → Check latest replay** — safe offline test without the Twitch API.
4. **Twitch** → enter your Public app Client ID and connect via OAuth (token is encrypted with DPAPI).
5. **Automation** → start monitoring. Simulation mode is on by default; enable real Predictions only after verifying the parser on your replays.

---

*Unofficial fan tool. Not affiliated with Blizzard Entertainment or Twitch. HotS is a trademark of Blizzard Entertainment, Inc.*
