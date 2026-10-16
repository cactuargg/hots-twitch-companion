# BattleLobby research and fixture provenance

The four unmodified binary fixtures in this directory come from
[HeroesToolChest/Heroes.StormReplayParser](https://github.com/HeroesToolChest/Heroes.StormReplayParser/tree/d381dd9c273205c0fd7ac402c75eef98c6fa6c78/Tests/Heroes.StormReplayParser.Tests/BattleLobbys),
commit `d381dd9c273205c0fd7ac402c75eef98c6fa6c78`, under the accompanying MIT
[LICENSE](LICENSE). The Go layout is adapted from that project's
[ReplayServerBattlelobby.cs](https://github.com/HeroesToolChest/Heroes.StormReplayParser/blob/d381dd9c273205c0fd7ac402c75eef98c6fa6c78/Heroes.StormReplayParser/MpqFiles/ReplayServerBattlelobby.cs).
These are actual game files, not a synthetic invented wire format.

| File suffix | SHA-256 |
| --- | --- |
| ARAM1_89754 | b2360245375f1e4411145845e04571e39d72f1fcf12df48c6614689da6b91408 |
| ARAM2_93640 | 320ae2612d38b1559c67a03410c68328382ed51d371347b62730b0094c23f789 |
| Custom1_90670 | 4ad54ea0a493a88758040ba14fe61b1b36d6be7c3c52129ffea5e9a4324be7ab |
| StormLeague1_89566 | 10e952df59eac0d35e38f8d2ee5912ad70fbd15bee6c63cf2d37c499180e0e22 |

## Findings and implementation boundary

- The temporary `replay.server.battlelobby` is a raw bit stream, whereas
  `replay.details` is a versioned object inside an MPQ archive. The existing
  MPQ/versioned parser cannot decode this lobby by substituting a member name.
- Dependencies/cache handles precede attribute definitions, selections, mod and
  collection tables, then client identities. Bit chunks consume low bits within
  each byte and concatenate in big-endian order. Blobs align to the next byte.
- We extract occupied working-set slots, raw slot types (`Humn`/`Comp`), roles
  (`Part`/`Watc`), hero attribute IDs, numeric ToonHandles, team size, the custom
  flag, client build, random seed and AMM mode code. `50091` means Storm League;
  `50101` means ARAM. AMM 0 is absent/unknown; custom is stored separately.
- Custom1 has one human observer and ten AI participants: client identity count
  is not participant count. We preserve all occupied slots; AI ToonHandles stay
  absent. Slot order is **not** assigned a team number. Hero IDs can be empty.
- Supported builds are explicitly 89566, 89754, 90670 and 93640. Add another real
  fixture and verify its layout before extending the allowlist. This is not a
  general promise of support for every build >= 85027.
- The parser ends after client identities and the AMM choice. Later cosmetic
  metadata is opaque, so `roster_parsed` does not certify the whole file. A file
  truncated *after* this prefix can still yield a roster. Unsupported branches,
  missing prefix data, bad indices, inconsistent builds or duplicate identities
  yield an error with no partial result. Read size is limited to 8 MiB.
- No common Match ID has been demonstrated. Random seed is not guaranteed unique;
  SHA-256 describes these lobby bytes, not an ID found in `replay.details`.
  ToonHandles are player identities, not match identities.
- Map names in the reference parser depend on external Battle.net cache files.
  We do not open paths embedded in untrusted lobby data. Names, BattleTags,
  shortcut IDs, cosmetics and account data are skipped rather than stored.

## Persistence and scope

The scanner stores the latest eligible observation as `state.json:lastLobby`,
including parser version, status, file basename, source timestamp/size/hash and
the parsed roster (or error). Stat checks reject files replaced or changed during
capture. As with any metadata check, same-size writes restoring timestamps are
not guaranteed detectable. Only the latest observation is retained; no archive
or retention UI is introduced. Existing state files remain readable.

The observation is independent of `active`, and never enters settlement rules.
Capturing a newer lobby while a Prediction is active must not relabel that
Prediction. Correlation, repeated rosters, ambiguous matches, startup recovery
policy and payout changes belong to PR #19. Existing automatic settlement remains
as on main; this PR does not claim to fix its correlation limitations.

Tests use upstream expected build/seed/mode, Muradin/Ragnaros and ToonHandle
assertions, observer/AI counts, malformed/truncated inputs, wire order, capture
hashes, state round trips, legacy state, and scanner observation without modifying
an active simulation Prediction. A fuzz target is included.
