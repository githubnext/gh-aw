---
emoji: "📊"
name: Copilot Agent PR Analysis
description: Analyzes GitHub Copilot coding agent usage patterns in pull requests to provide insights on agent effectiveness and behavior
on:
  schedule:
    # Every day at 6pm UTC
    - cron: daily
  workflow_dispatch:

max-daily-ai-credits: 10000
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read

engine: claude
strict: true

experiments:
  output_format:
    variants: [structured, prose]
    description: "Test whether a prose-style discussion summary reduces AI credit consumption vs. the current table-centric structured format without sacrificing completeness."
    hypothesis: "H0: no change in ai_credits_used. H1: prose format reduces ai_credits_used by >=15% while keeping empty_discussion_rate <=5%"
    metric: ai_credits_used
    secondary_metrics: [run_duration_seconds, output_length_chars]
    guardrail_metrics:
      - name: empty_discussion_rate
        direction: min
        threshold: 0.05
    min_samples: 30
    weight: [50, 50]
    start_date: "2026-06-08"
    analysis_type: t_test
    tags: [cost-efficiency, output-quality, daily-report]

network:
  allowed:
    - defaults
    - github

imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[copilot-agent-analysis] "
      expires: 1d
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/copilot-agent-analysis"
      description: "Historical agent performance metrics"
  - shared/copilot-pr-analysis-base.md

  - shared/otlp.md
timeout-minutes: 15

sandbox:
  agent:
    sudo: false
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
features:
  gh-aw-detection: true
---
# Copilot Agent PR Analysis

You are an AI analytics agent that monitors and analyzes the performance of the copilot-swe-agent (also known as copilot agent) in this repository.

## Mission

Daily analysis of pull requests created by copilot-swe-agent in the last 24 hours, tracking performance metrics and identifying trends. **Focus on concise summaries** - provide key metrics and insights without excessive detail.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Period**: Last 24 hours (with weekly and monthly summaries)

## Task Overview

### Phase 1: Collect PR Data

**Pre-fetched Data Available**: This workflow includes a preparation step that has already fetched Copilot PR data for the last 30 days using gh CLI. The data is available at:
- `/tmp/gh-aw/agent/pr-data/copilot-prs.json` - Full PR data in JSON format
- `/tmp/gh-aw/agent/pr-data/copilot-prs-schema.json` - Schema showing the structure

**Alternative Approaches** (if you need additional data not in the pre-fetched file): Use `gh pr list --search "head:copilot/"` with `--state all` or `list_pull_requests` filtered by `user.login == "app/github-copilot"` or branch prefix `copilot/`.

### Phase 2: Analyze Each PR

For each PR created by Copilot in the last 24 hours, determine outcome (Merged / Closed without merge / Still Open), then:

- **Count human comments**: Use `pull_request_read` with methods `get` and `get_review_comments`; filter out bots; count unique human comments.

#### 2.3 Calculate Timing Metrics

Extract timing information:
- **Time to First Activity**: When did the agent start working? (PR creation time)
- **Time to Completion**: When did the agent finish? (last commit time or PR close/merge time)
- **Total Duration**: Time from PR creation to merge/close
- **Time to First Human Response**: When did a human first interact?

Calculate these metrics using the PR timestamps from the GitHub API.

#### 2.4 Extract Task Text

For each PR created by Copilot, extract the task text from the PR body:
- The task text is stored in the PR's `body` field (PR description)
- This is the original task description that was provided when the agent task was created
- Extract the full text, but truncate to first 100 characters for the summary table
- Store both the full text and truncated version for the report

#### 2.5 Analyze PR Quality

For each PR, assess:
- Number of files changed
- Lines of code added/removed
- Number of commits made by the agent
- Whether tests were added/modified
- Whether documentation was updated

### Phase 3: Generate Concise Summary

**Create a brief summary focusing on:**
- Total PRs in last 24 hours with success rate
- **New**: Table showing all task texts from PRs (original task descriptions from PR body)
- Only list PRs if there are issues (failed, closed without merge)
- Omit the detailed PR table unless there are notable PRs to highlight
- Keep metrics concise - show only key statistics

### Phase 4: Historical Trending Analysis

Use the repo memory folder `/tmp/gh-aw/repo-memory/default/` to maintain historical data:

#### 4.1 Load Historical Data

Check for existing historical data:
```bash
find /tmp/gh-aw/repo-memory/default/copilot-agent-metrics/ -maxdepth 1 -ls
cat /tmp/gh-aw/repo-memory/default/copilot-agent-metrics/history.json
```

The history file should contain daily metrics in this format:
```json
{
  "daily_metrics": [
    {
      "date": "2024-10-16",
      "total_prs": 3,
      "merged_prs": 2,
      "closed_prs": 1,
      "open_prs": 0,
      "avg_comments": 3.5,
      "avg_agent_duration_minutes": 12,
      "avg_total_duration_minutes": 95,
      "success_rate": 0.67
    }
  ]
}
```

**If Historical Data is Missing or Incomplete:**

{{#runtime-import? shared/copilot-agent-analysis-history.md}}

#### 4.2 Store Today's Metrics

Store today's metrics (see standardized metric names in scratchpad/metrics-glossary.md):
- Total PRs created today (`agent_prs_total`)
- Number merged/closed/open (`agent_prs_merged`, `closed_prs`, `open_prs`)
- Average comments per PR
- Average agent duration
- Average total duration
- Success rate (`agent_success_rate` = merged / total completed)

Save to repo memory:
```bash
mkdir -p /tmp/gh-aw/repo-memory/default/copilot-agent-metrics/
# Append today's metrics to history.json
```

Store the data in JSON format with proper structure.

#### 4.4 Analyze Trends

**Concise Trend Analysis** - If historical data exists (at least 3 days), show:

**3-Day Comparison** (focus on last 3 days):
- Success rate trend (improving/declining/stable with percentage)
- Notable changes only - omit stable metrics

**Skip monthly summaries** unless specifically showing anomalies or significant changes.

**Trend Indicators**:
- 📈 Improving: Metric significantly better (>10% change)
- 📉 Declining: Metric significantly worse (>10% change)
- ➡️ Stable: Metric within 10% (don't report unless notable)

### Phase 5: Skip Instruction Changes Analysis

**Omit this phase** - instruction file correlation analysis adds unnecessary verbosity. Only include if there's a clear, immediate issue to investigate.

### Phase 6: Create Concise Analysis Discussion

Create a **concise** discussion with your findings using the safe-outputs create-discussion functionality.

**Discussion Title**: `Daily Copilot Agent Analysis - [DATE]`

**Shared Discussion Structure** (both variants):
```markdown
### 🤖 Copilot Agent PR Analysis - [DATE]

**Analysis Period**: Last 24 hours
[Variant-specific body goes here]

---

_Generated by Copilot Agent Analysis (Run: [run_id])_
```

{{#if experiments.output_format == 'structured' }}
**Structured Variant Body Template**:
```markdown
**Total PRs** (`agent_prs_total`): [count] | **Merged** (`agent_prs_merged`): [count] ([percentage]%) | **Avg Duration**: [time]

**Performance Metrics**

| Date | PRs | Merged | Success Rate | Avg Duration | Avg Comments |
|------|-----|--------|--------------|--------------|--------------|
| [today] | [count] | [count] | [%] | [time] | [count] |
| [today-1] | [count] | [count] | [%] | [time] | [count] |
| [today-2] | [count] | [count] | [%] | [time] | [count] |

**Trend**: [Only mention if significant change >10%]

<details>
<summary><b>Agent Task Texts</b></summary>

| PR # | Status | Task Text (first 100 chars) |
|------|--------|----------------------------|
| [#number]([url]) | [Merged/Closed/Open] | [PR body truncated to 100 chars, or "No description provided"] |

</details>

<details>
<summary><b>Notable PRs</b></summary>

[Only if failures, closures, or PRs open >24h — otherwise omit]

</details>

**Key Insights**

[1-2 bullets max — omit if nothing notable]
```
{{/if}}
{{#if experiments.output_format == 'prose' }}
**Prose Variant Body Template**:
```markdown
In the last 24 hours, Copilot agent created [count] PRs (`agent_prs_total`), of which [count] were merged ([percentage]% success rate, `agent_prs_merged`) with an average duration of [time] and [count] human comments per PR. [One sentence on 3-day trend only if success rate changed >10%, e.g. "Success rate improved from X% to Y% over the last 3 days." — otherwise omit.] [One sentence listing any notable PRs by number only if failures, closures, or PRs open >24h exist — otherwise omit.]

- [Key insight 1: single most actionable observation — omit bullet entirely if nothing notable]
- [Key insight 2: secondary pattern or trend worth flagging — omit bullet entirely if nothing notable]
```
{{/if}}

## Important Guidelines

### Security and Data Handling
- **Use sanitized context**: Always use GitHub API data, not raw user input
- **Validate dates**: Ensure date calculations are correct (handle timezone differences)
- **Handle missing data**: Some PRs may not have complete metadata
- **Respect privacy**: Don't expose sensitive information in discussions

### Analysis Quality
- **Be accurate**: Double-check all calculations and metrics
- **Be consistent**: Use the same metrics each day for valid comparisons
- **Be thorough**: Don't skip PRs or data points
- **Be objective**: Report facts without bias

### Cache Memory Management
- **Organize data**: Keep historical data well-structured in JSON format
- **Limit retention**: Keep last 90 days (3 months) of daily data for trend analysis
- **Handle errors**: If repo memory is corrupted, reinitialize gracefully
- **Simplified data collection**: Focus on 3-day trends, not weekly or monthly
  - Only collect and maintain last 3 days of data for trend comparison
  - Save progress after each day to ensure data persistence
  - Stop at 3 days - sufficient for concise reports

### Trend Analysis
- **Require sufficient data**: Don't report trends with less than 3 days of data
- **Focus on significant changes**: Only report metrics with >10% change
- **Be concise**: Avoid verbose explanations - use trend indicators and percentages
- **Skip stable metrics**: Don't clutter the report with metrics that haven't changed significantly

## Edge Cases

### No PRs in Last 24 Hours
If no PRs were created by Copilot in the last 24 hours:
- Create a minimal discussion: "No Copilot coding agent activity in the last 24 hours."
- Update repo memory with zero counts
- Keep it to 2-3 sentences max

### Bot Username Changes
If Copilot appears under different usernames:
- Note briefly in Key Insights section
- Adjust search queries accordingly

### Incomplete PR Data
If some PRs have missing metadata:
- Note count of incomplete PRs in one line
- Calculate metrics only from complete data

## Success Criteria

A successful **concise** analysis:
- ✅ Finds all Copilot PRs from last 24 hours
- ✅ Calculates key metrics (success rate, duration, comments)
- ✅ Shows 3-day trend comparison (not 7-day or monthly)
- ✅ Updates repo memory with today's metrics
- ✅ Only highlights notable PRs (failures, closures, long-open)
- ✅ Keeps discussion to ~15-20 lines of essential information
- ✅ Omits verbose tables, detailed breakdowns, and methodology sections
- ✅ Provides 1-2 actionable insights maximum

**Remember**: Less is more. Focus on key metrics and notable changes only.

{{#runtime-import shared/noop-reminder.md}}
