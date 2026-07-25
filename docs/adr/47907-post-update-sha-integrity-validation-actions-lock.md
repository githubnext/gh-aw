# ADR-47907: Post-Update SHA Integrity Validation for actions-lock.json

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw update` command resolves and pins GitHub Action commit SHAs and container image digests into `.github/aw/actions-lock.json`. Prior to this change, the file was written without any final structural check, meaning malformed entries (e.g., a truncated SHA, a mismatched map key, or an incorrectly formatted container digest) could be silently persisted and later used in production workflows. The only time a bad entry would be caught was at workflow execution time — far downstream from where the corruption was introduced.

### Decision

We will add a terminal validation phase at the end of `RunUpdateWorkflows` that re-reads and structurally verifies all entries in `actions-lock.json` after the update, recompile, and pin-refresh steps complete. Validation failures surface as a hard error (`update validation failed: ...`) rather than being silently ignored. This validation only runs when no earlier error occurred (`firstErr == nil`), so it does not mask upstream failures.

### Alternatives Considered

#### Alternative 1: Inline validation during update (validate-as-you-write)

Each action or container entry could be validated at the point it is written into the cache, as part of the existing resolve-and-pin logic. This would catch errors closer to their origin and avoid re-reading the file. It was not chosen because it would require threading validation logic into multiple existing write paths (action resolution, container pinning, recompile step), significantly increasing the coupling and risk of regressions. A single terminal pass is simpler and keeps validation isolated from write logic.

#### Alternative 2: No post-update validation (status quo)

The existing approach relied on correctness guarantees from the upstream resolution APIs and the write path. Malformed entries would only surface at workflow-run time. This was rejected because supply-chain integrity requires catching bad pins before they are persisted; silent failures at the lock-write layer are a security risk in an action-pinning tool.

### Consequences

#### Positive
- Malformed or inconsistent action commit SHAs and container digests are caught immediately at update time, before being committed to the repository.
- The validator produces aggregated, human-readable diagnostics covering all invalid entries at once, rather than stopping at the first error.
- Validation logic is isolated in a dedicated file (`update_validation.go`) with injected resolver functions, making it straightforward to unit-test without real network calls.

#### Negative
- The terminal validation phase makes additional GitHub API calls (one per action entry, one per container entry) to re-resolve SHAs. For repositories with many pinned actions this adds latency to every `gh aw update` invocation.
- Transient API failures during validation will cause the update command to return an error even when the lock file is structurally correct, potentially creating spurious failures in CI.

#### Neutral
- The validation step is gated on `firstErr == nil`, so it does not run when the main update phase already encountered an error. This is consistent with the existing error-propagation strategy but means validation is skipped on partial updates.
- Verbose mode logs a confirmation message to stderr when validation succeeds, matching the existing log pattern used elsewhere in the update flow.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
