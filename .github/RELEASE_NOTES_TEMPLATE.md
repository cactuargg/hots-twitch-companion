# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в __TAG__

- Подробности текущего матча перенесены на главный экран: герой, локальный винрейт, число игр и знакомые участники с отметкой союзников и соперников.
- Экспериментальная оценка победы по игрокам и героям добавлена в существующий OBS-счётчик W/L. Включается отдельно в настройках OBS, новый источник не нужен.
- По умолчанию процент виден первые 15 секунд, затем 10 секунд каждые 2 минуты от начала лобби. Все три интервала настраиваются; перезагрузка OBS не перезапускает таймер.
- При нехватке истории или недостоверном лобби процент скрыт. После обработки соответствующего итогового реплея показ прекращается до следующего матча; при потере связи оценка исчезает не позднее чем через 5 секунд.
- Команды читаются из явного атрибута BattleLobby. В ARAM неподтверждённые препики не участвуют в оценке. Старый отдельный источник лобби сохранён для совместимости.

## What's new

- Current-match details now appear on the home screen, including hero results, sample counts and known allies/opponents.
- An optional experimental win estimate appears inside the existing OBS W/L counter; no additional source is required.
- Default schedule: 15 seconds at lobby start, then 10 seconds every 2 minutes. All intervals are configurable and OBS reloads do not restart the clock.
- Insufficient history, stale lobbies and completed matches hide the percentage. Completion follows the matching replay; disconnects clear the estimate within 5 seconds.
- Explicit BattleLobby teams replace slot-order assumptions. ARAM preselection is excluded from hero estimates. The legacy standalone lobby source remains compatible.

## Experimental estimate

This is an uncalibrated local-history heuristic, not an official rating or a validated probability model. It does not identify streamsnipers. Local streamer recordings are a biased sample. Only earlier, consistent human 5v5 replays in the same mode contribute. At least 20 matches and coverage for 6 of 10 participants (at least 2 per side) are required. Small samples shrink toward 50%; manually chosen player/hero coefficients produce a conservative 20–80% display. See the included lobby-widget.ru.md for the formula and limitations. No external statistics service is contacted.

## Statistics notes

These are observed results in the available local replays, not global win rates or official ratings. The separate experimental estimate has the limitations described above. Valid human player IDs and known wins/losses are required. Unknown results and AI entries are skipped. Two appearances of a hero in one match are two player results. Different client recordings of the same match are not guaranteed to deduplicate; identical bytes are.

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
