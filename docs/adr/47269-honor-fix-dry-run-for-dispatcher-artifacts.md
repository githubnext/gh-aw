# ADR-47269: Honor `gh aw fix` Dry-Run for Dispatcher Artifacts

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw fix` command documents two modes: dry-run (default, no file mutations) and write mode (`--write`). However, the dispatcher artifact helpers — `ensureAgenticWorkflowsDispatcher` and `ensureAgenticWorkflowsAgent` — unconditionally created parent directories and wrote files regardless of the `write` flag. This violated the advertised contract: running `gh aw fix` without `--write` would still mutate `.github/skills/agentic-workflows/SKILL.md` and `.github/agents/agentic-workflows.md` when they drifted from the bundled templates. Fixing #47141, the core requirement is that dry-run must leave all files byte-for-byte unchanged while still reporting what would change.

### Decision

We will thread a `write bool` parameter through `ensureAgenticWorkflowsDispatcher` and `ensureAgenticWorkflowsAgent`, and extract a shared `writeGeneratedRepositoryInstructionFile` helper that hard-errors if called with `write=false`. In dry-run mode the ensure functions compare existing content against the generated template and emit a `Would create/update …` message to stderr, then return without touching the filesystem. The `init` and `upgrade` callers always pass `write=true`, so their behavior is unchanged.

### Alternatives Considered

#### Alternative 1: Skip dispatcher refresh entirely in dry-run

Guard the two `ensure*` calls at the `runFixCommand` call site with `if write { ... }`. This is simpler — no parameter threading, no new helper. However, it silently swallows drift: users running dry-run would not learn that their dispatcher files are stale, defeating the purpose of a check-all-workflows diagnostic mode. Rejected because drift reporting in dry-run is explicitly desirable.

#### Alternative 2: Accept a no-op write path without a hard guard

Allow `writeGeneratedRepositoryInstructionFile` to accept `write=false` and simply return `nil` silently. This avoids an "internal error" return but makes it easy to introduce future callers that forget to check `write` before calling the helper, silently skipping writes. Rejected because the explicit error acts as a compile-time-equivalent invariant: if a future caller passes `write=false` it immediately surfaces rather than silently no-oping.

### Consequences

#### Positive
- Dry-run mode now correctly reports drift (`Would create/update dispatcher skill: …`) without mutating any files.
- The `writeGeneratedRepositoryInstructionFile` helper centralises `EnsureParentDir` + `WriteFile` + error formatting, reducing duplication across the two ensure functions.
- Both sides of the contract are explicitly covered by tests (`TestFixCommand_DryRunDoesNotUpdatePromptAndAgentFiles`, `TestFixCommand_WriteUpdatesPromptAndAgentFiles`, `TestWriteGeneratedRepositoryInstructionFile_RefusesDryRun`).

#### Negative
- All existing callers of the two `ensure*` functions required a mechanical signature update (adding the `write` argument), increasing the diff surface area slightly.
- The `writeGeneratedRepositoryInstructionFile` helper returns an error on `write=false`; this path should never be reached in production, but it adds a code path that cannot be tested through the normal call graph without deliberate misuse.

#### Neutral
- `init` and `upgrade` callers now explicitly pass `true` for `write`, making the write intent visible at call sites rather than implied by function design.
- CLI help text for `gh aw fix` was updated to list dispatcher refresh as a write-mode action, bringing documentation into sync with behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
