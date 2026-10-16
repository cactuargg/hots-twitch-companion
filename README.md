# HotS Twitch Companion v0.9.2

v0.9.2 keeps the verified Twitch connection when settings are saved, so the launch step after authorization no longer bounces back to setup. v0.9.1 replaced simulation with a confirmed live Twitch launch; see [Desktop controls and build instructions](docs/compact-desktop.ru.md).

The v0.8.0 engine provides durable outcome recovery, exact Twitch status verification and queued
next-match lobbies without the old two-minute cooldown. Unknown POST results are
never automatically recreated for the same lobby. An unconfirmed Win/Loss/Cancel
is preserved across restart; conflicting decisions are blocked until reconciled.
See [stabilization notes and validation limits](docs/stabilization-v0.8.0.ru.md).

BattleLobby observations persist the latest roster in `state.json`; each new
Prediction gets its own frozen lobby and player binding. Automatic settlement
requires exact embedded BattleLobby content, consistent participant results,
a fresh in-game timestamp and a unique replay candidate. Only fixture-verified
builds 89566, 89754, 90670 and 93640 are supported. See [format research and fixture
provenance](testdata/battlelobby/README.md) and [correlation rules](docs/match-correlation.ru.md).

Portable Windows x64 EXE with a minimalist dark navy, blue and purple interface. The gaming visuals and decorative panels were removed. The GUI is built from local HTML/CSS/JS embedded into the executable with `go:embed`. The desktop window uses Microsoft WebView2 Evergreen Runtime. A separate browser fallback build uses Edge app mode or the default browser. **[Читать на русском](README.ru.md)**

The app watches local Heroes of the Storm files, and when a match starts it creates a **Twitch Prediction** ("HotS win?"). When the match ends, the replay is parsed, the tracked player's win/loss is determined, and the Prediction resolves automatically.

## Automatic and manual Prediction settlement

### If the replay shows a result but the Prediction remains unresolved

The match-history preview and automatic Prediction settlement are different operations. In v0.7.0 **Check latest replay was read-only**; it did not distribute viewer points. The automatic scanner ran only while monitoring, evaluated up to eight files, and excluded replays with timestamps earlier than the Prediction. Restarting monitoring could resume settlement.

History displays the latest twelve files. Settlement instead checks the complete eligible file inventory within safety limits; it never accepts the first matching player from that list. On starting monitoring, the scanner runs immediately and subsequently every four seconds. Home shows missing match evidence, incomplete verification or ambiguity. Automatic support requires ten human participants in a supported 5v5 matchmaking lobby and a ToonHandle selected before Prediction creation. Legacy Predictions without a binding, name-only identities, AI/custom lobbies and ambiguous matches require manual Win / Loss / Cancel. Restarting or pressing Check result again cannot bypass the correlation gate.

**View latest replay result** remains a read-only action. The new **Check result again** explicitly reruns replay-based settlement and *can finish the active Prediction* if a valid matching result exists. For real Twitch Predictions, you must first confirm that viewer points may be distributed. This action works even while monitoring is stopped; it verifies the active Prediction ID and creation time and never settles using replays from previous matches. If Twitch already finalized it elsewhere, use the separate Sync with Twitch action.



You no longer have to wait for the full Twitch betting window when a HotS game ends early. The app checks for completed replay files every four seconds and waits for the replay file to be stable for approximately eight seconds before parsing it. The previous mandatory 75-second delay is gone. The result still comes from a completed replay, **not by reading the game's screen or memory**, so settlement occurs after HotS has saved the replay.

While a Prediction is pending, Home shows **Win / Loss / Cancel**. These are available even with monitoring stopped. On **live Twitch Predictions**, Win or Loss immediately distributes viewer points, and Cancel refunds them. Every live action requires confirmation. Buttons disappear once a decision is successful.

If you already finished the Prediction through Twitch while the app still shows it as active, use **Sync with Twitch**. The app verifies the exact Prediction ID and clears the local state only if Twitch reports `RESOLVED` or `CANCELED`. It will not silently forget an `ACTIVE` or `LOCKED` Prediction. The API checks both the prediction ID and creation time before applying an action, preventing stale UI clicks from affecting the next match.

**Important:** manual actions only affect the Prediction currently tracked by this app, not arbitrary Predictions on the broadcaster's channel. Auto-settlement requires monitoring to remain running; manual controls stay available after stopping it.

## Key features

- **English and Russian interface** — switch language from Settings or the setup wizard. Backend log and error messages follow the selected language.
- **Configurable Prediction duration** — the betting window (1–30 minutes) is selected in Settings, just like the duration picker in the Twitch Predictions interface. Default: 2 minutes.
- **Customizable Prediction texts** — title and outcome names can be changed in Settings; empty fields use localized defaults.
- **Replays folder check** — Settings has a button that verifies the folder (default or custom) and counts found `.StormReplay` files before saving.
- **Explicit launch** — connect Twitch and confirm real Predictions. Saving configuration never starts monitoring.
- Read-only MPQ replay parser (zlib/BZip2) written from public file-format documentation — no DLL injection, hooks, drivers, or game-memory access.

## Compact desktop and Windows display scaling

Desktop starts at **520×740**, resizes between 480 and 560 logical pixels in width, and supports Windows DPI scaling. The custom title bar provides minimize, hide to tray, and close. Hiding keeps monitoring active; closing keeps the existing confirmation and saves any active Prediction. Click the tray icon or Open to restore; Exit uses the same close confirmation.

[Desktop details, prerequisites and validation](docs/compact-desktop.ru.md). BrowserFallback keeps the previous responsive interface.

## Twitch login without entering a Client ID

Official Windows builds can embed one **public** OAuth Client ID for HotS Twitch Companion. Set the repository's **Settings → Secrets and variables → Actions → Variables → TWITCH_CLIENT_ID**. The Windows workflow injects it via `-ldflags -X main.defaultTwitchClientID=...`; no OAuth Client Secret or user tokens are bundled into the EXE.

Register a **Public** Twitch app for the Device Code Flow. Each user still authorizes their own channel on Twitch, and their tokens are stored locally using Windows DPAPI. For normal users the flow is just Connect Twitch, then approve access using the code on Twitch Activate.

The build variable is optional for local or fork builds. If unset, users can enter their own Twitch Client ID in **Advanced settings**. If a built-in ID exists, leaving the field empty automatically uses it; a custom ID takes precedence. Changing apps requires Twitch reauthorization because tokens are bound to the Client ID.

The Client ID is **not secret** and is extractable from any distributed EXE. Never commit or embed a Client Secret, access token or refresh token. Public Device Code refresh tokens are single-use and may require reauthorization after prolonged inactivity.

## Simplified first-run setup

Choose your HotS player, connect Twitch, then confirm **Start Twitch Predictions**. Twitch is required; there is no simulation or mode switch. **Finish without starting** saves setup without monitoring. Failed or cancelled launch leaves the wizard open.

[Live launch and profile migration](docs/live-predictions.ru.md).

## Simplified settings

Three cards: Twitch connection, HotS player and Prediction duration/preview. Technical fields remain in Advanced settings. Saving settings does not start monitoring; starting real Predictions requires explicit confirmation. Manual outcomes keep their own confirmations.

## Minimal production interface

The live Home screen now follows the approved minimalist prototype: **one status, one Start/Stop action, Twitch, Prediction and last match**. The full event log retains its filters inside expandable Diagnostics in History. Navigation has three tabs: Home, History and Settings. Decorative game frames, glowing cards and banners were removed. Live mode still warns about real viewer channel points.

## Dashboard status model

Overview now puts **Start/Stop monitoring** and **Check latest replay** above the secondary cards. Its monitoring, Twitch connection, and Prediction statuses are separate. A stored OAuth token is shown as *Needs verification* until Twitch validates it. The estimated betting-window countdown is derived from local timestamps and configuration; it **does not confirm** the current remote Twitch Prediction status.

## Easier setup and preview

The setup wizard and Settings now offer **Choose player from a replay**: pick your identity from the most recent readable replay and the player name plus Toon Handle are filled in together. Choose **Save settings** (or advance through the wizard) to apply your selection. If you have no recorded games, enter the player name manually.

**What viewers will see** previews the exact draft Prediction title, outcome texts and selected betting duration. The preview never makes a Twitch API call or creates a Prediction. It also appears on the wizard's final step.

Twitch still requires a **Public** application Client ID that you provide yourself; no Client Secret is needed. Setup now shows step-by-step instructions, connection status, and a button to copy the Device Code. An embedded shared Client ID is intentionally not bundled.

CI also checks the frontend JavaScript syntax and runs dependency-free Node.js onboarding tests; end users do not need Node.js installed.

## Match history and diagnostics

**Match History** searches across the most recent 12 saved replays by map, filename or player, and filters by Win / Loss / Unknown. An unreadable replay is different from a readable replay where the configured player cannot be identified. In the latter case the expanded card shows an explanation and a link to player Settings.

**Event Log** now categorizes failures as Info, Warning or Error using structured backend event metadata, not translated text. Filter by severity and search within recent events; known errors offer actionable hints (for example Twitch authorization, Prediction creation/settlement, missing replays). The preview on Overview still shows the seven most recent events.

The event log remains **in memory only**, with a limit of 150 entries; it is not a persistent audit log and it does not guarantee Twitch's live remote Prediction state. Replay history displays the most recent 12 parsed replays, not a complete match archive.

## Frontend architecture and accessibility

The Windows application still runs from **one Go executable**. HTML/CSS/JS assets are compiled into it via `go:embed`; users do not need Node.js or a separate local web server.

The old monolithic frontend is now organized into standalone files, loaded in order without bundlers or external dependencies:

- `core.js` — local API, shared state, navigation, helpers
- `i18n.js` — Russian/English labels and locale switching
- `replays.js` — match history and replay cards
- `diagnostics.js` — event log and troubleshooting
- `onboarding.js` — setup wizard, player picker and Prediction preview
- `app.js` — status rendering, actions and application startup

CSS follows the same pattern: `style.css` foundation, then `wizard.css`, `overview.css`, `onboarding.css`, and `diagnostics.css`. Their order in `web/index.html` is intentional. The existing Go embed directive includes every file in `web/assets/`.

Tabs support keyboard Left/Right, Home/End, track focus, and expose their selected state to assistive technologies.

For contributors, run `go test ./...`, `node --test`, and `node --check web/assets/app.js`. GitHub Actions performs syntax checks on **all** JS files and runs all tests before building the Windows executable. Node.js is only needed for development and CI.

## Getting started

A **setup wizard** runs automatically on first launch: it walks you through language, your HotS player name (with a safe replay test), Twitch connection and automation, explaining what to expect at every step. You can re-run it any time from **Settings**.

1. Stop any previous `watch` process and old `.exe`.
2. Run `HotS-Twitch-Companion-v0.9.2.exe` from the release ZIP. Node.js, Go and Python are not required.
3. Follow the wizard — or configure manually: in **Settings** set your HotS player name and save; **Match History → Check latest replay** is a safe test without the Twitch API.
4. In **Settings → Twitch** enter the `Client ID` of a **Public** application, save, and connect via the OAuth Device Code Flow (the token is encrypted with DPAPI).
5. Optionally set the **Prediction duration** in **Settings → Operation mode**.
6. Start monitoring from **Overview** (or the wizard). Confirm real Twitch Predictions.
7. For real points, separately enable real mode with confirmation in **Settings**.
8. The **Exit** button at the bottom stops the process immediately. Closing the window also stops the app after a few minutes of lost contact.

**Match History** shows your recent games; click any match to expand full game data — map, date and every player with their result, your entry highlighted.

## Built-in safety

The local interface listens only on `127.0.0.1` at a random port, the address is guarded by a random 192-bit path segment, API calls require a special header, POST requests are checked for Origin/Sec-Fetch-Site, CSP is enabled, and cross-origin requests are denied. Twitch tokens never reach the UI and are stored via Windows DPAPI; HotS files are opened read-only, without DLL injection, hooks, drivers, or game-memory reads. None of these mechanisms guarantees approval by Blizzard.

The logic from the older Go version (MPQ zlib/BZip2 handling, Twitch OAuth, Predictions, `replay.server.battlelobby`/`.StormReplay` watching) is preserved. State in `%LOCALAPPDATA%\HotSTwitchCompanion` is reused — **do not run two versions at the same time**. The old Node.js prototype is not compatible in state/token format.

## Building from source

```powershell
go test ./...
go build -trimpath -ldflags="-H=windowsgui -s -w" -o HotS-Twitch-Companion-v0.9.2.exe .
```

Linux cross-compile: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H=windowsgui -s -w" -o HotS-Twitch-Companion-v0.9.2.exe .`

Go 1.25+, Node.js 18+ for checks. Run `./build.ps1` for Desktop or `./build.ps1 -BrowserFallback` for the browser package. Desktop uses Wails v2/WebView2 and fyne.io/systray; see [Desktop build instructions](docs/compact-desktop.ru.md). `service.go` — HTTP/UI wiring, `main_windows.go` — Edge window, `core.go` — Twitch/HotS logic, `replay.go` — MPQ decoder, `i18n.go` — EN/RU message catalog, `web/` — the interface.

## Tests

MPQ, BZip2/zlib, result reading, file integrity, and local API tests pass on Linux; the `.exe` is cross-compiled and verified as a Windows x64 GUI PE. **The live Twitch API and on-screen behavior with real HotS have not been tested yet.** Replay parsing and launch consent are covered by automated tests; live payouts are not performed by tests.

## GitHub and CI/CD

Workflow `.github/workflows/windows.yml`:

- on push to `main` / Pull Request / manual run: runs `go test ./...`, builds the Windows x64 `.exe`, and uploads a ZIP to **Actions → Artifacts**;
- on pushing a `v*` tag: additionally creates a **GitHub Release** with the ZIP + SHA256.

Local build in PowerShell:

```powershell
.\build.ps1
```

Result: `dist/HotS-Twitch-Companion-v0.9.2.exe` and `HotS-Twitch-Companion-v0.9.2-windows-x64.zip`.

First release (in Git Bash or another terminal with Git):

```bash
git add .
git commit -m "Publish HotS Twitch Companion v0.3"
git push origin main
git tag v0.3.0
git push origin v0.3.0
```

Before publishing, check `git status` and the list of tracked files: **do not commit Client Secrets, access/refresh tokens, or personal HotS files**. User configuration and DPAPI tokens live in `%LOCALAPPDATA%\HotSTwitchCompanion`, not in the repository. Only publish artwork/UI files you have the rights to distribute. HotS is a Blizzard trademark; add a disclaimer that the project is unofficial.

## License

The project source code is distributed under the **MIT License** — see [LICENSE](LICENSE). This permits use, modification, and redistribution of the source code, including commercial use, provided the license text and copyright notice are retained. `LICENSE` names `HotS Twitch Companion contributors` as the collective copyright holder; replace it with your name or organization before publishing if you prefer.

The MIT License **does not grant rights to trademarks, logos, or third-party artwork** of Blizzard and Twitch. Verify the origin of files in `web/assets/` and your rights to publish them. This application is unofficial and is not affiliated with Blizzard Entertainment or Twitch.

GitHub Actions and the Windows GUI require verification on a real Windows run. The `.exe` is not Authenticode-signed; SHA256 pins the content but does not prove the absence of malicious code.

Profile reset: Settings → User data → Clear user data; restart the app to begin setup again. See [reset behavior](docs/profile-reset.ru.md).
