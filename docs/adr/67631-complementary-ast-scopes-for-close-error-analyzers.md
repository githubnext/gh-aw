# ADR-67631: Treat the Two Close-Error Analyzers as Complementary AST Scopes Under One Policy

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR author); gh-aw maintainers (review pending)

---

### Context

gh-aw ships two custom Go analyzers that both concern discarded `Close()` errors:
`pkg/linters/closeerrorunchecked` and `pkg/linters/unchecked-deferred-close`. Issue #67608
reported that their guidance contradicted itself — the `closeerrorunchecked` fixtures labelled
`defer f.Close()` as "best practice — not flagged" while `unchecked_deferredclose` reports exactly
that construct as an unhandled error. The underlying cause is that the analyzers match different AST
node kinds: `closeerrorunchecked` inspects `ast.ExprStmt` and blank-identifier assignments (including
statements inside deferred closures), while `unchecked_deferredclose` inspects `ast.DeferStmt`. The
constraint is that neither analyzer may be weakened, both must keep their existing CI rollout
exclusions, and the same unchecked call must not be reported twice.

### Decision

We will treat the two analyzers as complementary scopes of a single, documented close-error policy
rather than as overlapping or competing checks. Concretely: the analyzer descriptions, package
comments, and fixtures now state which node kinds each analyzer owns; silence from one analyzer is
explicitly declared *not* to be approval of the construct; the `unchecked_deferredclose` diagnostic
recommends a deferred closure that handles the cleanup error; and `pkg/linters/README.md` gains a
"Close error policy" section with an `errors.Join` example that preserves both the operation error and
the cleanup error. A combined-analyzer regression fixture
(`closeerrorunchecked/testdata/src/combined`) runs both analyzers over one file and asserts exactly
one finding per unchecked call, locking the scope boundary in place. The primary driver is
correctness of developer guidance, not changed enforcement: no detection logic was altered.

### Alternatives Considered

#### Alternative 1: Merge the two analyzers into one

A single analyzer handling `ExprStmt`, assignments, and `DeferStmt` would make the policy
self-evidently consistent and remove any chance of future scope drift. It was rejected for this
change because the two analyzers have independent registry entries and independent CI rollout
exclusions tied to distinct backlogs of existing findings; merging them would force both backlogs to
be remediated at once and would change which suppression directive existing `nolint` comments must
use. This was a close call and remains a reasonable follow-up once the exclusions are retired.

#### Alternative 2: Relax `unchecked_deferredclose` so bare `defer c.Close()` is accepted

This would resolve the contradiction by making the fixtures' original "best practice" claim true, and
would have been the smallest change. It was rejected because a bare deferred `Close()` genuinely
discards the error — guaranteeing cleanup runs and handling the cleanup failure are separate
requirements — and relaxing the analyzer would silently reduce coverage of real resource-cleanup
bugs across the repository.

### Consequences

#### Positive

- The contradictory "defer is best practice" guidance is removed, so developers receive one coherent
  rule from fixtures, analyzer descriptions, and the README.
- The combined fixture is an executable specification of the scope boundary: a future change that
  makes either analyzer report an out-of-scope node, or report a node twice, fails the test.
- The `unchecked_deferredclose` diagnostic is now actionable — it names the deferred-closure remedy
  instead of the vague "consider handling errors explicitly".

#### Negative

- The policy still depends on two analyzers staying in sync by convention plus one fixture; a
  structural guarantee would require the merge rejected in Alternative 1.
- The combined test reaches across package boundaries and re-implements a miniature multi-analyzer
  driver (cloning `analysis.Pass` and wrapping `Report`), which couples the
  `closeerrorunchecked` test package to `unchecked-deferred-close` internals and may need updating if
  `analysis.Pass` gains fields.
- Documenting `defer c.Close()` as non-compliant makes the size of the existing remediation backlog
  more visible without retiring the CI rollout exclusions that hide it.

#### Neutral

- No enforcement behaviour changes: the same code that passed before this PR still passes, so no
  existing call sites require edits as a direct result.
- The `errors.Join` idiom is now the repository's recommended remedy, which implies named result
  parameters on functions that must propagate cleanup errors.
- The combined fixture lives under `closeerrorunchecked/testdata`, so cross-analyzer regressions are
  discovered by that package's test rather than by a neutral shared location.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
