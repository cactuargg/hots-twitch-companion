# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в __TAG__

- Игровые профили HotS: добавление аккаунтов, название, ник/Toon Handle и папка реплеев для каждого.
- Текущий игрок автоматически переносится в первый профиль, без сброса Twitch, OBS, тем и истории.
- Быстрое переключение аккаунта на главном экране. В настройках доступны редактирование и удаление профилей.
- Смена аккаунта заблокирована при мониторинге, незавершённом прогнозе или неподтверждённом создании. История и счёт привязаны к каждому игроку.

## What's new

- Named HotS game profiles with per-account player identity and replay folder, plus quick switching from Overview.
- Automatic migration of the current player into the first profile. Twitch authorization, prediction settings, OBS URLs, themes and history stay shared or preserved.
- Profile editing and deletion; history and score remain tied to each player identity.
- Switching is blocked during monitoring, active Predictions or unconfirmed creation. Unsaved drafts must be saved before quick switching.

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
