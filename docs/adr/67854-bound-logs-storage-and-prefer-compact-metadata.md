# ADR-67854: Bound `gh aw logs` storage by default and prefer compact metadata artifacts

**Date**: 2026-10-12
**Status**: Draft
**Deciders**: pelikhan (PR author); gh-aw maintainers (review pending)

---

### Context

Agentic workflows in this repository prefetch up to 3000 runs of logs on every scheduled execution (for example the shared `aw-logs-24h-fetch` templates used by nine daily workflows). [Detection Analysis Report run 38183895070](https://github.com/github/gh-aw/actions/runs/38183895070) exhausted the GitHub Actions runner disk before the agent even started, and the previous day's run failed the same way. Two factors combined to cause this: `gh aw logs --max-storage` defaulted to `0` (unlimited), so a bulk download had no storage ceiling at all; and the `aw_info.json` fallback path always downloaded *activation* artifacts, which embed large repository snapshots, even when a compact `info` artifact carrying the same metadata was available on the run. Runner disk is a hard, non-negotiable constraint shared by every workflow using these templates, and the failure mode is total (the job dies before doing any useful work).

### Decision

We will make `gh aw logs` bound by default: `--max-storage` now defaults to `1024` MB (1 GB) instead of unlimited, with `--max-storage 0` retained as the explicit opt-out for unlimited storage. We will also change the `aw_info.json` fallback to prefer compact `info` artifacts and only fall back to activation artifacts when no `info` artifact yields `aw_info.json`, downloading candidates one at a time and stopping at the first success. The primary driver is operational reliability: a safe-by-default budget prevents a single bulk download from taking down a runner, and preferring compact metadata reduces the bytes needed to obtain the same information. Because the MCP `max_storage` argument must distinguish "omitted" from "explicitly zero", its type changes from `int` to `*int` so an explicit `0` still means unlimited.

Shared single-target, multi-target, and stdin APIs normalize an omitted storage budget to 1024 MB, including the in-process `models --refresh` caller. `LogsDownloadOptions` and `StdinLogsOptions` retain `MaxStorageMB int` for existing positive budgets and add `MaxStorageMBSet bool` to distinguish omitted zero from explicit unlimited storage. Callers requesting unlimited storage set `MaxStorageMB: 0, MaxStorageMBSet: true`. Continuation JSON always includes `max_storage`, including zero, so resuming an unlimited request does not restore the default budget.

### Alternatives Considered

#### Alternative 1: Leave the default unlimited and set `--max-storage` per workflow

Each shared prefetch template could simply pass an explicit budget (the pre-existing templates already passed `--max-storage 10240` in some cases). This keeps the CLI backward compatible and avoids a major changeset. It was rejected because the failure mode is silent and repeats for every caller that forgets the flag — including ad-hoc local invocations and future workflows — so the unsafe default would keep re-introducing the same outage. Several of the affected templates in this very diff had no budget at all, which is exactly the failure this PR is fixing.

#### Alternative 2: Keep activation-artifact fallback and rely only on pruning / `--prune-older-runs`

Rather than changing artifact preference, we could lean entirely on the storage budget plus `--prune-older-runs` to evict data after it lands on disk. This was rejected because pruning is reactive: the large activation snapshot must be fully downloaded (and briefly occupy disk) before it can be pruned, and the budget is soft with respect to concurrent in-flight downloads. Preferring the compact `info` artifact avoids transferring the bytes in the first place, which also saves wall-clock time and API bandwidth. Activation fallback is retained for legacy runs where no `info` artifact exists.

#### Alternative 3: Pick a smaller or larger default budget (e.g. 256 MB or 10 GB)

A smaller ceiling would be safer still but would truncate legitimate multi-run analyses; a 10 GB ceiling matches what some templates already used but is close enough to the runner's free disk to still fail. 1 GB was chosen as a value that comfortably holds a 24-hour, 3000-run compact `info,usage` collection while leaving ample headroom on a standard runner.

### Consequences

#### Positive
- Bulk `gh aw logs` downloads can no longer exhaust runner disk by default; the Detection Analysis Report and the other eight regenerated daily workflows get a 1 GB ceiling.
- Metadata-only fallbacks transfer substantially less data by preferring compact `info` artifacts over activation snapshots containing repository contents.
- The fallback loop short-circuits on the first artifact that yields `aw_info.json`, avoiding unnecessary downloads of the remaining candidates.
- The shared prefetch templates now request `info,usage` explicitly, making the intended artifact set self-documenting rather than implicit.

#### Negative
- This is a breaking change for any existing script or workflow that relied on unlimited default storage; such callers must now pass `--max-storage 0` (captured in a `major` changeset and migration guidance).
- The budget is *soft*: concurrent in-flight downloads can temporarily exceed 1 GB, so the guarantee is best-effort rather than a hard cap.
- Log collections that legitimately need more than 1 GB will now silently collect less data unless the budget is raised, so reports must document incomplete coverage when collection is limited.
- The MCP `max_storage` field becomes a pointer (`*int`), adding nil-handling complexity in `appendLogsFilterArgs` and any future consumer of `logsArgs`.

#### Neutral
- The existing reporting window (`--start-date -1d`) and run ceiling (`--count 3000`) are unchanged; only artifact selection and storage bounds move.
- Nine workflow `.lock.yml` files were regenerated purely as a consequence of the shared-template edits; the diff there is mechanical (hashes plus the `logs` command line).
- Activation-artifact flattening logic is preserved and merely renamed (`downloadActivationAwInfoFallback` → `downloadAwInfoFallback`, `resolveActivationArtifactNames` → `resolveAwInfoArtifactNames`) to reflect the broadened responsibility.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
