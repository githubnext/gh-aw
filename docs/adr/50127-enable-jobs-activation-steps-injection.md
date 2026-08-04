# ADR-50127: Enable `jobs.activation.steps` Injection

**Date**: 2026-08-04
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw compiler supports built-in activation jobs and shared workflow imports, but Squad initialization previously used `jobs.activation.pre-steps`. Those steps run before the activation checkout, which means setup that depends on the checked-out activation context cannot reliably prepare files before the activation artifact is staged.

Workflow authors need a narrowly scoped extension point for the activation job that runs after activation checkout and after the activation gate/output logic, but before the activation artifact is staged and uploaded for later jobs. This placement lets shared workflows prepare activation-time content without moving the work into unrelated jobs or relying on custom artifact handling.

### Decision

We will support `jobs.activation.steps` as an activation-only built-in job injection field. The compiler inserts these steps after the generated activation checkout/gate sequence and before activation artifact staging/upload. Imported activation `steps` are merged in import declaration order before the main workflow's activation `steps`, preserving the same layered behavior as other imported step-injection fields.

The field is intentionally limited to the activation job. `jobs.<other-built-in>.steps` is rejected with a compile-time error so unsupported built-in job step injection is not silently ignored. Activation injected steps are included in GitHub CLI permission inference, and the staging/upload insertion anchors use shared step-name constants so compiler-generated names and insertion matching stay synchronized.

### Alternatives Considered

#### Alternative 1: Keep using `jobs.activation.pre-steps`

This was rejected because `pre-steps` run before the activation checkout. Squad initialization needs the activation checkout to be available before preparing content for the activation artifact.

#### Alternative 2: Add a new top-level activation-specific field

A separate field such as `activation-steps` could express the lifecycle phase without nesting under `jobs.activation`. This was rejected because built-in job customization already lives under the `jobs.<built-in>` namespace, and keeping the field there makes import merging and validation consistent with existing job customization behavior.

#### Alternative 3: Allow `steps` on every built-in job

This was rejected because each built-in job has different generated ordering and security constraints. Allowing the field everywhere would either require broader injection semantics that are not currently designed, or would silently drop unsupported steps and surprise authors.

### Consequences

#### Positive

- Shared workflows can prepare activation artifact content after activation checkout and before artifact staging.
- Squad setup can move out of `pre-steps` and run at the lifecycle point it actually requires.
- Unsupported built-in job `steps` fields now fail fast instead of being silently ignored.
- Import ordering for multiple activation step providers is deterministic and covered by regression tests.

#### Negative

- The built-in job customization surface now has another lifecycle-specific field, increasing documentation and validation responsibilities.
- Insertion before staging depends on compiler-generated activation staging/upload anchors staying synchronized with the insertion logic.

#### Neutral

- `jobs.activation.steps` does not change the semantics of custom jobs; custom job `steps` remain ordinary GitHub Actions job steps.
- The ADR file name uses the PR number, matching the repository's existing ADR naming convention.

---

*ADR created for PR #50127 to satisfy the Design Decision Gate before human review and merge.*
