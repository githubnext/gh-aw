# ADR-70611: Keep Custom Resource Linters Syntactic and Locally Scoped, with Single Diagnostic Ownership

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: pelikhan (PR author) [TODO: verify reviewers]

---

### Context

`pkg/linters/` ships custom `go/analysis` analyzers that look for resource-handling mistakes: `goroutineresourceleak`, `manualmutexunlock`, `fileclosenotdeferred`, `contextcancelnotdeferred`, `httprespbodyclose`, `slicemakezerolength`, `deferrerrorunchecked`, and `unchecked_deferredclose`, plus the shared `internal/resourcetracker` package. Issue #69881 listed gaps. Some analyzers did not unwrap parenthesized expressions. Bookkeeping statements between an acquisition and its `defer` caused false positives. Channels handed to helpers caused noisy leak reports. Zero-capacity hints were found only in adjacent statements. Two analyzers (`deferrerrorunchecked` and `unchecked_deferredclose`) also reported the same deferred `Close()` call. The analyzers match acquisitions and cleanups syntactically, without interprocedural or alias analysis, so each fix had to choose between reporting more and staying quiet.

### Decision

We will keep these analyzers syntactic and limited to one function. We will document that limit in the code and in `pkg/linters/README.md`, instead of trying to infer what helpers do:

- Only directly matched cleanup calls count as cleanup. Passing a resource to a helper (for example, `closeFile(f)`) does not prove cleanup. Acquisitions with no matched cleanup are not reported, because ownership may move to the caller.
- In `goroutineresourceleak`, bookkeeping may appear between an acquisition and its `defer` in the same block. The scan stops at possible exits, loops, or reassignment of the resource. A channel passed to an opaque call, stored, returned, or captured by a closure is treated as outside the local check. These handoffs are not treated as proof of safety. Built-in `len` and `cap` calls do not count as handoffs.
- `slicemakezerolength` scans past adjacent statements and stops when the target or range inputs are referenced or changed.
- Each pattern has one diagnostic owner. Direct zero-argument deferred `Close()` calls belong only to `unchecked_deferredclose`, and `deferrerrorunchecked` skips them. This keeps existing `nolint:unchecked_deferredclose` suppressions meaningful.
- All matching code uses the shared `astutil.UnwrapParenExpr` helper.

### Alternatives Considered

#### Alternative 1: Interprocedural / alias-aware analysis (e.g., SSA-based)

Build on `go/ssa` or analysis facts to follow resources into helpers and through aliases. That would prove cleanup instead of skipping these cases. We did not choose it because it adds a lot of complexity and runtime cost to a set of lightweight, mostly independent analyzers. It is also a much larger change than closing the gaps in #69881.

#### Alternative 2: Treat any helper handoff as a leak (conservative reporting)

Keep reporting channels and resources passed to opaque helpers, and require `nolint` at each call site. This gives the strongest guarantees. We did not choose it because it creates false positives that teach developers to suppress the analyzers, which hurts their value in CI.

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

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
