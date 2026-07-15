# ADR-45573: Add --engine, --repo, and --approve Flags to upgrade and update Commands for CLI Consistency

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw` CLI exposes several workflow management commands — `compile`, `validate`, `update`, and `upgrade` — that share an underlying compilation pipeline. A CLI consistency audit identified four high-severity flag gaps: the `upgrade` command was missing `--engine/-e` (AI engine override) and `--repo/-r` (target repository), while the `update` command was missing `--approve` (bypass strict-mode enforcement). Sibling commands already supported these flags, creating asymmetry that forced users to use different commands for equivalent operations or to work around missing controls.

The `--approve` flag controls whether the compiler allows action additions/removals and restricted secrets not already present in the `gh-aw-manifest` during strict-mode compilation. The `--engine` flag allows overriding the AI engine used during workflow compilation. The `--repo` flag dispatches compilation work to a remote repository.

### Decision

We will add `--engine/-e` and `--repo/-r` to `upgrade` and `--approve` to `update`, threading the new values through to the shared compilation pipeline (`compileWorkflowWithRefresh`, `UpdateActionsInWorkflowFiles`, `recompileAllWorkflows`). We will also enforce `--repo`/`--org` mutual exclusion on `upgrade`, matching the existing guard on `update`. The `--no-security-scanner` gap on `compile`/`validate` (F3) is intentionally skipped because those commands operate exclusively on local files where the scanner is a no-op.

### Alternatives Considered

#### Alternative 1: Document the Asymmetry and Provide Workarounds

Accept that `upgrade` and `update` expose fewer controls than `compile`/`validate`, and add documentation explaining which flags are available on which commands. Users needing `--engine` during upgrade would run a separate `compile` pass afterward.

This was rejected because it degrades usability: users running upgrade in CI cannot override the engine mid-pipeline without invoking an extra command, and the asymmetry is not obvious from `--help` output alone.

#### Alternative 2: Add Flag Declarations Without Wiring Them to the Pipeline

Register the new flags on `upgrade`/`update` to satisfy the consistency checker but silently ignore their values, deferring the pipeline integration to a later PR.

This was rejected because it would introduce misleading no-op flags — a worse UX than missing flags — and any CI automation that relied on the flags would silently get wrong behavior.

#### Alternative 3: Unify Commands Behind a Single Orchestration Layer

Refactor `upgrade` and `update` to delegate to `compile` internally, so any flag supported by `compile` is automatically available to callers.

This was considered but rejected as out of scope for a consistency fix. It would require a larger architectural refactor that risks regressions across the entire compilation pipeline.

### Consequences

#### Positive
- `upgrade` and `update` now accept the same engine, repo, and approval controls as their sibling commands, removing user-visible asymmetry.
- Users can override the AI engine during upgrade without a separate `compile` pass — important in CI pipelines where compilation must use a specific engine.
- The `--repo`/`--org` mutual exclusion guard on `upgrade` matches `update` behavior, preventing ambiguous invocations.
- Flag coverage is verified by new unit tests for registration, description consistency, and mutual exclusion.

#### Negative
- `compileWorkflowWithRefresh` and several internal helpers now take an additional `approve bool` parameter, growing the function signatures. Seven call sites required updates.
- The `approve` value is threaded through multiple layers (`update_command.go` → `update_manifest.go`, `update_workflows.go`, `update_actions.go`), increasing the surface area for future parameter drift if the compilation interface changes.

#### Neutral
- F3 (`--no-security-scanner` on `compile`/`validate`) is explicitly deferred and documented as a no-op case; no code or flag was added.
- Documentation in `docs/src/content/docs/setup/cli.md` for both `update` and `upgrade` is updated to reflect the new flags.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
