# HotS Twitch Companion __TAG__

Portable Windows x64 companion for Heroes of the Storm and Twitch.

## Новое в v0.10.0

- Постоянные сессии побед/поражений с ручным сбросом, отменой сброса и историей. Перезапуск приложения не обнуляет счёт.
- Два отдельных источника «Браузер» для OBS: компактный счёт и маска препиков. Ссылки сохраняются между запусками; настройки цвета, размера и изображения доступны в приложении.
- Увеличенное окно 1100 × 800 со свободным изменением размера. Общий «Обзор» объединяет историю с пагинацией и винрейт по героям/картам справа. Статистика использует всю отфильтрованную выборку, независимо от страницы.
- Фоновый импорт реплеев пополняет общую статистику без изменения текущего счёта. Копии не считаются дважды; ручные исправления сохраняются.
- Настройки на одной странице без бокового меню: переключатели, группы кнопок, общая кнопка сохранения и открытые поля заголовка/победы/поражения с предпросмотром.

## What's new

- Durable win/loss sessions independent of Twitch, with archived sessions, undo, match reassignment and manual corrections.
- Separate read-only local OBS Browser Sources for score and draft mask, with stable URLs across restarts.
- Resizable 1100 × 800 Overview combining paginated matches and hero/map win rates. Filters and statistics cover all matching records, not just the visible page.
- Background replay import with duplicate protection, roster metadata and preservation of current session totals and existing corrections.
- Grouped settings on one page, switches and segmented controls, shared saving, and editable Prediction title/outcomes with a live preview.
- Go tests/vet for Desktop and Browser, 94 UI tests, Windows updater subprocess tests and race CI checks gate publication. Existing profiles and saved Twitch authorization remain compatible.

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
