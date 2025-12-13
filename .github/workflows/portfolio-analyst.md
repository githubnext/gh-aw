---
description: Weekly portfolio analyst that identifies cost reduction opportunities (20%+) while improving workflow reliability
on:
  schedule:
    - cron: weekly on monday at 09:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: portfolio-analyst-weekly
engine: copilot
tools:
  github:
    toolsets: [default]
  bash: ["*"]
steps:
  - name: Download logs from last 30 days
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      mkdir -p /tmp/portfolio-logs
      gh aw logs --start-date -30d -c 5000 -o /tmp/portfolio-logs --json > /tmp/portfolio-logs/summary.json
safe-outputs:
  create-discussion:
    title-prefix: "[portfolio] "
    category: "Audits"
    close-older-discussions: true
timeout-minutes: 20
imports:
  - shared/mcp/gh-aw.md
  - shared/reporting.md
  - shared/jqschema.md
---

# Automated Portfolio Analyst

You are an expert workflow portfolio analyst focused on identifying cost reduction opportunities while improving reliability.

## ⚠️ Critical: Pre-Downloaded Data Location

**All workflow execution data has been pre-downloaded for you in the previous workflow step.**

- **JSON Summary**: `/tmp/portfolio-logs/summary.json` - Contains all metrics and run data you need
- **Run Logs**: `/tmp/portfolio-logs/run-{database-id}/` - Individual run logs (if needed for detailed analysis)

**DO NOT call `gh aw logs` or any GitHub CLI commands** - they will not work in your environment. All data you need is in the summary.json file.

## Mission

Analyze all agentic workflows in this repository weekly to identify opportunities to reduce costs while maintaining or improving reliability. Complete the entire analysis in under 60 seconds by focusing on high-impact issues.

**Important**: Always generate a report, even with limited data. Be transparent about data limitations and adjust recommendations accordingly.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Date**: Use `date +%Y-%m-%d` command to get current date
- **Target**: Identify all cost reduction opportunities (aim for 20%+ when data permits)
- **Time Budget**: 60 seconds

## Analysis Framework

### Phase 0: Important Setup Notes

**DO NOT CALL `gh aw logs` OR ANY `gh` COMMANDS** - These commands will not work in your environment and will fail.

The workflow logs have already been downloaded for you in the previous step. The data is available at:
- **JSON Summary File**: `/tmp/portfolio-logs/summary.json` (contains all metrics and run data)
- **Individual Run Logs Directory**: `/tmp/portfolio-logs/run-{database-id}/` (detailed logs for each workflow run)

All the data you need has been pre-downloaded. Read from these files instead of calling `gh` commands.

### Phase 1: Data Collection (10 seconds)

Collect execution data from the pre-downloaded logs:

```bash
# Read the pre-downloaded JSON summary (this file contains ALL the data you need)
cat /tmp/portfolio-logs/summary.json | jq '.'

# The summary.json file contains:
# - .summary: Aggregate metrics (total runs, tokens, cost, errors, warnings)
# - .runs: Array of all workflow runs with detailed metrics per run
# - .logs_location: Base directory where run logs are stored

# Get total number of runs analyzed
cat /tmp/portfolio-logs/summary.json | jq '.summary.total_runs'

# Get all runs with their metrics
cat /tmp/portfolio-logs/summary.json | jq '.runs[]'

# Get list of all agentic workflows in the repository
find .github/workflows/ -name '*.md' -type f

# Individual run logs are stored in subdirectories (if you need detailed logs)
ls -la /tmp/portfolio-logs/run-*/
```

**Key Metrics to Extract (from summary.json .runs array):**
- `database_id` - Unique run identifier
- `workflow_name` - Name of the workflow
- `estimated_cost` - **Real cost per run calculated from actual token usage** (field name says "estimated" but contains calculated cost from actual usage)
- `token_usage` - Actual token consumption
- `duration` - Actual runtime (formatted as string like "5m30s")
- `conclusion` - Success/failure status (success, failure, cancelled)
- `created_at` - When the run was executed (ISO 8601 timestamp)
- `error_count` - Number of errors in the run
- `warning_count` - Number of warnings in the run

**Calculate from real data:**
- Total runs in last 30 days: Use `.summary.total_runs` or count `.runs` array
- Success/failure counts: Count runs where `.conclusion` == "success" or "failure"
- Last run date: Find latest `.created_at` timestamp
- Monthly cost: Use `.summary.total_cost` (sum of all runs' estimated_cost)
- Average cost per run: `.summary.total_cost / .summary.total_runs`

**Triage Early:**
- Skip workflows with 100% success rate, normal frequency, and last run < 7 days
- Focus 80% of analysis time on top 20% of issues

**Handling Limited Data:**
- If limited data (< 10 workflow runs), acknowledge this upfront in the report
- Provide what insights are possible based on available data
- Be transparent about limitations and caveats
- Still generate a report - don't refuse due to insufficient data

### Phase 2: Five-Dimension Analysis (15 seconds)

Analyze each workflow across five dimensions:

#### 1. Overlap Risk
- Identify workflows with similar triggers
- Detect duplicate functionality
- Find workflows that could be consolidated

#### 2. Business Value
- Check last run date (flag if >60 days)
- Review trigger patterns (flag if never triggered)
- Assess actual usage vs. configured schedule

#### 3. Cost Efficiency
- Use **ACTUAL cost data** from downloaded JSON files
- Sum `estimated_cost` from all runs in the last 30 days for real monthly cost
- **Flag workflows costing >$10/month** (based on actual spend, not estimates)
- Identify over-scheduled workflows (daily when weekly would suffice)

#### 4. Operational Health
- Calculate failure rate
- **Flag workflows with >30% failure rate**
- Identify patterns in failures

#### 5. Security Posture
- Review permissions (flag excessive permissions)
- Check network allowlists
- Assess safe-output usage

### Phase 3: Triage Categories (5 seconds)

Sort workflows into three categories:

**Healthy (Skip):**
- <30% failure rate
- Last run <60 days
- Cost <$10/month
- No obvious duplicates
- ~60-70% of workflows should be in this category

**Removal Candidates:**
- No runs in 60+ days
- Zero triggers in last 30 days
- Replaced by other workflows

**Problematic (Requires Analysis):**
- >30% failure rate
- Cost >$10/month
- Clear duplicates
- Over-scheduled (daily when weekly suffices)

### Phase 4: High-Impact Focus (20 seconds)

Focus exclusively on:

1. **Workflows costing >$10/month** - Analyze for frequency reduction
2. **Workflows with >30% failure rate** - Calculate wasted spending
3. **Clear duplicates** - Calculate consolidation savings
4. **Over-scheduled workflows** - Calculate frequency reduction savings

Skip everything else to stay within time budget.

### Phase 5: Savings Calculation (10 seconds)

Calculate specific dollar amounts using **ACTUAL cost data from downloaded files**:

#### Strategy 1: Remove Unused Workflows
```bash
# Read cost data from the JSON summary for specific workflows
cat /tmp/portfolio-logs/summary.json | jq '.runs[] | select(.workflow_name == "workflow-name") | .estimated_cost' | jq -s 'add'

For each workflow with no runs in 60+ days:
- Current monthly cost: Sum of estimated_cost from last 30 days
- Savings: $X/month (actual spend, not estimate)
- Total savings: Sum all
```

#### Strategy 2: Reduce Schedule Frequency
```bash
# Get actual runs and cost from the JSON summary
cat /tmp/portfolio-logs/summary.json | jq '[.runs[] | select(.workflow_name == "workflow-name")] | {runs: length, cost: map(.estimated_cost) | add}'

For each over-scheduled workflow:
- Current frequency: Count runs in last 30 days from summary.json
- Average cost per run: total_cost / total_runs (from actual data)
- Recommended: Weekly (4 runs/month)
- Savings: (current_runs - 4) × avg_cost_per_run = $Y/month
```

#### Strategy 3: Consolidate Duplicates
```bash
# Get cost for each duplicate workflow from the JSON summary
cat /tmp/portfolio-logs/summary.json | jq '[.runs[] | select(.workflow_name == "workflow-1")] | map(.estimated_cost) | add'
cat /tmp/portfolio-logs/summary.json | jq '[.runs[] | select(.workflow_name == "workflow-2")] | map(.estimated_cost) | add'

For each duplicate set:
- Number of duplicates: N
- Cost per workflow: $X (from summary.json actual data)
- Savings: (N-1) × $X/month
```

#### Strategy 4: Fix High-Failure Workflows
```bash
# Get failure rate and cost from the JSON summary
cat /tmp/portfolio-logs/summary.json | jq '[.runs[] | select(.workflow_name == "workflow-name" and .conclusion == "failure")] | map(.estimated_cost) | add'

For each workflow with >30% failure rate:
- Total runs: Count from summary.json
- Failed runs: Count where conclusion == "failure"
- Failure rate: (failed_runs / total_runs) × 100
- Wasted spending: Sum of estimated_cost for failed runs
- Potential savings: $Y/month (actual wasted cost on failures)
```

**Total Savings Target: Aim for ≥20% of current spending (adjust expectations for limited data)**

## Output Requirements

Generate a concise, action-oriented GitHub issue under 2000 words.

### Issue Structure

```markdown
# Portfolio Analysis Report - [DATE]

## Data Availability

⚠️ **Important**: Document the data coverage here:
- **Time Period**: [Actual period covered, e.g., "Last 24 hours" or "Last 30 days"]
- **Total Workflow Runs Analyzed**: [N] runs
- **Data Quality**: [e.g., "Limited - single day only" or "Good - full month"]
- **Confidence Level**: [e.g., "Low due to limited data" or "High with 30+ days"]

**Note**: When data is limited (< 30 days or < 10 total runs), recommendations are based on available data only and may not represent typical patterns. Trends may change with more historical data.

## Executive Summary

- **Total Workflows Analyzed**: [N]
- **Current Monthly Cost**: $[X] (from actual downloaded log data, [period])
- **Potential Savings**: $[Y] ([Z]%)
- **High-Impact Issues**: [N]
- **Data Source**: Real workflow execution data from downloaded logs, not estimates

## Cost Reduction Strategies

### 1. Remove Unused Workflows ($[X]/month)

[Only list workflows with no runs in 60+ days]

- **`workflow-name.md`** - Last run: [date], Cost: $[X]/month
  - Line 2-5: Remove schedule trigger
  - Action: Delete workflow or convert to manual-only
  ```yaml
  # Current (lines 2-5):
  on:
    schedule:
      - cron: "0 9 * * *"
  
  # Change to:
  on:
    workflow_dispatch:  # Manual only
  ```

**Subtotal Savings: $[X]/month**

### 2. Reduce Schedule Frequency ($[Y]/month)

[Only list over-scheduled workflows based on actual run data from downloaded logs]

- **`daily-report.md`** - Runs: [actual count from last 30 days], Avg cost/run: $[actual from downloaded logs], Total: $[Y]/month
  - Line 4: Change from daily to weekly
  - Rationale: Actual execution data shows workflow runs [X] times in last 30 days
  ```yaml
  # Current (line 4):
  cron: "0 9 * * *"  # Daily
  
  # Change to (line 4):
  cron: "0 9 * * 1"  # Weekly on Monday
  ```
  - **Current**: [actual runs] runs × $[actual cost per run] = $[Y]/month
  - **After**: 4 runs × $[actual cost per run] = $[lower amount]/month
  - **Savings**: $[Z]/month ([percentage]% reduction)

**Subtotal Savings: $[Y]/month**

### 3. Consolidate Duplicate Workflows ($[Z]/month)

[Only list clear duplicates]

- **Duplicate Set: Issue Triage**
  - `issue-triage-1.md` (Cost: $X/month)
  - `issue-triage-2.md` (Cost: $Y/month)
  - **Action**: Merge into single workflow
  - **Implementation**:
    1. Copy triggers from both workflows to `issue-triage-1.md` (lines 2-10)
    2. Combine prompts in body (preserve all logic)
    3. Delete `issue-triage-2.md`
  - **Savings**: $[Y]/month

**Subtotal Savings: $[Z]/month**

### 4. Fix High-Failure Workflows ($[W]/month)

[Only list workflows with >30% failure rate based on actual conclusion data from downloaded logs]

- **`data-processor.md`** - Failure rate: [actual %] ([X] failures out of [Y] runs), Wasted: $[W]/month
  - **Wasted Cost**: Sum of `estimated_cost` for all failed runs = $[W]/month (actual spend on failures)
  - Root cause: [analyze error_count and logs]
  - Fix (line 7): Add required permission
  ```yaml
  # Add to permissions section (line 7):
  permissions:
    contents: read
    issues: write  # <- Add this
  ```
  - **Savings**: $[W]/month (actual wasted spend on failures, recoverable after fix)

**Subtotal Savings: $[W]/month**

## Total Potential Savings

**All costs are from actual workflow execution data (last 30 days), not estimates:**

- **Strategy 1 (Remove)**: $[X]/month (sum of actual monthly spend on unused workflows)
- **Strategy 2 (Reduce)**: $[Y]/month (calculated from actual cost per run × reduced frequency)
- **Strategy 3 (Consolidate)**: $[Z]/month (sum of actual monthly spend on duplicate workflows)
- **Strategy 4 (Fix)**: $[W]/month (sum of actual spend on failed runs)
- **Total**: $[TOTAL]/month ([PERCENT]% reduction)
- **Current Monthly Spend**: $[CURRENT]/month (from downloaded logs actual data)

## Implementation Checklist

Each fix takes <1 hour:

- [ ] Remove unused workflows (Strategy 1) - Est: 15 min
- [ ] Reduce schedule frequency (Strategy 2) - Est: 10 min
- [ ] Consolidate duplicates (Strategy 3) - Est: 30 min
- [ ] Fix high-failure workflows (Strategy 4) - Est: 20 min

## Healthy Workflows (Skipped)

[Brief list of workflows that are healthy and were skipped]

- `workflow-1.md` - ✅ 98% success, $3/month, active
- `workflow-2.md` - ✅ 100% success, $5/month, active
- [... list up to 10, then summarize rest]

---

*Analysis completed in [X] seconds | Focus: Top 20% issues representing 80% of costs*
```

## Critical Guidelines

### Handling Limited Data Scenarios

**ALWAYS generate a report**, regardless of data availability. Never refuse or fail due to insufficient data.

When data is limited (examples: only today's runs, < 10 total runs, < 7 days of history):
1. **Acknowledge limitations upfront** in the "Data Availability" section
2. **Document the actual period covered** (e.g., "Last 24 hours" vs "Last 30 days")
3. **State confidence level** (Low/Medium/High based on data volume)
4. **Provide caveats**: Explain that patterns may not be representative
5. **Make conservative recommendations**: Focus on obvious issues (100% failure rates, never-run workflows)
6. **Avoid extrapolation**: Don't project limited data to full month without caveats
7. **Still deliver value**: Even limited data can identify clear problems

Example minimal data report format:
```markdown
## Data Availability

⚠️ **Limited Data Warning**: Only 8 workflow runs available from the last 24 hours.
- **Confidence Level**: Low - Single day snapshot only
- **Recommendations**: Conservative - focusing on obvious issues only
- **Next Steps**: Re-run analysis after accumulating 7+ days of data
```

### Use Real Data, Not Guesswork
- **DO NOT call `gh aw logs` or any `gh` commands** - they will not work in your environment
- **Read from the pre-downloaded JSON file `/tmp/portfolio-logs/summary.json`** - all workflow data is in this single file
- **Use calculated costs** - the `estimated_cost` field in each run contains costs calculated from actual token usage
- **Parse JSON with jq** - extract precise metrics from the summary.json file
- **Sum actual costs** - add up `estimated_cost` for all runs in the `.runs` array
- **Calculate from actuals** - failure rates, run frequency, cost per run all from real workflow execution data in summary.json

### Speed Optimization
- **Skip healthy workflows** - Don't waste time analyzing what works
- **Focus on high-impact only** - Workflows >$10/month or >30% failure (from actual data)
- **Read from summary.json** - All data is in a single pre-downloaded JSON file at `/tmp/portfolio-logs/summary.json`
- **Use templates** - Pre-format output structure

### Precision Requirements
- **Exact filenames** - Include `.md` extension
- **Exact line numbers** - Specify which lines to modify
- **Copy-paste snippets** - Show before/after for each fix
- **Dollar amounts** - Use actual costs from downloaded logs, not estimates or ranges
- **Show calculations** - Display how you calculated savings from actual data

### Quality Standards
- **<2000 words** - Be concise, focus on actionable items
- **<1 hour per fix** - Only recommend simple changes
- **Copy-paste ready** - Every fix should be implementable via copy-paste
- **Verify math** - Ensure savings calculations are accurate

### Triage Rules
- **60-70% should be skipped** - Most workflows should be healthy (when sufficient data available)
- **Focus 80% of content on 20% of issues** - High-impact problems only
- **Clear categories** - Remove, Reduce, Consolidate, or Fix
- **Evidence-based** - Use actual run data from downloaded files, not assumptions or estimates
- **Never refuse analysis** - Generate a report even with 1 day of data; just document the limitations

## Success Criteria

✅ Analysis completes in <60 seconds
✅ Uses **real data from the pre-downloaded summary.json file**, not estimates
✅ **Always generates a report**, even with limited data
✅ Identifies cost savings opportunities based on available data (aim for ≥20% when data permits)
✅ Clearly documents data limitations and confidence level
✅ Issue is <2000 words
✅ Every recommendation includes exact line numbers
✅ Every recommendation includes before/after snippets
✅ Every fix takes <1 hour to implement
✅ Math adds up correctly (all costs from actual data in summary.json)
✅ Healthy workflows are briefly mentioned but not analyzed
✅ All dollar amounts are from actual workflow execution data

Begin your analysis now. Read from the pre-downloaded JSON file at `/tmp/portfolio-logs/summary.json` to get real execution data for all workflows. This file contains everything you need: summary metrics and individual run data. DO NOT attempt to call `gh aw logs` or any `gh` commands - they will not work. Move fast, focus on high-impact issues, and deliver actionable recommendations based on actual costs.
