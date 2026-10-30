# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в __TAG__

- Отдельный OBS-виджет статистики текущего лобби: герой стримера, его локальный винрейт на герое и число игр.
- Знакомые участники: общий винрейт в этом режиме, статистика на герое и число встреч со стримером.
- Настройки → Оверлеи OBS → Виджет статистики лобби: включение, отдельный URL и живой предпросмотр. Рекомендуемый размер источника «Браузер»: 560 × 820.
- Автоматическое обновление и очистка данных при смене профиля, завершении/исчезновении лобби или отключении Companion. История не копируется целиком при опросе OBS.
- Для ARAM препик не выдаётся за фактического героя: до подтверждения показывается общий винрейт игрока в этом режиме. Статистика локальная, вероятность победы не рассчитывается.

## What's new

- Dedicated OBS lobby statistics widget with the streamer’s current hero, observed hero win rate and sample count.
- Known lobby participants with mode-specific overall/hero results and counts of matches observed alongside the streamer.
- Enable it under OBS overlays, copy its separate Browser Source URL (560 × 820), and use the live preview.
- Cached, lightweight polling clears old data on profile/lobby changes, completed matches and disconnects. No full history copies for widget polling.
- ARAM preselection is not treated as a confirmed hero; overall mode results are shown instead. No current-team inference or win-probability claim.

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
