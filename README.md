# HotS Twitch Companion v0.4

Portable Windows x64 EXE with a dark interface styled after Heroes of the Storm. The GUI is built from local HTML/CSS/JS embedded into the executable with `go:embed`. A separate desktop window is opened through the installed Microsoft Edge in `--app` mode; without Edge, your default browser is used. **[Читать на русском](README.ru.md)**

The app watches local Heroes of the Storm files, and when a match starts it creates a **Twitch Prediction** ("HotS win?"). When the match ends, the replay is parsed, the tracked player's win/loss is determined, and the Prediction resolves automatically.

## Key features

- **English and Russian interface** — switch any time via the `EN / RU` control in the top bar or in Settings. Backend log and error messages follow the selected language.
- **Configurable Prediction duration** — the betting window (1–30 minutes) is selected in Settings, just like the duration picker in the Twitch Predictions interface. Default: 2 minutes.
- **Simulation mode by default** — full pipeline runs without touching real channel points until you explicitly enable real Predictions.
- Read-only MPQ replay parser (zlib/BZip2) written from public file-format documentation — no DLL injection, hooks, drivers, or game-memory access.

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
