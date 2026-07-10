# ADR-44650: Consolidate `copilot-requests: write` Permission Check on `*Permissions`

**Date**: 2026-07-10
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent (implementor)

---

### Context

The `copilot-requests: write` permission check — determining whether a workflow has granted Copilot token-auth enablement — was implemented inline at three separate call sites: `hasCopilotRequestsWritePermission` in `pkg/workflow/permissions_operations.go`, `shouldEmitCopilotRequestsEnableTip` in `pkg/workflow/permissions_compiler_validator.go`, and `hasCopilotRequestsWritePermission` in `pkg/cli/workflow_secrets.go`. Each re-derived the same `perms.Get(PermissionCopilotRequests) && level == PermissionWrite` logic independently. The `*Permissions` type already hosted an analogous method, `HasContentsReadAccess()`, establishing a prior pattern for centralising permission predicates on the type.

### Decision

We will add `(*Permissions).HasCopilotRequestsWrite() bool` as the single source of truth for the `copilot-requests: write` check. All three existing call sites are updated to delegate to this method. The method handles nil receivers, consistent with the existing `HasContentsReadAccess()` pattern.

### Alternatives Considered

#### Alternative 1: Keep Inline Checks at Each Call Site (Status Quo)

Each call site continues to call `perms.Get(PermissionCopilotRequests)` and compare the result to `PermissionWrite`. No shared helper is introduced.

This was rejected because the three implementations can silently diverge over time (e.g., one site adds nil-guard logic that the others miss), making the effective permission rule difficult to reason about and audit.

#### Alternative 2: Extract a Package-Level Helper Function

A package-private function `hasCopilotRequestsWrite(p *Permissions) bool` is extracted rather than adding a method to `*Permissions`.

This was rejected because a method on `*Permissions` is consistent with the existing `HasContentsReadAccess()` pattern, keeps the nil-safety concern inside the type, and is discoverable by callers who already hold a `*Permissions` value.

### Consequences

#### Positive
- Single location for the `copilot-requests: write` business rule — future changes to how this permission is evaluated affect all call sites uniformly.
- Built-in nil safety — callers no longer need to guard against a nil `*Permissions` before checking the permission.
- Follows the established `Has*` method convention on `*Permissions`, improving API consistency.

#### Negative
- Increases the public API surface of the `Permissions` type by one exported method.
- A bug introduced in `HasCopilotRequestsWrite()` now propagates to all three callers simultaneously, whereas previously each call site could only affect its own code path.

#### Neutral
- No change in observable behaviour at the call sites — the refactor is behaviour-preserving by construction.
- New unit tests are added for `HasCopilotRequestsWrite()` directly, covering nil, shorthand write-all, read-all, explicit write, explicit none, and write-all-with-explicit-none override cases.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
