# ADR-34545: Align Gemini Metrics with Shared Finalization and Collapse Safe-Output Helper Files

**Date**: 2026-05-25
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

A semantic-clustering pass over `pkg/workflow` surfaced three drift artifacts that share a single root cause: helpers had been split into files that no longer carried their semantic weight. `GeminiEngine.ParseLogMetrics` wrote turns, token usage, and a deduplicated tool-call slice directly onto `metrics`, bypassing the shared `FinalizeToolMetrics` contract that every other engine uses to canonicalize ordering and structure. `compiler_safe_outputs_env.go` existed solely to host the 25-line `addAllSafeOutputConfigEnvVars` method — its single production caller (`buildHandlerManagerStep`) lived in a different file. `compiler_safe_outputs_core.go` retained the unused `SafeOutputStepConfig` struct and a stale "moved files" navigation comment after earlier refactors (ADR-26297) had relocated every implementation function out of it. Together these created file sprawl in `pkg/workflow` and silent engine-behavior divergence between Gemini and the other log parsers.

### Decision

We will (1) refactor `GeminiEngine.ParseLogMetrics` to collect `turns`, `tokenUsage`, and a `toolCallMap` into local variables and finalize them through `FinalizeToolMetrics(...)`, matching the contract used by other engines; (2) move `(*Compiler).addAllSafeOutputConfigEnvVars` into `compiler_safe_outputs_steps.go` next to its sole production caller `buildHandlerManagerStep` and delete `compiler_safe_outputs_env.go`; and (3) delete `compiler_safe_outputs_core.go` together with the unused `SafeOutputStepConfig` type and stale navigation comment, redirecting its surviving marshal-error log call to `compilerSafeOutputsConfigLog`. This extends the file-organization convention from ADR-27325 / ADR-28282 (semantic homes for helpers) and the splitting convention from ADR-26297 (concern-based safe-outputs files) by pruning files that no longer earn their existence.

### Alternatives Considered

#### Alternative 1: Keep Gemini's Bespoke Tool-Call Assembly

`GeminiEngine.ParseLogMetrics` worked correctly — its in-place writes produced the right metric values. Leaving the function as-is would avoid touching engine code that is exercised in production. This was rejected because the bespoke path also bypassed `FinalizeToolMetrics`'s ordering and structural guarantees, so any future change to the shared finalization contract (deterministic ordering, tool-call dedupe semantics) would silently skip Gemini. Drift between engines was the root cause the cluster pass identified, and patching it locally would have re-introduced the same drift at the next refactor.

#### Alternative 2: Keep `compiler_safe_outputs_env.go` and `compiler_safe_outputs_core.go` as Stable Anchors

The two safe-output files could remain in place: `_env.go` for symmetry with the other concern-split files from ADR-26297, and `_core.go` as a navigation index for the consolidated job. This was rejected because (a) `_env.go` hosted a single method called from one site in a sibling file — the split added a file-boundary hop without isolating a real concern, and (b) `_core.go` no longer contained any implementation; its `SafeOutputStepConfig` struct had no callers and its "moved to…" comment was already stale. A file that exists only to point elsewhere is documentation that rots; deletion is more honest than maintenance.

#### Alternative 3: Move the Env Helper Into a New Shared Utility File

`addAllSafeOutputConfigEnvVars` could have been moved to a new `compiler_safe_outputs_helpers.go` (or similar catch-all). This was rejected because it recreates the umbrella pattern ADR-27325 explicitly discourages: it preserves file count without preserving semantic meaning. Co-locating with the production caller in `_steps.go` puts the helper next to the code path that actually invokes it, making the call graph readable inside one file.

### Consequences

#### Positive

- Gemini engine metrics now flow through the same `FinalizeToolMetrics` path as the other engines, so future changes to ordering / dedupe / token aggregation semantics apply uniformly without per-engine patches.
- `compiler_safe_outputs_steps.go` now contains both `buildHandlerManagerStep` and its env-var helper, making the safe-outputs step-construction call graph readable in one file.
- File count in `pkg/workflow` drops by two, removing two safe-output files that no longer carried real concerns and aligning the package surface with the live code paths.
- New focused test coverage in `gemini_logs_test.go` asserts the post-refactor contract (token aggregation, tool-call dedupe with counts, deterministic ordering, invalid-line tolerance), guarding against regression if `FinalizeToolMetrics` is later changed.

#### Negative

- `git log` / `git blame` for `addAllSafeOutputConfigEnvVars` now starts at this PR unless callers use `--follow`; the original authorship signal moves to the relocation commit.
- The deleted `SafeOutputStepConfig` struct, even though unreferenced, could in principle have been re-used by a future consolidated-step abstraction; reviving it later would require resurrecting it from history.
- Test coverage previously implicitly covered Gemini's bespoke assembly through end-to-end log-parsing tests; the new unit tests must now be maintained as the canonical guard, and any drift between them and `FinalizeToolMetrics` semantics will require a coordinated update.

#### Neutral

- No public API or workflow YAML schema surface changes; the refactors are purely internal to `pkg/workflow`.
- The surviving marshal-error log call in `compiler_safe_outputs_config.go` switches from `consolidatedSafeOutputsLog` to the file-local `compilerSafeOutputsConfigLog`, a cosmetic alignment with the file's own logger; no log output text changes.
- `consolidatedSafeOutputsLog` is removed along with `compiler_safe_outputs_core.go`; any future safe-outputs file that needs a shared logger **MUST** introduce one explicitly rather than relying on a phantom global.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Engine Log-Metric Finalization

1. `GeminiEngine.ParseLogMetrics` **MUST** finalize tool calls, turns, and token usage via `FinalizeToolMetrics(FinalizeToolMetricsOptions{...})` and **MUST NOT** assign `metrics.ToolCalls`, `metrics.Turns`, or `metrics.TokenUsage` by hand after that finalization call.
2. New `Engine.ParseLogMetrics` implementations **MUST** collect tool-call data into a `map[string]*ToolCallInfo` and **MUST** delegate ordering, dedupe, and per-turn aggregation to `FinalizeToolMetrics`.
3. Per-engine bespoke tool-call slice assembly (appending `ToolCallInfo` directly to `metrics.ToolCalls`) **MUST NOT** be reintroduced; any engine-specific aggregation **SHOULD** happen prior to the finalization call into the shared `toolCallMap`.

### Safe-Output Helper File Organization

1. `(*Compiler).addAllSafeOutputConfigEnvVars` **MUST** reside in `pkg/workflow/compiler_safe_outputs_steps.go` alongside `buildHandlerManagerStep`.
2. `pkg/workflow/compiler_safe_outputs_env.go` **MUST NOT** be recreated; env-var step construction for safe outputs **MUST** live next to the step-builder that consumes it.
3. `pkg/workflow/compiler_safe_outputs_core.go` **MUST NOT** be recreated as a navigation-only file. A safe-outputs file **MUST** carry real implementation, not a "moved to…" pointer or an unreferenced struct.
4. The `SafeOutputStepConfig` struct **MUST NOT** be reintroduced unless a concrete caller is added in the same change.

### Logger Scoping for Safe-Output Files

1. Log statements inside a `compiler_safe_outputs_<concern>.go` file **MUST** use a logger named for that file's concern (e.g., `compilerSafeOutputsConfigLog` in `compiler_safe_outputs_config.go`) and **MUST NOT** reach into loggers defined in sibling files.
2. A shared cross-file logger for safe-outputs (analogous to the removed `consolidatedSafeOutputsLog`) **MUST NOT** be reintroduced; if multiple files need the same log channel, each file **MUST** declare its own logger with a file-scoped name.

### Test Coverage for Gemini Log Parsing

1. `pkg/workflow/gemini_logs_test.go` **MUST** assert that `GeminiEngine.ParseLogMetrics` aggregates token counts across multiple JSON lines, deduplicates tool calls with correct counts, returns a deterministic tool-call ordering via finalization, and tolerates non-JSON / empty lines without panicking.
2. New behavioral changes to `GeminiEngine.ParseLogMetrics` **SHOULD** be accompanied by corresponding cases in `gemini_logs_test.go`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. In particular, hand-assembling `metrics.ToolCalls` in `GeminiEngine.ParseLogMetrics`, restoring `compiler_safe_outputs_env.go` or `compiler_safe_outputs_core.go`, reintroducing `SafeOutputStepConfig` without a caller, or reaching into a cross-file safe-outputs logger each constitute non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26378565904) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
