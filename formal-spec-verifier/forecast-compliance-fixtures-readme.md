# Formal Notes: forecast-compliance-fixtures/README.md

**Last formalized**: 2026-08-25-15-43-48
**Notation**: mixed (TLA+-style state predicates, SMT/Z3-style arithmetic constraints)
**Issue**: (see workflow run for created issue number)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `SampleLimit` | Sampling respects --sample cap (T-FC-020) |
| P2 | `DateWindowCutoff` | Sampling respects --days cutoff (T-FC-021) |
| P3 | `MissingArtifactZeroET` | Missing artifact run: zero ET, still counted (T-FC-022) |
| P4 | `EmptySampleNilProjection` | Zero sampled runs => nil projection (T-FC-023) |
| P5 | `PartialObservation` | In-progress run w/ non-zero ET is partial (T-FC-024) |
| P6 | `HighETNoOverflow` | ET >= 1,000,000 handled without overflow (T-ET-006) |
| P7 | `KnuthForLowLambda` | lambda <= 15 uses Knuth's algorithm (T-FC-031) |
| P8 | `NormalForHighLambda` | lambda > 15 uses normal approx, non-negative (T-FC-032) |
| P9 | `ZeroLambdaZeroTokens` | lambda = 0 => projected tokens 0 (T-FC-033) |
| P10 | `BootstrapWithReplacement` | Bootstrap draws with replacement (T-FC-034) |
| P11 | `BernoulliGatesET` | Only successful Bernoulli draws contribute ET (T-FC-035) |
| P12 | `TrialCountIs10000` | Exactly 10,000 trials per workflow (T-FC-036) |
| P13 | `PercentileOrdering` | P10 <= P50 <= P90 (T-FC-037) |
| P14 | `P50FieldConsistency` | projected_effective_tokens == p50 (T-FC-038) |
| P15 | `LambdaCrossoverAt15` | lambda == 15 uses exact Knuth branch (T-FC-039/040) |

## Key Invariants

- Sampled runs must always respect both `--sample` and `--days` limits simultaneously.
- Runs missing the `aw_info.json` artifact are counted but contribute zero ET.
- The Monte Carlo engine always runs exactly 10,000 trials, gated by Bernoulli success and bootstrap-with-replacement ET sampling.
- Percentile fields must satisfy P10 <= P50 <= P90, and `projected_effective_tokens` is defined as P50.
- The lambda=15 boundary is inclusive of Knuth's exact algorithm; anything strictly greater uses the normal approximation.

## Edge Cases Identified

- Run with missing artifact (zero ET, still sampled).
- In-progress run with partial (non-zero) token snapshot, not a Bernoulli success.
- Run with ET >= 1,000,000 (overflow / NaN / Inf safety).

## Notes for Future Runs

- The actual production functions (`runMonteCarlo`, `poissonSample`, `useNormalApproximationForPoisson`, `gammaSample`) are unexported in `pkg/cli`, so the generated tests use local stubs mirroring their documented contracts. A follow-up could move the generated test into an internal `_test.go` file within `package cli` to test the real implementation directly instead of stubs.
- Did not yet formalize the Gamma-Poisson compound (Negative Binomial) posterior model for lambda uncertainty (R-MC-020, R-MC-030 minimum-observations threshold) — worth a deeper pass in a future run.
- Cross-reference: full canonical spec is `docs/src/content/docs/specs/forecast-specification.md`; this fixtures README is a pointer/bootstrap doc.
