# ADR-41553: Extract Shared Org-Wide Runner with Callback Struct

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `runUpdateForOrg` and `runUpgradeForOrg` functions in `pkg/cli/` each contained a complete, nearly identical implementation of the same structural algorithm: discover repositories in an org, filter by glob, optionally scan each repo for pending work with rate-limit awareness, sort results by oldest-edit time, render a preview report, and then apply changes or open issues — continuing past per-repo failures. This duplication meant that improvements to one command (such as graceful SIGTERM handling or skip-on-failure semantics) required manual replication to the other. As a result, the two commands had quietly diverged: `runUpgradeForOrg` lacked signal handling, stopped on the first repo error, and processed repos in arbitrary order.

### Decision

We will extract the shared org-wide loop into a single `runCommandForOrg` function in `pkg/cli/org_runner.go`, parameterised by an `orgRunCallbacks` struct containing pluggable functions (`SearchFn`, `ScanFn`, `ReportFn`, `ApplyFn`, `IssueFn`) and optional message-override string fields. Both `runUpdateForOrg` and `runUpgradeForOrg` become thin wrappers that build the appropriate callbacks and delegate to `runCommandForOrg`. All cross-cutting concerns — input validation, signal handling, rate-limit awareness, cancellation, sorting, reporting, and skip-on-failure — live in one place.

### Alternatives Considered

#### Alternative 1: Keep duplication and sync manually

Both commands could retain their full inline implementations, with developers responsible for propagating improvements between them. This avoids introducing any new abstraction and keeps each command self-contained and easy to read in isolation. It was rejected because the existing divergence (missing signal handling and stop-on-first-error in `upgrade`) demonstrated that manual synchronisation is unreliable at this codebase's rate of change.

#### Alternative 2: Interface-based abstraction (`OrgCommandRunner` interface)

Define an interface with methods for each phase (`Search`, `Scan`, `Report`, `Apply`, `Issue`) and implement it separately for update and upgrade. This is more type-safe and idiomatic for large Go codebases. It was not chosen because the two implementations differ only in a small number of leaf functions, not in control flow — an interface would require more boilerplate (two concrete types) for the same outcome, without providing the additional ability to pass `nil` for the optional `ScanFn` to skip the scan phase.

#### Alternative 3: Embed a shared base struct

Embed a common `orgBase` struct in `updateOrgRunner` and `upgradeOrgRunner`, placing shared fields on the base while each concrete type supplies its own methods. Rejected for the same reasons as the interface approach: too much scaffolding for what is effectively a single-function difference.

### Consequences

#### Positive
- The `upgrade` org command gains capabilities it previously lacked: graceful SIGTERM/Ctrl-C handling, skip-failed-repos semantics (instead of stop-on-first-error), and stable sort by oldest-edit time.
- Future cross-cutting improvements (e.g. progress bars, telemetry, dry-run output) need to be written once.
- The test rename `TestRunUpgradeForOrgStopsOnFirstError` → `TestRunUpgradeForOrgSkipsFailedRepos` makes the new contract explicit.

#### Negative
- `orgRunCallbacks` is a large struct (11 fields) mixing required functions with optional string overrides; callers must know which fields are mandatory versus optional, as there is no compile-time enforcement.
- The callback pattern (fields of function type) is harder to mock and unit-test in isolation compared to an interface, since each test must construct a full `orgRunCallbacks` literal even when only one callback is exercised.

#### Neutral
- The `orgRepoPreview` type, previously defined in `update_org.go`, remains there — it is shared implicitly via package scope, not moved to `org_runner.go`.
- The `orgUpdateLog` logger used inside `runCommandForOrg` continues to be declared in `update_org.go`; the runner depends on that package-level variable.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
