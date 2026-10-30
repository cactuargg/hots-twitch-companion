# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в __TAG__

- Пять готовых CSS-тем счётчика OBS и библиотека своих тем: сохранение под именем, обновление и удаление, с предпросмотром.

- При автоматическом расчёте после матча или ручном выборе Win/Loss прогноз отменяется с возвратом баллов всем участникам, если на любом исходе 0 баллов.
- Если оба исхода имеют ставки, выполняется обычный расчёт. Проверка происходит при расчёте, а не при закрытии приёма ставок.
- Решение об отмене сохраняется до запроса в Twitch и переживает сетевые ошибки и перезапуск. При отсутствии достоверных сумм расчёт откладывается.

## What's new

- Five built-in OBS score themes and a persistent custom CSS theme library with preview, named saves, updates and deletion.

- Refund all participants instead of resolving when either outcome has zero Channel Points at settlement, for automatic and manual Win/Loss.
- Funded outcomes settle normally. Points are checked at settlement, not when betting closes.
- Refund decisions persist before the Twitch request and survive network failures and restarts. Missing or invalid totals defer settlement.

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
