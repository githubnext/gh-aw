# ADR-50642: Additive Permissions Merge for Built-in Jobs

**Date**: 2026-08-05
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler generates built-in jobs (`safe_outputs`, `conclusion`) with a least-privilege permission set computed automatically. Authors sometimes need scopes beyond this baseline — most notably `id-token: write` for OIDC token minting — and express these by declaring a `permissions` block under `jobs.<built-in>` in their workflow file. Prior to this change, `applyBuiltinJobAugmentations` only consulted `needs` and `if` fields from a built-in job's config entry; a `permissions` block was silently skipped, causing the declared scopes to be absent from the compiled lock file. This caused OIDC token minting to fail at runtime with "Missing id-token permission" errors despite the author having explicitly declared `id-token: write`.

### Decision

We will support user-declared `permissions` blocks under `jobs.<built-in>` entries in `applyBuiltinJobAugmentations`, merging them **additively** with the compiler-computed least-privilege permissions. The merge applies "write overrides read" semantics and preserves all compiler-required scopes. A new helper `applyBuiltinJobPermissionsAugmentation` encapsulates the merge logic and is invoked before the existing `needs`/`if` augmentation path.

### Alternatives Considered

#### Alternative 1: Reject permissions blocks on built-in jobs (error on detection)

Treat any `permissions:` key under a built-in job config entry as a validation error. Authors would be required to rely solely on the compiler's least-privilege computation and could not extend it. This is the safest option for minimizing permission surface, but it makes OIDC-dependent workflows impossible to author without the compiler growing built-in knowledge of every OIDC-related scope.

#### Alternative 2: Full replacement of compiler-computed permissions

Allow the user's declared permissions to entirely replace the compiler-computed set, giving authors complete control. This is simpler to implement but risks silently dropping required scopes (e.g., `issues: write` that `safe_outputs` needs to post comments), which would cause different runtime failures. The additive approach avoids this regression.

### Consequences

#### Positive
- OIDC-dependent workflows that declare `id-token: write` under `jobs.safe_outputs.permissions` or `jobs.conclusion.permissions` now compile correctly and retain the scope in the lock file.
- Additive merge preserves all compiler-required scopes; no existing workflow is affected by this change (the new path is only entered when a `permissions` block is present).

#### Negative
- Authors can widen the permission surface of compiler-generated built-in jobs beyond the least-privilege baseline, potentially granting scopes that are not strictly needed at runtime.
- The compiler gains a new augmentation code path for permissions that must be kept in sync with future changes to built-in job generation and the `Permissions.Merge` / `RenderToYAML` helpers.

#### Neutral
- The error message for invalid built-in job augmentation now also covers the `.permissions` field, improving diagnostics when a user references a built-in job that the workflow does not generate.
- The integration test (`builtin_job_permissions_integration_test.go`) compiles a real workflow fixture and asserts lock-file contents, establishing a regression guard for this behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
