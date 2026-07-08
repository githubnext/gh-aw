# ADR-44015: Expose Memory Stores to `on.steps` in Pre-Activation

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw workflow engine compiles workflows into a `pre_activation` job and an agent job. Custom dispatch-gating steps defined via `on.steps` execute in `pre_activation`, but all three memory stores (cache-memory, repo-memory, comment-memory) were previously only restored in the agent job. This meant that any `on.steps` logic requiring prior memory state — for example, a gate step that reads a cached value to decide whether to proceed — either had to forgo memory access or consume an additional LLM turn in the agent phase to hydrate state. The inability to access memory in pre-activation limited the expressiveness of deterministic, non-LLM dispatch gates.

### Decision

We will restore all configured memory stores (cache-memory, repo-memory, comment-memory) in the pre-activation job before `on.steps` gates execute. The restore is **read-only**: it reuses the existing restore/load paths for each memory type and does not add cache-commit or repo-push write-back steps to pre-activation. Hydration is gated on the presence of `on.steps` in the workflow definition so that workflows without custom steps are unaffected.

### Alternatives Considered

#### Alternative 1: Execute `on.steps` in the Agent Job Instead of Pre-Activation

Move the `on.steps` execution point from `pre_activation` to the agent job, where memory is already available.

This would avoid the need to restore memory in a second location, but it would break the architectural contract that `on.steps` runs before the LLM is invoked. The entire purpose of `on.steps` is to gate activation deterministically and cheaply; running it in the agent job would consume a full LLM turn even when the gate rejects the request, negating the cost benefit.

#### Alternative 2: Require Workflows to Declare Their Own Memory Restoration Steps

Leave pre-activation unchanged and let workflow authors manually add memory restore steps before their `on.steps` gates via explicit YAML.

This avoids centralizing the restore logic in the compiler, but it places the burden on every workflow author to know the correct restore incantation for each memory type. It also creates drift risk: if the restore steps change, all affected workflows must be updated. The compiler already owns the pattern for generating memory steps; extending it is the consistent approach.

### Consequences

#### Positive
- `on.steps` gates can now read prior memory state to make deterministic dispatch decisions without spending an LLM turn.
- Reusing the existing restore/load paths ensures memory-type-specific details (branch names, cache keys, token handling) remain in a single authoritative location.
- The read-only constraint prevents pre-activation from accidentally mutating memory, preserving the invariant that only the agent job writes back.

#### Negative
- Every pre-activation run for a workflow with `on.steps` now executes additional restore steps, adding latency even when the gate does not use memory.
- The pre-activation job compiler gains a new code path (`buildPreActivationMemoryRestoreSteps`), increasing the surface area that must be maintained as memory store implementations evolve.

#### Neutral
- The change is scoped to workflows that define `on.steps`; workflows without `on.steps` are compiled identically to before.
- The `.github/workflows/daily-cli-performance.lock.yml` lock file is regenerated as a side effect of the compiler change, reflecting the new restore step.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
