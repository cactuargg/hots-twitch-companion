# Architectural refactoring audit

Audit base: `09d7418` (PRs #67–#85), following the original specification based on `7949214`.

| Planned step | Already merged | Remaining at audit |
| --- | --- | --- |
| 1. Configuration | #67: schema, paths, defaults and persistence | Remove command configuration wrapper after consumer migration |
| 2. Storage | #68, #69, #74, #78: DPAPI, credentials, reset, migration, state store | Move reset authorization into application; remove path adapter |
| 3. Twitch client | #71–#73: Device Flow, creation validation, settlement reconciliation | Move synchronized client and token refresh; inject transport and credentials |
| 4. HotS correlation | #70, #76 | Remove forwarding functions and migrate remaining callers/tests |
| 5. Engine | #75, #82–#84: queue, configuration/start policies, scan model | Move Engine, lifecycle, monitoring and dependency composition into app |
| 6. Dashboard | #79, #81, #85: updates, HTTP helpers and guarded router | Move service and API actions; replace Engine field access with operation/snapshot contract |
| 7. Assets and entry points | #77: embedded web assets | Reduce command to platform adapters and composition; verify build metadata/icon paths |
| 8. Adapter cleanup | Not complete | Remove compatibility aliases and forwarding functions after migration |
| 9. Documentation and validation | Not complete | Architecture guide; both packages, race CI and recovery/security tests |

The earlier local release-check extraction is preserved on `codex/release-version-validation`.
It must not be applied wholesale: #79 already implements the extraction and rejects malformed/overflowing versions.

Validation uses fake Twitch APIs or intercepted/local HTTP transports. No test may issue real Twitch prediction writes.

## Completion

- PR #86 completed the synchronized Twitch client and injected transport/token-store boundary. Standard/Desktop tests, both Windows packages and race CI passed before merge.
- PR #87 moved Engine and monitoring to `internal/app`, Dashboard and API actions to `internal/dashboard`, and removed production command aliases/forwarding files. Dashboard uses a consumer-owned operation contract and independent state snapshots. All 94 legacy test/fuzz entry points were retained; new tests cover snapshot ownership and per-instance dependencies. Standard/Desktop tests, vet, both packages, all 64 Node tests and race CI passed before merge.
- The final documentation/cleanup PR adds `docs/architecture.md`, updates both READMEs and the version-source reference, removes the remaining settlement forwarding method and makes embedded resources private to their owning package.

The nine planned outcomes are implemented. The command now contains composition, version/linker metadata, Windows launchers/profile locking, Wails/tray adapters and the generated resource. Test-only fixtures preserve readable legacy integration setup; they are not compiled into either executable.

Real Twitch authorization and payouts are intentionally outside automated validation. Device authorization, refresh/rotation, ownership checks, failures and recovery are covered through injected transports and local/fake APIs. Final validation requires successful CI on the merged `main` as well as on each PR.
