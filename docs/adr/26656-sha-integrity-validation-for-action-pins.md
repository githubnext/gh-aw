# ADR-26656: SHA Integrity Validation for Action Pins in `actions-lock.json`

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot [TODO: verify]

---

## Part 1 — Narrative (Human-Friendly)

### Context

`actions-lock.json` pins GitHub Actions to specific commit SHAs to provide
reproducibility and supply-chain security guarantees. A latent bug in
`gh aw update` could produce a lock file in which two distinct action
repositories (e.g., `github/gh-aw-actions/setup` and
`github/gh-aw-actions/setup-cli`) share the same commit SHA. Because each
repository represents a different code tree, a shared SHA is an invalid pin
state that either signals data corruption or a potential supply-chain
tampering attempt. Prior to this decision the validation path logged a
non-fatal warning and continued, meaning the invalid state could be silently
persisted and propagated.

### Decision

We will add hard-failure SHA-collision detection at two enforcement points:
(1) on load and on pre-save inside `UpdateActions`, and (2) inside
`ValidateActionSHAsInLockFile` which is called by `compile --validate`.
Collisions — defined as the same commit SHA appearing across two or more
distinct action repositories — are treated as fatal errors with actionable
remediation guidance, not as warnings. Multiple versions of the *same*
repository are allowed to share a SHA (e.g., `actions/checkout@v5` and
`actions/checkout@v5.0.1`), because that is a normal, expected state.

### Alternatives Considered

#### Alternative 1: Warning-Only (Status Quo)

The existing behaviour logged a warning and let the operation continue. This
was rejected because silent continuation on an integrity violation defeats the
purpose of SHA pinning: a downstream user or CI job running against a
lock file with colliding SHAs has no clear signal that the pins are invalid.

#### Alternative 2: Automatic Remediation

Automatically re-resolve colliding SHAs by re-fetching fresh resolution data
from GitHub at conflict time. This was rejected because the conflict indicates
an unknown source of truth — re-fetching could silently overwrite a
deliberately set pin and would mask a supply-chain attack rather than surfacing
it.

#### Alternative 3: Single Validation Point (Compile Only)

Validate only at `compile --validate` and not during `update`. This was
rejected because late detection leaves the lock file in an invalid state until
the next compile cycle, widening the window in which the corrupted file can
be checked in and distributed.

### Consequences

#### Positive
- Corrupted or tampered lock files are detected and rejected immediately,
  reducing the supply-chain attack surface.
- Both `update` and `compile --validate` paths provide consistent, symmetric
  integrity guarantees.
- Error messages include the conflicting SHAs (truncated) and all involved
  repository names, giving users enough context to investigate without reading
  source code.

#### Negative
- Any legitimate repository pair that genuinely resolves to the same commit
  SHA (an extremely unlikely but theoretically possible edge case in monorepo
  setups) would be blocked and require manual lock-file inspection.
- Two parallel implementations exist: `ActionCache.ValidateUniqueActionSHAs`
  (cache-level) and `ValidateDistinctActionSHAs` (lock-file-level). Both must
  be kept in sync if the collision semantics ever change.

#### Neutral
- `compile --validate` now returns a non-zero exit code where it previously
  printed a warning and returned success; existing scripts or CI steps that
  treat that exit code as advisory will see a behavioural change.
- Test coverage must explicitly verify the same-repo/multi-version allowance
  as well as the cross-repo rejection path.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**,
> **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in
> this section are to be interpreted as described in
> [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### SHA Collision Definition

1. A SHA collision **MUST** be defined as the same commit SHA string appearing
   in `actions-lock.json` under two or more distinct action repository paths
   (i.e., paths that differ after normalisation).
2. Multiple entries for the same repository path (e.g., different version
   tags resolving to the same commit) **MUST NOT** be treated as a collision.

### `UpdateActions` Enforcement

1. `UpdateActions` **MUST** validate `actions-lock.json` for SHA collisions
   immediately after loading the file and before processing any updates.
2. `UpdateActions` **MUST** validate `actions-lock.json` for SHA collisions
   a second time immediately before persisting the updated file.
3. If either validation detects a collision, `UpdateActions` **MUST** return
   a non-nil error and **MUST NOT** persist the file.
4. The error message **SHOULD** include the truncated SHA and the full
   repository paths of all conflicting entries to allow the user to diagnose
   the problem without reading source code.

### `compile --validate` Enforcement

1. `CompileWorkflowWithValidation` and `CompileWorkflowDataWithValidation`
   **MUST** return a non-nil error when `ValidateActionSHAsInLockFile` detects
   a SHA collision.
2. A SHA-collision error **MUST NOT** be downgraded to a warning or log
   message; it **MUST** propagate to the caller as a fatal error.
3. The error message **SHOULD** include a remediation hint directing the user
   to run `gh aw update-actions` and recompile.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all
**MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or
**MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24520084941) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
