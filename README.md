# HotS Twitch Companion v0.7.0

Portable Windows x64 EXE with a minimalist dark navy, blue and purple interface. The gaming visuals and decorative panels were removed. The GUI is built from local HTML/CSS/JS embedded into the executable with `go:embed`. A separate desktop window is opened through the installed Microsoft Edge in `--app` mode; without Edge, your default browser is used. **[Читать на русском](README.ru.md)**

The app watches local Heroes of the Storm files, and when a match starts it creates a **Twitch Prediction** ("HotS win?"). When the match ends, the replay is parsed, the tracked player's win/loss is determined, and the Prediction resolves automatically.

## Automatic and manual Prediction settlement

### If the replay shows a result but the Prediction remains unresolved

The match-history preview and automatic Prediction settlement are different operations. In v0.7.0 **Check latest replay was read-only**; it did not distribute viewer points. The automatic scanner ran only while monitoring, evaluated up to eight files, and excluded replays with timestamps earlier than the Prediction. Restarting monitoring could resume settlement.

The scanner and History now consider the same **12 latest replay files**. On starting monitoring, the scanner runs immediately and subsequently every four seconds. A pending Prediction displays a specific diagnosis on Home: no replay, file still being written, replay predating the Prediction, player identity mismatch, Twitch settlement error, or stalled scanning.

**View latest replay result** remains a read-only action. The new **Check result again** explicitly reruns replay-based settlement and *can finish the active Prediction* if a valid matching result exists. For real Twitch Predictions, you must first confirm that viewer points may be distributed. This action works even while monitoring is stopped; it verifies the active Prediction ID and creation time and never settles using replays from previous matches. If Twitch already finalized it elsewhere, use the separate Sync with Twitch action.



You no longer have to wait for the full Twitch betting window when a HotS game ends early. The app checks for completed replay files every four seconds and waits for the replay file to be stable for approximately eight seconds before parsing it. The previous mandatory 75-second delay is gone. The result still comes from a completed replay, **not by reading the game's screen or memory**, so settlement occurs after HotS has saved the replay.

While a Prediction is pending, Home shows **Win / Loss / Cancel**. These are available even with monitoring stopped. On **live Twitch Predictions**, Win or Loss immediately distributes viewer points, and Cancel refunds them. Every live action requires confirmation. Simulation does not call Twitch. Buttons disappear once a decision is successful.

If you already finished the Prediction through Twitch while the app still shows it as active, use **Sync with Twitch**. The app verifies the exact Prediction ID and clears the local state only if Twitch reports `RESOLVED` or `CANCELED`. It will not silently forget an `ACTIVE` or `LOCKED` Prediction. The API checks both the prediction ID and creation time before applying an action, preventing stale UI clicks from affecting the next match.

**Important:** manual actions only affect the Prediction currently tracked by this app, not arbitrary Predictions on the broadcaster's channel. Auto-settlement requires monitoring to remain running; manual controls stay available after stopping it.

## Key features

- **English and Russian interface** — switch language from Settings or the setup wizard. Backend log and error messages follow the selected language.
- **Configurable Prediction duration** — the betting window (1–30 minutes) is selected in Settings, just like the duration picker in the Twitch Predictions interface. Default: 2 minutes.
- **Customizable Prediction texts** — title and outcome names can be changed in Settings; empty fields use localized defaults.
- **Replays folder check** — Settings has a button that verifies the folder (default or custom) and counts found `.StormReplay` files before saving.
- **Simulation mode by default** — full pipeline runs without touching real channel points until you explicitly enable real Predictions.
- Read-only MPQ replay parser (zlib/BZip2) written from public file-format documentation — no DLL injection, hooks, drivers, or game-memory access.

## Compact desktop and Windows display scaling

The Edge app now starts at **1024×800** instead of 1440×950 (the window remains resizable). Spacing and card dimensions are reduced without shrinking text everywhere, and content reflows instead of forcing horizontal scrolling in narrow windows.

Responsive CSS addresses effective viewport widths caused by **125–150% Windows display scaling** and browser zoom. Stop Monitoring is visually distinct from Start. While an app request runs, conflicting action buttons are disabled and the activated button shows a progress indicator; controls recover after success or failure. The OS reduced-motion preference is honored.

Automated checks cover CSS breakpoints and interactive busy states, but cannot replace manual visual review on a high-DPI Windows desktop.

## Simplified first-run setup

The first-run wizard now has **three focused steps**: (1) choose a HotS player from a replay or enter the nickname manually, (2) optionally connect Twitch, and (3) **start a safe simulation**. Twitch authorization is **not required to simulate**. Likewise, replay validation is optional until a finished match is available.

Twitch Client ID and manual replay validation live in expandable details, rather than lengthy instructions. The final step has one primary **Start simulation** action, with **Finish without starting** as a secondary choice. The app persists `realMode=false` before asking the engine to start; it will not silently switch an already-running live monitor. If starting fails, the wizard remains open, without marking the setup complete.

Existing Go API endpoints, Twitch OAuth, and configuration file format are unchanged. A compact EN/RU switch remains available in the wizard header and in Settings.

## Simplified settings

Settings now has three straightforward cards: Twitch (connection and status), Heroes of the Storm (player name and optional replay-based selection), and Prediction (betting window, preview and an **explicit live viewer-points toggle**).

The technical controls (Twitch Client ID, Toon Handle, replay path and customized Prediction copy) remain fully supported but are tucked into **Advanced settings**. The configuration format and behavior are unchanged. If a user clicks Connect without a Client ID, the advanced section opens and focuses the correct field.

A persistent Save bar indicates draft changes. First enabling real Predictions still requires explicit confirmation, and unchecking an enabled live mode warns that live points remain enabled until the change is saved. Language selection is visible at the top and persists independently.

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
2. Run `HotS-Twitch-Companion.exe` from the release ZIP. Node.js, Go and Python are not required.
3. Follow the wizard — or configure manually: in **Settings** set your HotS player name and save; **Match History → Check latest replay** is a safe test without the Twitch API.
4. In **Settings → Twitch** enter the `Client ID` of a **Public** application, save, and connect via the OAuth Device Code Flow (the token is encrypted with DPAPI).
5. Optionally set the **Prediction duration** in **Settings → Operation mode**.
6. Start monitoring from **Overview** (or the wizard). Simulation is used by default.
7. For real points, separately enable real mode with confirmation in **Settings**.
8. The **Exit** button at the bottom stops the process immediately. Closing the window also stops the app after a few minutes of lost contact.

**Match History** shows your recent games; click any match to expand full game data — map, date and every player with their result, your entry highlighted.

## Built-in safety

The local interface listens only on `127.0.0.1` at a random port, the address is guarded by a random 192-bit path segment, API calls require a special header, POST requests are checked for Origin/Sec-Fetch-Site, CSP is enabled, and cross-origin requests are denied. Twitch tokens never reach the UI and are stored via Windows DPAPI; HotS files are opened read-only, without DLL injection, hooks, drivers, or game-memory reads. None of these mechanisms guarantees approval by Blizzard.

The logic from the older Go version (MPQ zlib/BZip2 handling, Twitch OAuth, Predictions, `replay.server.battlelobby`/`.StormReplay` watching) is preserved. State in `%LOCALAPPDATA%\HotSTwitchCompanion` is reused — **do not run two versions at the same time**. The old Node.js prototype is not compatible in state/token format.

## Building from source

```powershell
go test ./...
go build -trimpath -ldflags="-H=windowsgui -s -w" -o HotS-Twitch-Companion.exe .
```

Linux cross-compile: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H=windowsgui -s -w" -o HotS-Twitch-Companion.exe .`

Go 1.23, standard library only, no dependencies. `service.go` — HTTP/UI wiring, `main_windows.go` — Edge window, `core.go` — Twitch/HotS logic, `replay.go` — MPQ decoder, `i18n.go` — EN/RU message catalog, `web/` — the interface.

## Tests

MPQ, BZip2/zlib, result reading, file integrity, and local API tests pass on Linux; the `.exe` is cross-compiled and verified as a Windows x64 GUI PE. **The live Twitch API and on-screen behavior with real HotS have not been tested yet.** Check replays and simulation first; do not enable real Predictions until correctness is confirmed.

## GitHub and CI/CD

Workflow `.github/workflows/windows.yml`:

- on push to `main` / Pull Request / manual run: runs `go test ./...`, builds the Windows x64 `.exe`, and uploads a ZIP to **Actions → Artifacts**;
- on pushing a `v*` tag: additionally creates a **GitHub Release** with the ZIP + SHA256.

Local build in PowerShell:

```powershell
.\build.ps1
```

Result: `dist/HotS-Twitch-Companion.exe` and `HotS-Twitch-Companion-windows-x64.zip`.

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
