# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в v0.10.1

- Исправлен герой в истории и винрейте: используется фактически сыгранный герой из replay.details вместо выбора в лобби, включая ARAM.
- Старые записи автоматически исправляются при запуске, если исходные реплеи доступны. Ручные результаты, исключения и сессии сохраняются.
- В составе матча рядом с каждым ником отображается герой; игроки сопоставляются по ToonHandle.
- Исправлены отступы в сессиях, Twitch и раскрытых настройках OBS. Превью оверлеев выровнены; изображение маски показывается целиком.

## What's new

- Use the actual played hero from replay.details for history and win rates, including ARAM; automatically repair existing entries when replay files remain available.
- Display each participant's played hero beside their nickname, matched by ToonHandle.
- Preserve manual results, exclusions and session assignments during repair and reimport.
- Consistent spacing in session, Twitch and expanded OBS panels; aligned overlay previews and uncropped mask images.

## OBS and data notes

Keep Companion running while using its OBS sources. The read-only server listens on 127.0.0.1:18765; another application on that port is reported as an error. Closing Companion closes the server; minimizing to tray keeps it running.

Automatic masking uses a fresh BattleLobby as a loading/match signal and restores the mask after the corresponding stable replay appears. It does not directly recognize the draft screen or cancellation, and transitions are delayed. Unknown state or connection loss shows the mask; manual Auto/Show/Hide controls remain available. Live draft/OBS behavior still needs validation on the user's setup; automated coverage uses real replay fixtures.

Matches without an observed start require manual session assignment. Historical imports do not change the current score. Unknown outcomes and excluded games do not enter win-rate calculations. Full profile reset also deletes session history and regenerates OBS links; HotS replay files remain untouched.

Existing Twitch settlement checks remain in place: exact lobby/replay correlation, durable settlement intents, manual Win/Loss/Cancel confirmations and recovery of uncertain requests.

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

The main EXE uses Microsoft WebView2 Evergreen Runtime. No Node.js or Go installation is needed to run the EXE. Twitch tokens stay local under Windows DPAPI. Close the previous Companion instance before replacing the portable executable.

Automated tests include real BattleLobby/StormReplay data, mocked Twitch transitions, local server startup/shutdown, guarded tray requests, live-launch consent and profile migration. Real Twitch payouts are not performed by these tests. The EXE is unsigned.

Unofficial fan-made software, not affiliated with Blizzard Entertainment or Twitch.
