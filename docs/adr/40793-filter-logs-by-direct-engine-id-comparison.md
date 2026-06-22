# ADR-40793: Filter Logs by Direct Engine ID String Comparison

**Date**: 2026-06-22
**Status**: Draft
**Deciders**: Unknown (auto-generated from PR #40793 by Design Decision Gate)

---

### Context

The `gh aw logs --engine <name>` command filters downloaded workflow runs by the agentic engine that produced them. Each run carries an `aw_info.json` artifact whose `engine_id` field records the engine (e.g. `claude`, `copilot`, `codex`). The original filter implementation re-parsed `aw_info.json` via `extractEngineFromAwInfo()` to obtain a `CodingAgentEngine` interface value, then reverse-looked-up its string name by iterating over every registered engine in the global engine registry and comparing interface identity. This roundabout path silently failed to match, so the `--engine` filter was effectively ignored and runs from all engines were returned regardless of the filter value. The parsed `awInfo` was already available earlier in the same code block, making the re-parse and registry lookup redundant.

### Decision

We decided to filter runs by comparing the `--engine` filter string directly against the already-parsed `awInfo.EngineID` field (`engineMatches = (awInfo.EngineID == engine)`). This removes the `extractEngineFromAwInfo()` re-parse, the `CodingAgentEngine` interface comparison, the registry reverse-lookup loop, and the now-unused `pkg/workflow` import. The same fix is applied in both `DownloadWorkflowLogs` and `DownloadWorkflowLogsFromStdin`. The primary driver is correctness — the filter must actually filter — with a secondary benefit of simplicity.

### Alternatives Considered

#### Alternative 1: Keep the registry reverse-lookup and fix the comparison bug

Retain `extractEngineFromAwInfo()` plus the registry loop, but correct whatever caused the interface-identity comparison to never match. Rejected because it preserves redundant work (re-parsing a file already parsed and looping the full registry per run) and keeps a fragile dependency on engine-instance identity, when the canonical engine identifier (`engine_id`) is already present as a plain string in the parsed `awInfo`.

#### Alternative 2: Normalize/validate the engine ID through the registry before comparing

Look the `engine_id` string up in the engine registry to canonicalize it (resolve aliases, reject unknown engines) and compare the canonical form. Rejected for now because `engine_id` is already the canonical identifier written by gh-aw itself, so an extra validation hop adds complexity without a concrete need; aliasing is not currently a requirement. [TODO: verify no engine aliases exist that would require normalization.]

### Consequences

#### Positive
- The `--engine` filter works correctly: a run is included only when its `engine_id` equals the requested engine.
- Less code and one fewer package dependency (`pkg/workflow` import removed); skip log messages now report the detected engine ID instead of "unknown".
- Avoids redundant file re-parsing and a per-run registry loop, so filtering is cheaper.

#### Negative
- Filtering now trusts the raw `engine_id` string verbatim, with no registry-backed validation; a malformed or unexpected value silently fails to match rather than surfacing an error.
- No alias/synonym resolution — if engine IDs ever gain aliases, exact string comparison would need revisiting.

#### Neutral
- A regression test (`logs_engine_filter_test.go`) replicates the filter logic and covers matching, non-matching, missing `aw_info.json`, and empty `engine_id` cases.
- Behavior depends on `aw_info.json` being present and populated; runs without it never match any `--engine` filter (treated as "unknown").

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
