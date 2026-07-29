# Formal Notes: forecast-compliance-fixtures/README.md

**Last formalized**: 2026-07-29-16-00-47
**Notation**: TLA+ / Z3 / F* (mixed)
**Issue**: (see workflow run 30468452952)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `BernoulliSuccessFromConclusion` | conclusion == "success" maps to Bernoulli success indicator 1, else 0 |
| P2 | `EffectiveTokensNonNegative` | total_effective_tokens must never be negative |
| P3 | `ZeroETFixtureIndicatesMissingArtifact` | zero-ET fixture models a run with no downloaded token artifact (T-FC-022) |
| P4 | `HighETFixtureExceedsOverflowThreshold` | high-ET fixture (>=1,000,000) probes overflow handling (T-ET-006) |
| P5 | `DurationSecondsNonNegative` | duration derived from run_started_at/updated_at must be >= 0 |
| P6 | `FailedFixtureHasFailureConclusion` | failed-run fixture must have conclusion == "failure" (T-FC-035) |
| P7 | `RunSummarySchemaFieldsPresent` | required RunSummary fields (run_id, conclusion, total_effective_tokens) present |
| P8 | `RunSummaryRoundTripSerialization` | JSON marshal/unmarshal round-trip stability (cache-hit determinism) |
| P9 | `RunStartedBeforeOrEqualUpdated` | run_started_at <= updated_at ordering invariant |
| P10 | `MonteCarloInputFixtureCompleteness` | across all 4 fixtures: valid conclusion, non-negative ET, timestamps set |

## Key Invariants

- ET (total_effective_tokens) is always >= 0.
- Bernoulli sampling is driven strictly by conclusion == "success".
- Duration is always computed as updated_at - run_started_at and must be non-negative.
- Cached RunSummary JSON must be stable under round-trip serialization (marker of full processing).

## Edge Cases Identified

- Zero-ET run (artifact not downloaded) — T-FC-022.
- Failed run (conclusion: failure) — T-FC-035.
- Very high ET (>= 1,000,000) — T-ET-006 overflow probing.
- Only `run_summary_minimal.json` is confirmed present on disk; the zero-ET, failed, and high-ET fixtures are documented in the README but may not yet be materialized as files.

## Notes for Future Runs

- Verify whether `run_summary_zero_et.json`, `run_summary_failed.json`, and `run_summary_high_et.json` now exist in specs/forecast-compliance-fixtures/; if so, drop the skip logic from TestMonteCarloInputFixtureCompleteness.
- Consider cross-referencing docs/src/content/docs/specs/forecast-specification.md (Section 12) directly in a future run for deeper Monte Carlo engine invariants (Poisson/Gamma sampling distributions) beyond fixture schema.
