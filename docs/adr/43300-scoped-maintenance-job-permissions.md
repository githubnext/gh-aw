# ADR-43300: Scoped Permissions for Maintenance Jobs via Job Splitting and Selective Disablement

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown (Copilot SWE agent, pelikhan)

---

### Context

The generated `agentics-maintenance.yml` contained a single `close-expired-entities` job that always requested `discussions: write`, `issues: write`, and `pull-requests: write` together — regardless of which entity types a repo actually used. Similarly, jobs such as `apply_safe_outputs` and the label-triggered jobs were always emitted, forcing write scopes on repos that had no need for them. This violated least-privilege principles and increased the blast radius of any token compromise. Repos that handled only issues (no discussions, no PRs) still received all three write permissions. There was no mechanism for repo owners to opt out of individual maintenance jobs without forking the generated workflow or disabling maintenance entirely.

### Decision

We will split `close-expired-entities` into three separate jobs (`close-expired-discussions`, `close-expired-issues`, `close-expired-pull-requests`), each requesting only the single write permission it needs. We will also add a `maintenance.disabled_jobs` array to `aw.json` that allows repo owners to omit specific maintenance jobs from the generated workflow. Job IDs in the list are normalized (case-insensitive, `_` and `-` treated equivalently). When `apply_safe_outputs` is disabled, the `workflow_call.outputs.applied_run_url` output falls back to `inputs.run_url` to avoid dangling job-output references. When all label-triggered jobs are individually disabled, the `issues: labeled` trigger is suppressed.

### Alternatives Considered

#### Alternative 1: Conditional steps within the existing monolithic job

Keep the single `close-expired-entities` job but add `if:` conditions on each step (discussions, issues, PRs) so only the relevant steps run. This avoids splitting the job definition.

Why not chosen: The job-level `permissions:` block is evaluated regardless of which steps actually run; GitHub Actions does not support per-step permission narrowing. A repo that disabled the discussions step would still receive `discussions: write` on the runner token — defeating the least-privilege goal.

#### Alternative 2: Extend the existing `label_triggers: false` pattern with per-job boolean flags

Instead of a free-form `disabled_jobs` array, add individual boolean fields (e.g., `close_expired_discussions: false`, `apply_safe_outputs: false`) to the maintenance config schema.

Why not chosen: Each new maintenance job would require a corresponding schema field, making the config schema grow proportionally with the workflow. A string list of job IDs scales to any number of jobs without schema changes, and the normalization strategy (case-insensitive, `_`/`-` equivalence) makes the values human-friendly. The downside is that typos in job IDs silently have no effect — a validation warning or enumeration constraint could be added later.

#### Alternative 3: Generate separate maintenance workflow variants per repo type

Detect the repo's entity mix (issues-only, discussions-only, etc.) automatically from repository settings and emit a pre-tailored workflow with only the relevant jobs and permissions.

Why not chosen: Auto-detection would require additional API calls at compile time and could misclassify repos mid-lifecycle (e.g., a repo that enables discussions after initial setup). Explicit opt-out via `disabled_jobs` gives repo owners deterministic, auditable control without silent automation surprises.

### Consequences

#### Positive
- Each `close-expired-*` job now requests only the one write permission it needs, eliminating unnecessary token scopes.
- Repos can omit entire maintenance jobs (including `apply_safe_outputs` and label-triggered jobs) without disabling maintenance globally.
- The `issues: labeled` trigger is automatically suppressed when all label-triggered jobs are disabled, reducing event processing overhead.
- The refactored `writeCloseExpiredJob` helper eliminates code duplication across the three close-expired jobs.

#### Negative
- The generated `agentics-maintenance.yml` now contains up to three separate jobs where there was previously one, increasing YAML verbosity.
- `disabled_jobs` entries with typos silently have no effect; there is no validation error for unknown job names. [TODO: verify whether a schema `enum` constraint should be added]
- The normalization logic (`_`/`-` equivalence, case-insensitivity) adds a subtle invariant that must be maintained consistently across the config parser, YAML builder, and tests.

#### Neutral
- Existing repos that do not set `disabled_jobs` see no behavioral change; all jobs continue to be generated as before (backward-compatible default).
- The `workflow_call` output `applied_run_url` changes its source expression when `apply_safe_outputs` is disabled — callers of the maintenance workflow via `workflow_call` should be aware this value may now reflect `inputs.run_url` rather than the job's own output.
- Tests were updated to assert per-job scoped permissions and to cover the new `disabled_jobs` config path.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
