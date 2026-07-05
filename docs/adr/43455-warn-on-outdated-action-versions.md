# ADR-43455: Warn on Outdated Action Versions in User-Provided Steps

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown

---

### Context

AI agents that generate workflow files frequently emit `uses:` step references with stale
major versions (e.g. `actions/checkout@v3`) even when a newer version is recorded in the
embedded `action_pins.json`. Before this change, those outdated tags were silently accepted
and pinned to the corresponding (old) SHA, giving users no signal that their workflow file
should be updated. The compiler already maintains an embedded list of latest action pins,
making it the natural place to surface this discrepancy at compile time rather than leaving
it to downstream runtime surprises.

### Decision

We will add a compile-time warning (`warnIfOutdatedActionVersion`) hooked into the single
action-pinning entry point (`applyActionPinToTypedStep`) that emits a stderr diagnostic
whenever a user-supplied version tag is strictly older than the latest version in the
embedded pin database. Partial major-only tags (e.g. `@v4`) are treated as floating
within-major references and only warned on when a higher major is available. SHA refs and
non-semver refs are silently skipped to avoid false positives. Warnings are deduplicated
per `repo@version` pair within a single compilation run via `WorkflowData.ActionPinWarnings`.

### Alternatives Considered

#### Alternative 1: Hard-fail compilation on outdated version tags

Reject the workflow at compile time with an error instead of a warning, forcing users to
update the version tag before proceeding. This would provide stronger enforcement but is
too disruptive: partial major tags like `@v4` are still valid floating references within
that major series, and treating them as errors would break legitimate workflows. A warning
preserves forward motion while still surfacing the issue.

#### Alternative 2: Silently upgrade the version tag to the latest

Automatically rewrite the `uses:` field to the latest version from `action_pins.json`
without user intervention. This avoids the warning noise but silently mutates
user-provided workflow content, violating the principle that the compiler should not
change user intent without explicit confirmation. It would also interact poorly with
SHA-pinning strategies where the caller expects a specific version to be pinned.

### Consequences

#### Positive
- Users and AI agents receive an actionable compile-time warning when a stale action
  version is specified, enabling them to update the workflow source rather than silently
  accumulating technical debt.
- Deduplication via `ActionPinWarnings` ensures repeated steps (pre-steps, steps,
  post-steps) produce exactly one diagnostic per `repo@version` pair per compilation run.
- SHA refs and branch refs are silently skipped, eliminating false positives for already-
  pinned workflows.
- Partial major tags (`@vN`) are correctly treated as floating references, avoiding
  spurious warnings for valid usage.

#### Negative
- The warning is written to stderr only; it is not surfaced in structured output or
  returned as a typed diagnostic, so programmatic consumers cannot easily filter or act
  on it.
- Warning accuracy depends on `action_pins.json` being kept up-to-date; a stale pin
  database means outdated versions may not be flagged.
- Teams that intentionally pin to an older major version for compatibility will see
  warnings they cannot currently suppress on a per-action basis.

#### Neutral
- The new `ActionPinWarnings` map is added to `WorkflowData`, slightly increasing the
  struct's memory footprint per compilation run.
- The feature is integrated at the single pinning entry point (`applyActionPinToTypedStep`),
  so all callers — user-provided steps, pre-steps, pre-agent-steps, and post-steps — get
  the check automatically without further changes.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
