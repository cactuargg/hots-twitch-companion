# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в __TAG__

- Локальная статистика винрейта героев и игроков: число игр, победы, поражения, фильтр по режиму, поиск и разбор каждого игрока по героям.
- Общая база реплеев для всех игровых профилей. Импорт старых файлов учитывает всех участников, даже если выбранного игрока в матче нет.
- Новые стабильные реплеи обрабатываются автоматически, пока Companion работает. Счётчик сессии и мониторинг Twitch могут быть выключены.
- Данные записываются на диск при обработке реплеев, а не при выходе. Идентичные копии учитываются один раз; повторный импорт использует разобранные данные.
- Фоновая обработка ограничена четырьмя файлами за проход, записи группируются, таблицы кэшируются. Для старого архива есть кнопка импорта.

## What's new

- Local hero/player win rates with game counts, wins/losses, mode filters, search, pagination and per-player hero breakdowns.
- A shared replay database across game profiles. Historical import indexes all participants, including matches without the selected streamer.
- New stable replays are collected while Companion is running, even with the session counter and Twitch monitoring off.
- Results are saved to disk during replay processing, not at shutdown. Identical copies count once; parsed records are reused on reimport.
- Background work is limited to four files per pass, with batched writes and cached aggregates. Import older archives explicitly.

## Statistics notes

These are observed results in the available local replays, not global win rates, official ratings or predictions. Valid human player IDs and known wins/losses are required. Unknown results and AI entries are skipped. Two appearances of a hero in one match are two player results. Different client recordings of the same match are not guaranteed to deduplicate; identical bytes are.

Session edits and exclusions do not rewrite the separate raw replay statistics database. Game profile deletion preserves it; full application reset deletes it. Very large archives still need separate performance validation. The initial import is user initiated and does not change the current session score. Data stays local; no external statistics service is contacted.

## OBS and data notes

Keep Companion running while using its OBS sources. The read-only server listens on 127.0.0.1:18765; another application on that port is reported as an error. Closing Companion closes the server; minimizing to tray keeps it running.

Automatic masking uses a fresh BattleLobby as a loading/match signal and restores the mask after the corresponding stable replay appears. It does not directly recognize the draft screen or cancellation, and transitions are delayed. Unknown state or connection loss shows the mask; manual Auto/Show/Hide controls remain available. Live draft/OBS behavior still needs validation on the user's setup; automated coverage uses real replay fixtures.

Matches without an observed start require manual session assignment. Historical imports do not change the current score. Unknown outcomes and excluded games do not enter session win-rate calculations. The separate replay database retains original observations. Full profile reset also deletes session history and the local replay database and regenerates OBS links; HotS replay files remain untouched.

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
