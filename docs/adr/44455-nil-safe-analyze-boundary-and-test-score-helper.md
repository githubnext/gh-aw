# ADR-44455: Nil-safe `Analyze` boundary and test score helper for anomaly detection

**Date**: 2026-07-09
**Status**: Draft
**Deciders**: Unknown (generated from PR #44455 by adr-writer agent)

---

### Context

`AnomalyDetector.Analyze` accepted a `*MatchResult` pointer but had no nil-guard. In production `AnalyzeEvent` calls `Analyze` after a clustering operation that can produce a nil result on certain error paths, causing an unrecovered panic. Additionally, test assertions in `anomaly_test.go` hard-coded magic float literals (e.g., `0.65`, `0.15`) that duplicated the weight constants defined in `Analyze`, making tests brittle when weights change.

### Decision

We will add a nil-guard at the top of `Analyze` that returns a zero-value `AnomalyReport` (score 0, all flags false, reason "no anomaly detected") when the caller passes `nil`. We will also introduce an `anomalyScore` test-only helper that mirrors the weight constants from `Analyze`, so test expectations stay in sync with production scoring logic without duplicating magic literals.

### Alternatives Considered

#### Alternative 1: Panic with a descriptive message on nil input

Treat nil as a programming error — add `if result == nil { panic("Analyze: nil MatchResult") }`. This makes misuse louder, forcing callers to fix the bug rather than silently swallowing it. Rejected because the call site in `AnalyzeEvent` legitimately reaches this path during error handling and a panic there would surface as an unrecoverable runtime failure rather than a handled error.

#### Alternative 2: Change the API to `(*AnomalyReport, error)`

Return `(nil, error)` when `result == nil`, surfacing the anomaly as an explicit error the caller must handle. This is the most type-safe option and makes nil-path observable at compile time. Rejected because it requires changing every call site across the package and is a larger API break than the targeted bug fix warrants; callers currently rely on `Analyze` always returning a non-nil `*AnomalyReport`.

### Consequences

#### Positive
- Eliminates the production panic on nil `*MatchResult`; the anomaly pipeline continues operating safely.
- Test expectations are derived from production weight constants via the helper, so a weight change in `Analyze` propagates automatically to test failures rather than silently passing.
- `t.Parallel()` added to all stateless sub-tests reduces wall-clock test time.

#### Negative
- A caller that passes nil due to a logic bug gets no feedback — the nil maps silently to "no anomaly detected", which could mask incorrect upstream behavior.
- The `anomalyScore` helper duplicates the scoring algorithm in test code; if the scoring structure changes significantly (e.g., new flags added) the helper must be updated manually alongside the production code.

#### Neutral
- `TestAnalyzeEvent` sub-tests are not parallelised because they share a miner with accumulated state; the sequential dependency is now made explicit in comments.
- `assert.Nil` → `require.Nil` in the error branch of `TestNewAnomalyDetector_ThresholdBoundaries` prevents further assertions on a nil detector after a validation error, which is a test correctness improvement with no production impact.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
