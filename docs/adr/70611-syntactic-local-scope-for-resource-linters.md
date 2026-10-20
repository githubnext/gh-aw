# ADR-70611: Keep Custom Resource Linters Syntactic and Locally Scoped, with Single Diagnostic Ownership

**Date**: 2026-10-20
**Status**: Proposed
**Deciders**: PR author; repository maintainers review acceptance through PR #70611.

---

### Context

`pkg/linters/` ships custom `go/analysis` analyzers that look for resource-handling mistakes: `goroutineresourceleak`, `manualmutexunlock`, `fileclosenotdeferred`, `contextcancelnotdeferred`, `httprespbodyclose`, `slicemakezerolength`, `deferrerrorunchecked`, and `unchecked_deferredclose`, plus the shared `internal/resourcetracker` package. Issue #69881 listed gaps. Some analyzers did not unwrap parenthesized expressions. Bookkeeping statements between an acquisition and its `defer` caused false positives. Channels handed to helpers caused noisy leak reports. Zero-capacity hints were found only in adjacent statements. Two analyzers (`deferrerrorunchecked` and `unchecked_deferredclose`) also reported the same deferred `Close()` call. The analyzers match acquisitions and cleanups syntactically, without interprocedural or alias analysis, so each fix had to choose between reporting more and staying quiet.

### Decision

We will keep these analyzers syntactic and limited to one function. We will document that limit in the code and in `pkg/linters/README.md`, instead of trying to infer what helpers do:

- Only directly matched cleanup calls count as cleanup. Passing a resource to a helper (for example, `closeFile(f)`) does not prove cleanup. Acquisitions with no matched cleanup are not reported, because ownership may move to the caller.
- In `goroutineresourceleak`, bookkeeping may appear between an acquisition and its `defer` in the same block. The scan stops at possible exits (including synchronous builtin `panic`), loops, repeated acquisition of the same lock, or reassignment of the resource. Nested statements are inspected, but function literals are not executed as part of this scan.
- A channel passed to an opaque call, assigned to a local alias, stored (including through `append`), or captured by a closure is treated as outside the local channel check. These handoffs are not proof of safety. Built-in `len` and `cap`, comparisons, receives, and type conversions do not themselves hand off a channel. Returning a channel from the launched goroutine does not transfer ownership, because Go discards the return values.
- `slicemakezerolength` scans past unrelated statements and range loops, but stops when the target or range inputs are referenced or changed, or an intervening exit path makes evaluating the capacity expression earlier unsafe.
- Each pattern has one diagnostic owner. Direct zero-argument deferred `Close()` calls belong only to `unchecked_deferredclose`, and `deferrerrorunchecked` skips them. This keeps existing `nolint:unchecked_deferredclose` suppressions meaningful.
- All matching code uses the shared `astutil.UnwrapParenExpr` helper.

### Alternatives Considered

#### Alternative 1: Interprocedural / alias-aware analysis (for example, SSA-based)

Build on `go/ssa` or analysis facts to follow resources into helpers and through aliases. This could establish more precise cleanup and ownership facts, but would require control-flow and escape modeling rather than merely matching call shapes. It is not selected because it expands the scope and runtime cost of lightweight analyzers beyond closing the gaps in #69881. Local channel aliases are instead conservatively excluded without attempting to prove subsequent cleanup.

#### Alternative 2: Treat any helper handoff as a leak (conservative reporting)

Report channels and resources passed to opaque helpers, and require `nolint` at each call site. This catches more potential leaks but still does not guarantee safety without control-flow analysis. It is not selected because ordinary ownership transfers would create false positives and suppression churn in CI.

#### Alternative 3: Let both deferred-error analyzers report `Close()`

Keep the overlap between `deferrerrorunchecked` and `unchecked_deferredclose`. This needs no code change. We did not choose it because one site would need two suppressions, and existing `nolint:unchecked_deferredclose` directives would no longer silence every report.

### Consequences

#### Positive
- Fewer false positives from bookkeeping before `defer`, parenthesized expressions, and channel handoffs, so suppressions are rarer.
- Each deferred `Close()` site has one owning analyzer, and existing suppressions keep their meaning.
- The analysis limit is written down in the README and in package docs, so future contributors know what the analyzers guarantee.
- Regression fixtures cover every source report in #69881.

#### Negative
- Real leaks hidden behind helpers, aliases, or channel handoffs are not reported. These are false negatives by design.
- The rules for where scans stop (exits, loops, reassignment, references) add per-analyzer logic that must stay consistent as the analyzers change.
- `deferrerrorunchecked` no longer covers `Close()` on its own. If `unchecked_deferredclose` is disabled or excluded in some rollout, those sites go unchecked.

#### Neutral
- Acquisitions with no directly matched cleanup keep their current behavior (not reported).
- Interprocedural analysis is still possible later. If adopted, it should replace this ADR.

---

Acceptance remains part of maintainer review of PR #70611; this proposed record does not assert approval.
