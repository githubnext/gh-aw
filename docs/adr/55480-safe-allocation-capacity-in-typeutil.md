# ADR-55480: Centralize Safe Allocation Capacity Calculation in typeutil

**Date**: 2026-08-24
**Status**: Accepted
**Deciders**: Copilot

---

### Context

CodeQL flagged several `go/allocation-size-overflow` paths where summed `len(...)` values were passed directly as allocation capacity hints. If extremely large or malformed inputs caused those sums to overflow `int`, the resulting capacity could become negative or otherwise unsafe before reaching `make`.

The affected call sites were in multiple packages, including `pkg/workflow` and `pkg/cli`, so a package-private helper would either duplicate the overflow logic or leave future call sites without a shared convention.

### Decision

We will provide `typeutil.SafeAllocationCapacity(parts ...int) int` as the shared helper for allocation capacity hints built from multiple integer parts. The helper returns the summed capacity when every part is non-negative and the addition does not overflow; otherwise it returns zero so callers still allocate correctly without unsafe preallocation.

Call sites that previously used direct additive capacity expressions will use this shared helper when summing length-derived allocation hints across packages.

### Alternatives Considered

#### Alternative 1: Keep package-local helpers

Keeping separate helpers in `pkg/workflow` and `pkg/cli` avoids a new shared API, but it duplicates security-sensitive overflow handling and lets behavior drift between packages.

#### Alternative 2: Inline overflow checks at each allocation site

Inlining checks keeps each call site self-contained, but it makes the overflow policy harder to audit and increases the chance that a future allocation hint misses one of the required checks.

#### Alternative 3: Remove capacity hints entirely

Removing all summed capacity hints would also avoid overflow, but it discards useful preallocation for normal inputs and obscures the intended size relationship between the source collections and the destination allocation.

### Consequences

#### Positive

- CodeQL-flagged allocation capacity calculations now use a single overflow-safe helper.
- The zero-capacity fallback preserves correctness while avoiding unsafe preallocation on overflow or negative input.
- Future callers have one reusable helper for length-derived allocation hints.

#### Negative

- `pkg/typeutil` gains a small public API that should keep its current overflow semantics stable.

#### Neutral

- Valid inputs preserve the existing capacity hint behavior.
- Overflow and negative inputs may allocate with default growth instead of the original precomputed capacity.

---

*ADR finalized after implementation and CodeQL review.*
