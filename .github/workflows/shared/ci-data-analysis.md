---
# CI Data Analysis
# Shared module for analyzing CI run data
#
# Usage:
#   imports:
#     - shared/ci-data-analysis.md
#
# This import provides:
# - Pre-download CI runs and artifacts
# - Build and test the project
# - Collect performance metrics

imports:
  - shared/jqschema.md

tools:
  cache-memory: true
  bash: ["*"]

steps:
  - name: Download CI workflow runs from last 7 days
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Download workflow runs for split CI workflows (ci, cgo, cjs)
      gh run list --repo "$GITHUB_REPOSITORY" --workflow=ci.yml --limit 30 --json databaseId,status,conclusion,createdAt,updatedAt,displayTitle,headBranch,event,url,workflowDatabaseId,number > /tmp/ci-runs-ci.json
      gh run list --repo "$GITHUB_REPOSITORY" --workflow=cgo.yml --limit 30 --json databaseId,status,conclusion,createdAt,updatedAt,displayTitle,headBranch,event,url,workflowDatabaseId,number > /tmp/ci-runs-cgo.json
      gh run list --repo "$GITHUB_REPOSITORY" --workflow=cjs.yml --limit 30 --json databaseId,status,conclusion,createdAt,updatedAt,displayTitle,headBranch,event,url,workflowDatabaseId,number > /tmp/ci-runs-cjs.json
      jq -s 'add | sort_by(.createdAt) | reverse | .[0:60]' /tmp/ci-runs-ci.json /tmp/ci-runs-cgo.json /tmp/ci-runs-cjs.json > /tmp/ci-runs.json
      
      # Create directory for artifacts
      mkdir -p /tmp/ci-artifacts
      
      # Download artifacts from recent successful runs across split workflows
      echo "Downloading artifacts from recent CI/cgo/cjs runs..."
      {
        gh run list --repo "$GITHUB_REPOSITORY" --workflow=ci.yml --status success --limit 2 --json databaseId
        gh run list --repo "$GITHUB_REPOSITORY" --workflow=cgo.yml --status success --limit 2 --json databaseId
        gh run list --repo "$GITHUB_REPOSITORY" --workflow=cjs.yml --status success --limit 2 --json databaseId
      } | jq -s 'add | .[].databaseId' -r | while read -r run_id; do
        echo "Processing run $run_id"
        gh run download "$run_id" --repo "$GITHUB_REPOSITORY" --dir "/tmp/ci-artifacts/$run_id" 2>/dev/null || echo "No artifacts for run $run_id"
      done
      
      echo "CI runs data saved to /tmp/ci-runs.json"
      echo "Artifacts saved to /tmp/ci-artifacts/"
      
  - name: Build CI summary for optimization analysis
    run: |
      jq '
      def safe_duration:
        if (.createdAt and .updatedAt) then
          ((.updatedAt | fromdateiso8601) - (.createdAt | fromdateiso8601))
        else null end;
      {
        generated_at: now | todateiso8601,
        total_runs: length,
        status_counts: (group_by(.status) | map({status: .[0].status, count: length})),
        conclusion_counts: (map(select(.conclusion != null)) | group_by(.conclusion) | map({conclusion: .[0].conclusion, count: length})),
        branch_counts: (group_by(.headBranch) | map({branch: .[0].headBranch, count: length}) | sort_by(-.count) | .[0:10]),
        avg_duration_seconds: ([.[] | safe_duration | select(. != null)] | if length > 0 then (add / length) else null end),
        top_recent_failures: ([.[] | select(.conclusion == "failure" or .conclusion == "cancelled") | {id: .databaseId, run_number: .number, title: .displayTitle, branch: .headBranch, event: .event, url: .url, updated_at: .updatedAt}] | sort_by(.updated_at) | reverse | .[0:10])
      }' /tmp/ci-runs.json > /tmp/ci-summary.json

      echo "## CI Summary" >> "$GITHUB_STEP_SUMMARY"
      jq -r '"- runs analyzed: \(.total_runs)\n- avg duration (sec): \(.avg_duration_seconds // "n/a")\n- recent failure records: \(.top_recent_failures | length)"' /tmp/ci-summary.json >> "$GITHUB_STEP_SUMMARY"
  
  - name: Setup Node.js
    uses: actions/setup-node@v6.4.0
    with:
      node-version: "24"
      cache: npm
      cache-dependency-path: actions/setup/js/package-lock.json
  
  - name: Setup Go
    uses: actions/setup-go@v6.4.0
    with:
      go-version-file: go.mod
      cache: true
  
  - name: Install development dependencies
    run: make deps-dev
  
  - name: Run linter
    run: make lint
  
  - name: Lint error messages
    run: make lint-errors
  
  - name: Install npm dependencies
    run: npm ci
    working-directory: ./actions/setup/js
  
  - name: Build code
    run: make build
  
  - name: Recompile workflows
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: make recompile
  
  - name: Run unit tests
    continue-on-error: true
    run: |
      mkdir -p /tmp/gh-aw
      go test -v -json -count=1 -timeout=3m -tags '!integration' -run='^Test' ./... | tee /tmp/gh-aw/test-results.json
---

# CI Data Analysis

Pre-downloaded CI run data and artifacts are available for analysis:

## Available Data

1. **CI Runs**: `/tmp/ci-runs.json`
   - Last 60 workflow runs with status, timing, and metadata from `ci.yml`, `cgo.yml`, and `cjs.yml`

2. **CI Summary**: `/tmp/ci-summary.json`
   - Pre-computed totals, failure patterns, branch distribution, and average duration
    
3. **Artifacts**: `/tmp/ci-artifacts/`
   - Coverage reports and benchmark results from recent successful runs
   - **Fuzz test results**: `*/fuzz-results/*.txt` - Output from fuzz tests
   - **Fuzz corpus data**: `*/fuzz-results/corpus/*` - Input corpus for each fuzz test
   
4. **CI Configuration**:
   - `.github/workflows/ci.yml`
   - `.github/workflows/cgo.yml`
   - `.github/workflows/cjs.yml`
   
5. **Cache Memory**: `/tmp/cache-memory/`
   - Historical analysis data from previous runs
   
6. **Test Results**: `/tmp/gh-aw/test-results.json`
   - JSON output from Go unit tests with performance and timing data

## Test Case Locations

Go test cases are located throughout the repository:
- **Command tests**: `./cmd/gh-aw/*_test.go`
- **Workflow tests**: `./pkg/workflow/*_test.go`
- **CLI tests**: `./pkg/cli/*_test.go`
- **Parser tests**: `./pkg/parser/*_test.go`
- **Campaign tests**: `./pkg/campaign/*_test.go`
- **Other package tests**: Various `./pkg/*/test.go` files

## Environment Setup

The workflow has already completed:
- ✅ **Linting**: Dev dependencies installed, linters run successfully
- ✅ **Building**: Code built with `make build`, lock files compiled with `make recompile`
- ✅ **Testing**: Unit tests run (with performance data collected in JSON format)

This means you can:
- Make changes to code or configuration files
- Validate changes immediately by running `make lint`, `make build`, or `make test-unit`
- Ensure proposed optimizations don't break functionality before creating a PR

## Analyzing Run Data

Start with the pre-computed summary:

```bash
cat /tmp/ci-summary.json | jq .
```

Only use raw run data for deeper validation:

```bash
# Analyze run data
cat /tmp/ci-runs.json | jq '
{
  total_runs: length,
  by_status: group_by(.status) | map({status: .[0].status, count: length}),
  by_conclusion: group_by(.conclusion) | map({conclusion: .[0].conclusion, count: length}),
  by_branch: group_by(.headBranch) | map({branch: .[0].headBranch, count: length}),
  by_event: group_by(.event) | map({event: .[0].event, count: length})
}'
```

**Metrics to extract:**
- Success rate per job
- Average duration per job
- Failure patterns (which jobs fail most often)
- Cache hit rates from step summaries
- Resource usage patterns

## Review Artifacts

Examine downloaded artifacts for insights:

```bash
# List downloaded artifacts
find /tmp/ci-artifacts -type f -name "*.txt" -o -name "*.html" -o -name "*.json"

# Analyze coverage reports if available
# Check benchmark results for performance trends
```

## Historical Context

Check cache memory for previous analyses:

```bash
# Read previous optimization recommendations
if [ -f /tmp/cache-memory/ci-coach/last-analysis.json ]; then
  cat /tmp/cache-memory/ci-coach/last-analysis.json
fi

# Check if previous recommendations were implemented
# Compare current metrics with historical baselines
```
