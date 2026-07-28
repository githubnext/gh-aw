---
name: Daily Spending Forecast
description: Forecasts agentic workflow spending, reviews data quality, and publishes a daily report with charts
emoji: "📈"
on:
  schedule: daily around 9:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  copilot-requests: write
strict: true
tools:
  github:
    mode: gh-proxy
  agentic-workflows: true
steps:
  - name: Build gh-aw from source
    run: |
      set -euo pipefail
      make build
      "$GITHUB_WORKSPACE/gh-aw" --version
  - name: Prefetch forecast usage artifacts
    continue-on-error: true
    env:
      REPOSITORY: ${{ github.repository }}
      GH_TOKEN: ${{ github.token }}
    run: |
      # Download usage artifacts for the last 30 days in parallel so the main
      # forecast step reads from the local cache and produces output quickly.
      DEBUG='*' "$GITHUB_WORKSPACE/gh-aw" forecast \
        --repo "$REPOSITORY" \
        --days 30 \
        --sample 100 \
        --concurrency 8 \
        --timeout 25 \
        --verbose \
        > /dev/null 2>&1 || true
  - name: Run spending forecast
    id: spending_forecast
    continue-on-error: true
    env:
      REPOSITORY: ${{ github.repository }}
      GH_TOKEN: ${{ github.token }}
    run: |
      set -uo pipefail
      output_dir="/tmp/gh-aw/agent/spending-forecast"
      mkdir -p "$output_dir"

      set +e
      DEBUG='*' "$GITHUB_WORKSPACE/gh-aw" forecast \
        --repo "$REPOSITORY" \
        --days 30 \
        --period month \
        --sample 100 \
        --concurrency 8 \
        --timeout 10 \
        --verbose \
        --json \
        > >(tee "$output_dir/forecast.json") \
        2> >(tee "$output_dir/forecast.stderr.log" >&2)
      exit_code=$?
      wait
      set -e

      {
        printf 'exit_code=%s\n' "$exit_code"
        printf 'repository=%s\n' "$REPOSITORY"
        printf 'generated_at=%s\n' "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
      } > "$output_dir/forecast-metadata.txt"

      {
        echo "===== STDERR ====="
        cat "$output_dir/forecast.stderr.log"
        echo
        echo "===== STDOUT ====="
        cat "$output_dir/forecast.json"
      } > "$output_dir/forecast.full.log"
post-steps:
  - name: Upload spending forecast logs and report
    if: always()
    uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
    with:
      name: spending-forecast-${{ github.run_id }}
      path: /tmp/gh-aw/agent/spending-forecast/
      retention-days: 30
      if-no-files-found: warn
safe-outputs:
  create-issue:
    title-prefix: "[spending-forecast] "
    labels: [agentic-workflows]
    close-older-issues: true
    expires: 7d
    max: 1
  mentions: false
  allowed-github-references: []
timeout-minutes: 45
sandbox:
  agent:
    sudo: false
---

# Daily Spending Forecast

Analyze the prepared `gh aw forecast` output and publish a daily spending forecast for
`${{ github.repository }}`.

The complete initial command output is in
`/tmp/gh-aw/agent/spending-forecast/`:

- `forecast.json` — machine-readable forecast output
- `forecast.stderr.log` — verbose and debug diagnostics
- `forecast.full.log` — complete stderr and stdout transcript
- `forecast-metadata.txt` — command exit code and collection context

## Analysis

1. Validate that `forecast.json` parses and reconcile its workflow totals, sampled run
   counts, history windows, run samples, success rates, and P10/P50/P90 projections.
2. Review the data for inaccuracies. Flag zero or missing AIC values, sparse or stale
   samples, outliers, inconsistent date windows, implausible run frequencies, missing
   workflows, and confidence intervals that are too broad to support a reliable budget.
   Do not invent missing values.
3. If the prepared output is incomplete or suspicious, rerun
   `$GITHUB_WORKSPACE/gh-aw forecast` with targeted arguments (including `--eval` when
   backtesting would clarify accuracy). If sample collection still fails — especially
   when `sampled_runs` is zero for all workflows or artifact downloads fail — you MUST
   use the `agentic-workflows` MCP server to inspect recent runs and usage artifacts
   and derive observed/projected AIC directly from that evidence.
   Preserve any additional command output in the spending forecast directory so it is
   included in the artifact. Limit follow-up to the evidence needed to resolve or
   document the discrepancy.
4. Calculate historical spending from `run_samples[].aic` and clearly distinguish
   observed spending from projected spending.

## Report

Write the final GitHub-flavored markdown report to
`/tmp/gh-aw/agent/spending-forecast/report.md`, then create one issue titled
`Daily spending forecast - YYYY-MM-DD` with the same report body.

Use `###` headings and include:

- a concise executive summary with total observed AIC and weekly/monthly P10, P50, and
  P90 forecast totals;
- a compact ASCII spending trend chart and forecast confidence chart in fenced code
  blocks, each under 80 columns with spaces rather than tabs;
- a workflow table showing sample count, observed AIC, P50/P95 per-run AIC, projected
  weekly/monthly AIC, success rate, and confidence range;
- a visible data-quality and accuracy section that explains every detected discrepancy,
  the likely forecast impact, and whether follow-up evidence resolved it;
- assumptions, forecast date, 30-day history window, and a link to
  `[§${{ github.run_id }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})`.

Wrap verbose per-workflow evidence in `<details><summary>...</summary>`. If the initial
forecast failed and follow-up cannot recover it, still create an operational report from
the captured diagnostics rather than presenting fabricated projections.
