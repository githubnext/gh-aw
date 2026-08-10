# Forecast Compliance Fixtures

This directory contains fixture files for bootstrapping the Section 12 compliance tests of the
[Forecast Specification](../../docs/src/content/docs/specs/forecast-specification.md).

## Fixture Files

### `run_summary_minimal.json`

A minimal `run_summary.json` fixture conforming to the `RunSummary` schema used by `pkg/cli/`.
This fixture represents a single successful workflow run (`daily-report`) with:

- `conclusion: "success"` — the run is counted as successful in Bernoulli sampling
- `token_usage_summary.total_effective_tokens: 5400` — the ET observation used in bootstrap resampling
- `run.updated_at` and `run.run_started_at` — used to compute `duration_seconds`

Use this fixture as the baseline for Monte Carlo engine compliance tests (**T-FC-031** through
**T-FC-040**) by loading it as a cached run summary.

## How to Run Compliance Tests

The forecast compliance tests are located in `pkg/cli/forecast_montecarlo_test.go` and
`pkg/cli/forecast_test.go`.

To run the full forecast compliance test suite:

```bash
go test -v -run "TestForecast" ./pkg/cli/
```

To run only the Monte Carlo engine tests (covering T-FC-031–T-FC-040):

```bash
go test -v -run "TestMonteCarlo" ./pkg/cli/
```

To run with the race detector (recommended for CI):

```bash
go test -race -run "TestForecast|TestMonteCarlo" ./pkg/cli/
```

## Fixture Schema Reference

The `run_summary_minimal.json` fixture follows the `RunSummary` struct defined in
`pkg/cli/logs_models.go`. Key fields used by the forecast command:

| JSON Field | Go Field | Forecast Usage |
|---|---|---|
| `run.conclusion` | `Run.Conclusion` | Bernoulli success probability |
| `run.updated_at` | `Run.UpdatedAt` | Duration computation |
| `run.run_started_at` | `Run.RunStartedAt` | Duration computation |
| `token_usage_summary.total_effective_tokens` | `TokenUsage.TotalEffectiveTokens` | Bootstrap ET sample |
| `run_id` | `RunID` | Run identification |

## Adding New Fixtures

To add a fixture covering a specific compliance scenario:

1. Copy `run_summary_minimal.json` and modify the relevant fields.
2. Name the fixture descriptively (e.g., `run_summary_zero_et.json` for T-FC-022).
3. Document the fixture purpose and the test IDs it covers in this README.

### Available Additional Fixtures

| Fixture Name | Purpose | Test IDs |
|---|---|---|
| `run_summary_zero_et.json` | Run with missing/zero ET (artifact not downloaded) | T-FC-022 |
| `run_summary_failed.json` | Run with `conclusion: "failure"` for Bernoulli sampling | T-FC-035 |
| `run_summary_high_et.json` | Run with very high ET (≥ 1,000,000) for overflow checks | T-ET-006 |
| `run_summary_cancelled.json` | Run with `conclusion: "cancelled"` (included in sample but not a Bernoulli success; ET is zero because the run did not complete) | T-FC-036 |

### Section 12 Coverage Gaps

The fixture files above cover only the scenarios that require a variant cached run summary.
The remaining Section 12 test IDs of the parent specification are **not yet mapped to a
fixture in this directory**; they are currently exercised by unit tests that construct their
inputs in code:

| Section 12 group | Test IDs | Fixture status |
|---|---|---|
| Flag validation | T-FC-001–T-FC-005 | Not fixture-backed — CLI argument parsing needs no run summary |
| Workflow discovery | T-FC-010–T-FC-013, T-FC-030 | Not yet covered — would need lock-file and API-response fixtures |
| Data sampling | T-FC-020, T-FC-021, T-FC-023 | Not yet covered — needs multi-run summary sets (window cutoff, empty sample) |
| Monte Carlo engine | T-FC-031–T-FC-034, T-FC-037–T-FC-040 | Not yet covered individually; `run_summary_minimal.json` is the shared baseline input |
| Episode grouping | T-FC-041–T-FC-044 | Not yet covered — needs run summaries sharing `headSha`/`headBranch` |
| Output format | T-FC-050–T-FC-055 | Not fixture-backed — asserted against rendered console/JSON output |

When a new fixture is added for any of the IDs above, move the row into the
"Available Additional Fixtures" table and remove it from this gap list.
