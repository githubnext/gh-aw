# Scheduled Workflow Concurrency Analysis

**Analysis Date**: 2025-11-13 15:57 UTC  
**Repository**: githubnext/gh-aw

## Executive Summary

This analysis examines 42 scheduled agentic workflows to identify potential resource conflicts and optimization opportunities.

### Key Findings

🔴 **CRITICAL**: Zero workflows have concurrency controls configured  
🟡 **WARNING**: Peak hours have up to 7 workflows running simultaneously  
🟢 **INFO**: Total of ~62 scheduled workflow executions per day  

### Quick Stats

| Metric | Value |
|--------|-------|
| Total Scheduled Workflows | 42 |
| Workflows with Concurrency Control | 0 |
| Peak Hour Workflow Count | 7 |
| Most Active Hours | 06:00, 09:00, 12:00 UTC |
| Workflows Running 4+ Times Daily | 4 (changeset, 3 smoke tests) |

---

## Schedule Distribution by Hour

### Hourly Workflow Matrix

The following table shows which workflows are scheduled to run at each hour (UTC):

#### 00:00 UTC - 6 workflows

- `audit-workflows` - at 0:0 UTC daily
- `changeset` - every 2 hours daily
- `safe-output-health` - at 0:0 UTC daily
- `smoke-claude` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-codex` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-copilot` - at 0:00, 6:00, 12:00, 18:00 UTC daily

#### 02:00 UTC - 4 workflows

- `changeset` - every 2 hours daily
- `daily-perf-improver` - at 2:0 UTC weekdays (Mon-Fri)
- `daily-test-improver` - at 2:0 UTC weekdays (Mon-Fri)
- `schema-consistency-checker` - at 2:0 UTC daily

#### 03:00 UTC - 2 workflows

- `developer-docs-consolidator` - at 3:17 UTC daily
- `lockfile-stats` - at 3:0 UTC daily

#### 04:00 UTC - 1 workflow

- `changeset` - every 2 hours daily

#### 06:00 UTC - 7 workflows

- `artifacts-summary` - at 6:0 UTC on Sun
- `changeset` - every 2 hours daily
- `daily-doc-updater` - at 6:0 UTC daily
- `dictation-prompt` - at 6:0 UTC on Sun
- `smoke-claude` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-codex` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-copilot` - at 0:00, 6:00, 12:00, 18:00 UTC daily

#### 08:00 UTC - 3 workflows

- `changeset` - every 2 hours daily
- `daily-code-metrics` - at 8:0 UTC daily
- `semantic-function-refactor` - at 8:0 UTC daily

#### 09:00 UTC - 7 workflows

- `copilot-pr-prompt-analysis` - at 9:0 UTC daily
- `daily-multi-device-docs-tester` - at 9:0 UTC daily
- `daily-news` - at 9:0 UTC weekdays (Mon-Fri)
- `dependabot-go-checker` - at 9:0 UTC on Mon, Wed, Fri
- `example-workflow-analyzer` - at 9:0 UTC on Mon
- `instructions-janitor` - at 9:0 UTC daily
- `static-analysis-report` - at 9:0 UTC daily

#### 10:00 UTC - 3 workflows

- `changeset` - every 2 hours daily
- `copilot-pr-nlp-analysis` - at 10:0 UTC weekdays (Mon-Fri)
- `daily-firewall-report` - at 10:0 UTC daily

#### 11:00 UTC - 1 workflow

- `typist` - at 11:0 UTC weekdays (Mon-Fri)

#### 12:00 UTC - 7 workflows

- `blog-auditor` - at 12:0 UTC on Wed
- `changeset` - every 2 hours daily
- `github-mcp-tools-report` - at 12:0 UTC on Sun
- `go-logger` - at 12:0 UTC daily
- `smoke-claude` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-codex` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-copilot` - at 0:00, 6:00, 12:00, 18:00 UTC daily

#### 13:00 UTC - 2 workflows

- `cli-consistency-checker` - at 13:0 UTC weekdays (Mon-Fri)
- `repository-quality-improver` - at 13:0 UTC weekdays (Mon-Fri)

#### 14:00 UTC - 2 workflows

- `changeset` - every 2 hours daily
- `super-linter` - at 14:0 UTC weekdays (Mon-Fri)

#### 15:00 UTC - 3 workflows

- `cli-version-checker` - at 15:0 UTC daily
- `repo-tree-map` - at 15:0 UTC on Mon
- `weekly-issue-summary` - at 15:0 UTC on Mon

#### 16:00 UTC - 3 workflows

- `changeset` - every 2 hours daily
- `copilot-session-insights` - at 16:0 UTC daily
- `daily-repo-chronicle` - at 16:0 UTC weekdays (Mon-Fri)

#### 18:00 UTC - 6 workflows

- `changeset` - every 2 hours daily
- `copilot-agent-analysis` - at 18:0 UTC daily
- `mcp-inspector` - at 18:0 UTC on Mon
- `smoke-claude` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-codex` - at 0:00, 6:00, 12:00, 18:00 UTC daily
- `smoke-copilot` - at 0:00, 6:00, 12:00, 18:00 UTC daily

#### 19:00 UTC - 1 workflow

- `prompt-clustering-analysis` - at 19:0 UTC daily

#### 20:00 UTC - 1 workflow

- `changeset` - every 2 hours daily

#### 21:00 UTC - 1 workflow

- `duplicate-code-detector` - at 21:0 UTC daily

#### 22:00 UTC - 2 workflows

- `changeset` - every 2 hours daily
- `unbloat-docs` - at 22:0 UTC daily

---

## Peak Hour Analysis

The following hours have the highest workflow density:

| Hour (UTC) | Workflow Count | Workflows |
|------------|----------------|----------|
| 09:00 | 7 | copilot-pr-prompt-analysis, daily-multi-device-docs-tester, daily-news, ... (+4) |
| 06:00 | 7 | artifacts-summary, changeset, daily-doc-updater, ... (+4) |
| 12:00 | 7 | blog-auditor, changeset, github-mcp-tools-report, ... (+4) |
| 00:00 | 6 | audit-workflows, changeset, safe-output-health, ... (+3) |
| 18:00 | 6 | changeset, copilot-agent-analysis, mcp-inspector, ... (+3) |
| 02:00 | 4 | changeset, daily-perf-improver, daily-test-improver, ... (+1) |
| 15:00 | 3 | cli-version-checker, repo-tree-map, weekly-issue-summary |
| 08:00 | 3 | changeset, daily-code-metrics, semantic-function-refactor |
| 10:00 | 3 | changeset, copilot-pr-nlp-analysis, daily-firewall-report |
| 16:00 | 3 | changeset, copilot-session-insights, daily-repo-chronicle |


### Density Heat Map

```
Hour    0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20 21 22 23
Count   6  .  4  2  1  .  7  .  3  7  3  1  7  2  2  3  3  .  6  1  1  1  2  .

Visual representation:
. = No workflows
1-2 = Low density
3-4 = Medium density
5-6 = High density
7+ = Peak density (potential conflicts)
```

---

## High-Frequency Workflows

These workflows run multiple times per day:

| Workflow | Schedule | Runs per Day |
|----------|----------|-------------|
| `changeset` | every 2 hours daily | 12 |
| `smoke-claude` | at 0:00, 6:00, 12:00, 18:00 UTC daily | 4 |
| `smoke-copilot` | at 0:00, 6:00, 12:00, 18:00 UTC daily | 4 |
| `smoke-codex` | at 0:00, 6:00, 12:00, 18:00 UTC daily | 4 |


**Total**: 24 runs per day from high-frequency workflows

---

## Concurrency Control Analysis

### Current State

**⚠️ CRITICAL FINDING**: None of the 42 scheduled workflows have concurrency controls configured.

Without concurrency controls, the following risks exist:

1. **Resource Exhaustion**: Multiple workflows running simultaneously can exhaust GitHub Actions runner capacity
2. **Rate Limiting**: Parallel API calls to the same services may trigger rate limits
3. **Cost Inefficiency**: Unnecessary parallel execution increases compute costs
4. **Execution Conflicts**: Workflows modifying shared resources can interfere with each other

### Workflows Without Concurrency Control

All 42 scheduled workflows lack concurrency configuration:

- `artifacts-summary`
- `audit-workflows`
- `blog-auditor`
- `changeset`
- `cli-consistency-checker`
- `cli-version-checker`
- `copilot-agent-analysis`
- `copilot-pr-nlp-analysis`
- `copilot-pr-prompt-analysis`
- `copilot-session-insights`
- `daily-code-metrics`
- `daily-doc-updater`
- `daily-firewall-report`
- `daily-multi-device-docs-tester`
- `daily-news`
- `daily-perf-improver`
- `daily-repo-chronicle`
- `daily-test-improver`
- `dependabot-go-checker`
- `developer-docs-consolidator`
- `dictation-prompt`
- `duplicate-code-detector`
- `example-workflow-analyzer`
- `github-mcp-tools-report`
- `go-logger`
- `instructions-janitor`
- `lockfile-stats`
- `mcp-inspector`
- `prompt-clustering-analysis`
- `repo-tree-map`
- `repository-quality-improver`
- `safe-output-health`
- `schema-consistency-checker`
- `semantic-function-refactor`
- `smoke-claude`
- `smoke-codex`
- `smoke-copilot`
- `static-analysis-report`
- `super-linter`
- `typist`
- `unbloat-docs`
- `weekly-issue-summary`


---

## Complete Workflow Schedule

Comprehensive list of all scheduled workflows with their cron expressions:

| Workflow | Cron Expression | Human-Readable Schedule |
|----------|-----------------|-------------------------|
| `artifacts-summary` | `0 6 * * 0` | at 6:0 UTC on Sun |
| `audit-workflows` | `0 0 * * *` | at 0:0 UTC daily |
| `blog-auditor` | `0 12 * * 3` | at 12:0 UTC on Wed |
| `changeset` | `0 */2 * * *` | every 2 hours daily |
| `cli-consistency-checker` | `0 13 * * 1-5` | at 13:0 UTC weekdays (Mon-Fri) |
| `cli-version-checker` | `0 15 * * *` | at 15:0 UTC daily |
| `copilot-agent-analysis` | `0 18 * * *` | at 18:0 UTC daily |
| `copilot-pr-nlp-analysis` | `0 10 * * 1-5` | at 10:0 UTC weekdays (Mon-Fri) |
| `copilot-pr-prompt-analysis` | `0 9 * * *` | at 9:0 UTC daily |
| `copilot-session-insights` | `0 16 * * *` | at 16:0 UTC daily |
| `daily-code-metrics` | `0 8 * * *` | at 8:0 UTC daily |
| `daily-doc-updater` | `0 6 * * *` | at 6:0 UTC daily |
| `daily-firewall-report` | `0 10 * * *` | at 10:0 UTC daily |
| `daily-multi-device-docs-tester` | `0 9 * * *` | at 9:0 UTC daily |
| `daily-news` | `0 9 * * 1-5` | at 9:0 UTC weekdays (Mon-Fri) |
| `daily-perf-improver` | `0 2 * * 1-5` | at 2:0 UTC weekdays (Mon-Fri) |
| `daily-repo-chronicle` | `0 16 * * 1-5` | at 16:0 UTC weekdays (Mon-Fri) |
| `daily-test-improver` | `0 2 * * 1-5` | at 2:0 UTC weekdays (Mon-Fri) |
| `dependabot-go-checker` | `0 9 * * 1,3,5` | at 9:0 UTC on Mon, Wed, Fri |
| `developer-docs-consolidator` | `17 3 * * *` | at 3:17 UTC daily |
| `dictation-prompt` | `0 6 * * 0` | at 6:0 UTC on Sun |
| `duplicate-code-detector` | `0 21 * * *` | at 21:0 UTC daily |
| `example-workflow-analyzer` | `0 9 * * 1` | at 9:0 UTC on Mon |
| `github-mcp-tools-report` | `0 12 * * 0` | at 12:0 UTC on Sun |
| `go-logger` | `0 12 * * *` | at 12:0 UTC daily |
| `instructions-janitor` | `0 9 * * *` | at 9:0 UTC daily |
| `lockfile-stats` | `0 3 * * *` | at 3:0 UTC daily |
| `mcp-inspector` | `0 18 * * 1` | at 18:0 UTC on Mon |
| `prompt-clustering-analysis` | `0 19 * * *` | at 19:0 UTC daily |
| `repo-tree-map` | `0 15 * * 1` | at 15:0 UTC on Mon |
| `repository-quality-improver` | `0 13 * * 1-5` | at 13:0 UTC weekdays (Mon-Fri) |
| `safe-output-health` | `0 0 * * *` | at 0:0 UTC daily |
| `schema-consistency-checker` | `0 2 * * *` | at 2:0 UTC daily |
| `semantic-function-refactor` | `0 8 * * *` | at 8:0 UTC daily |
| `smoke-claude` | `0 0,6,12,18 * * *` | at 0:00, 6:00, 12:00, 18:00 UTC daily |
| `smoke-codex` | `0 0,6,12,18 * * *` | at 0:00, 6:00, 12:00, 18:00 UTC daily |
| `smoke-copilot` | `0 0,6,12,18 * * *` | at 0:00, 6:00, 12:00, 18:00 UTC daily |
| `static-analysis-report` | `0 9 * * *` | at 9:0 UTC daily |
| `super-linter` | `0 14 * * 1-5` | at 14:0 UTC weekdays (Mon-Fri) |
| `typist` | `0 11 * * 1-5` | at 11:0 UTC weekdays (Mon-Fri) |
| `unbloat-docs` | `0 22 * * *` | at 22:0 UTC daily |
| `weekly-issue-summary` | `0 15 * * 1` | at 15:0 UTC on Mon |


---

## Recommendations

### 1. Implement Concurrency Controls (Priority: HIGH)

Add concurrency groups to all scheduled workflows to prevent simultaneous execution:

```yaml
concurrency:
  group: workflow-name-scheduled
  cancel-in-progress: false  # Let scheduled runs complete
```

**Recommended Groups:**
- Use workflow-specific groups: `\${{ github.workflow }}-scheduled`
- For related workflows (e.g., smoke tests), use shared groups: `smoke-tests`
- For resource-intensive workflows, use: `heavy-compute`

### 2. Redistribute Peak Hour Workflows (Priority: MEDIUM)

Current peak hours (06:00, 09:00, 12:00 UTC) have 7 workflows each. Recommendations:

**06:00 UTC (7 workflows)**
- Move `dictation-prompt` to 05:00 UTC (weekly, low resource)
- Move `artifacts-summary` to 07:00 UTC (weekly, can be delayed)

**09:00 UTC (7 workflows)**
- Move `instructions-janitor` to 10:00 UTC
- Move `dependabot-go-checker` to 08:00 UTC (runs Mon/Wed/Fri)

**12:00 UTC (7 workflows)**
- Move `blog-auditor` to 11:00 UTC (weekly, low priority)
- Move `github-mcp-tools-report` to 13:00 UTC (weekly)

### 3. Optimize High-Frequency Workflows (Priority: MEDIUM)

**changeset.md** runs every 2 hours (12 times daily):
- Consider increasing interval to every 3-4 hours
- Add concurrency: `cancel-in-progress: true` for this workflow
- Estimated savings: ~50% reduction in runs

**Smoke test workflows** (claude, codex, copilot) run 4 times daily each:
- Consider consolidating into single workflow with matrix strategy
- Reduce frequency to 3 times daily: 06:00, 12:00, 18:00 UTC
- Add shared concurrency group: `smoke-tests`

### 4. Schedule Optimization Strategy (Priority: LOW)

Distribute workflows more evenly across off-peak hours:

**Current gaps** (0-1 workflows):
- 01:00 UTC, 04:00 UTC, 05:00 UTC, 07:00 UTC, 17:00 UTC, 20:00 UTC, 23:00 UTC

**Suggested moves**:
- Move some daily 09:00 workflows to 07:00 or 05:00 UTC
- Spread weekly workflows across different time slots
- Consider time zones of primary contributors for review/approval workflows

### 5. Resource-Based Grouping (Priority: MEDIUM)

Create shared concurrency groups for workflows with similar resource profiles:

**Heavy Compute** (long-running analysis):
- `audit-workflows`, `daily-code-metrics`, `copilot-agent-analysis`
- Concurrency group: `heavy-analysis`

**Documentation Updates** (file system writes):
- `daily-doc-updater`, `unbloat-docs`, `developer-docs-consolidator`
- Concurrency group: `docs-updates`

**Quick Checks** (fast validation):
- `cli-consistency-checker`, `cli-version-checker`, `schema-consistency-checker`
- Concurrency group: `validation-checks`

---

## Implementation Guide

### Adding Concurrency Control

For each scheduled workflow, add this to the frontmatter (after the `on:` section):

```yaml
---
on:
  schedule:
    - cron: "0 9 * * *"
  workflow_dispatch:

concurrency:
  group: \${{ github.workflow }}-scheduled
  cancel-in-progress: false

permissions:
  contents: read
---
```

### Testing Changes

1. Update schedules in batches (5-10 workflows at a time)
2. Monitor GitHub Actions dashboard for conflicts
3. Adjust concurrency groups based on observed behavior
4. Document any dependencies between workflows

### Monitoring

After implementing changes:
- Track runner queue times
- Monitor for workflow delays
- Review rate limit warnings in logs
- Measure cost changes over 1-week period

---

## Appendix: Schedule Adjustment Template

Use this template for redistributing workflows:

```yaml
# BEFORE
on:
  schedule:
    - cron: "0 9 * * *"  # Peak hour

# AFTER  
on:
  schedule:
    - cron: "0 7 * * *"  # Off-peak hour

concurrency:
  group: \${{ github.workflow }}-scheduled
  cancel-in-progress: false
```

---

## Related Resources

- [GitHub Actions Concurrency Documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#concurrency)
- [GitHub Actions Usage Limits](https://docs.github.com/en/actions/learn-github-actions/usage-limits-billing-and-administration)
- Issue: githubnext/gh-aw#3853 (CI/CD Workflow Optimization)

---

**Generated by**: Scheduled Workflow Analysis Tool  
**Last Updated**: 2025-11-13 15:57 UTC
