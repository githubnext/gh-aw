# Forecast Compliance Fixtures

This directory contains fixture files for bootstrapping the Section 12 compliance tests of the
[Forecast Specification](../../docs/src/content/docs/reference/forecast-specification.md).

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

### Recommended Additional Fixtures

| Fixture Name | Purpose | Test IDs |
|---|---|---|
| `run_summary_zero_et.json` | Run with missing/zero ET (artifact not downloaded) | T-FC-022 |
| `run_summary_failed.json` | Run with `conclusion: "failure"` for Bernoulli sampling | T-FC-035 |
| `run_summary_high_et.json` | Run with very high ET (≥ 1,000,000) for overflow checks | T-ET-006 |
