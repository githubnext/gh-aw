# Formal Notes: forecast-compliance-fixtures/README.md

**Last formalized**: 2026-08-13-15-43-30
**Notation**: mixed (TLA+ / Z3-style)
**Issue**: (see create_issue safe output result)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `BernoulliSuccess` | conclusion=success counts as a Bernoulli success |
| P2 | `CancelledNotSuccess` | cancelled/failure/timed_out runs sampled but not successes |
| P3 | `DurationNonNegative` | duration_seconds from updatedAt/startedAt is >= 0 |
| P4 | `ETObservationMapping` | total_effective_tokens maps to bootstrap ET observation |
| P5 | `ZeroETExcludedFromAIC` | AIC <= 0 excluded from bootstrap sampling |
| P6 | `HighETNoOverflow` | high AIC converts to milli-units without overflow |
| P7 | `MonteCarloNilGuard` | runMonteCarlo nil for degenerate inputs |
| P8 | `ReliabilityThreshold` | IsReliable flips at n >= 10 |
| P9 | `PercentileMonotonicity` | P10 <= P50 <= P90 |
| P10 | `SuccessRateBounded` | success rate in [0,1] |
| P11 | `PartialRunETValid` | in-progress run ET snapshot non-negative |
| P12 | `RunIDConsistency` | top-level run_id == run.id |

## Key Invariants

- Bernoulli success sampling only counts `conclusion == "success"`.
- Cancelled/failed/timed-out runs remain in the sample denominator (n) but not the success numerator.
- Monte Carlo bootstrap excludes any run with AIC <= 0 (missing/zero ET).
- Milli-AIC integer conversion (`int(math.Round(runAIC*1000))`) must not overflow for realistic AIC ranges.
- `runMonteCarlo` returns nil for empty observations or non-positive/NaN/Inf runsPerPeriod.
- IsReliable threshold is exactly 10 observations (`minObservationsForReliableForecast`).

## Edge Cases Identified

- T-FC-022: missing/zero ET (artifact not downloaded) — `run_summary_zero_et.json`.
- T-FC-035: failure conclusion for Bernoulli sampling — `run_summary_failed.json`.
- T-ET-006: very high ET (>= 1,000,000) overflow checks — `run_summary_high_et.json`.
- T-FC-036: cancelled run, in-sample but not success, ET forced to zero — `run_summary_cancelled.json`.
- T-FC-024: in-progress run with non-zero partial token snapshot — `run_summary_partial_et.json`.
- Malformed data: updatedAt before startedAt (defensive negative-duration guard, not explicitly documented in spec but tested).

## Notes for Future Runs

- `runMonteCarlo` in `pkg/cli/forecast_montecarlo.go` is unexported; a follow-up formalization pass could add an exported test seam so real (non-stub) tests can call it directly instead of using stub doubles.
- Additional fixtures referenced in the README (`run_summary_zero_et.json`, `run_summary_failed.json`, `run_summary_high_et.json`, `run_summary_cancelled.json`, `run_summary_partial_et.json`) do not appear to exist yet as files in the fixtures directory — only `run_summary_minimal.json` was found. A future run could formalize the gap between documented and actually-present fixtures.
- Cross-spec dependency: this fixture spec depends on `docs/src/content/docs/specs/forecast-specification.md` Section 12 (T-FC-0xx test IDs) — worth formalizing directly in a future pass for deeper Monte Carlo engine coverage (T-FC-031 through T-FC-040).
