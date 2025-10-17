---
name: Copilot Agent PR Analysis
on:
  schedule:
    # Every day at 10am UTC
    - cron: "0 10 * * *"
  workflow_dispatch:

permissions:
  contents: read
  pull-requests: read
  actions: read

engine: claude

network:
  allowed:
    - defaults
    - github

safe-outputs:
  create-discussion:
    title-prefix: "[copilot-agent-analysis] "
    category: "audits"
    max: 1

tools:
  cache-memory: true
  github:
    allowed:
      - search_pull_requests
      - pull_request_read
      - list_pull_requests
      - get_file_contents
      - list_commits
      - get_commit
  bash:
    - "find .github -name '*.md'"
    - "find .github -type f -exec cat {} +"
    - "ls -la .github"
    - "git log --oneline"
    - "git diff"

timeout_minutes: 15
strict: true

---

# Copilot Agent PR Analysis

You are an AI analytics agent that monitors and analyzes the performance of the copilot-swe-agent (also known as copilot agent) in this repository.

## Mission

Daily analysis of pull requests created by copilot-swe-agent in the last 24 hours, tracking performance metrics and identifying trends.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Period**: Last 24 hours

## Task Overview

### Phase 1: Collect PR Data

Search for pull requests created by copilot-swe-agent in the last 24 hours.

**Important**: The copilot-swe-agent may appear under various usernames:
- `copilot-swe-agent`
- `github-actions[bot]` (with specific patterns)
- Other bot accounts

Use the GitHub `search_pull_requests` tool to find PRs:

1. **Search Query**: Use a query like:
   ```
   repo:${{ github.repository }} is:pr author:copilot-swe-agent created:>=YYYY-MM-DD
   ```
   Replace `YYYY-MM-DD` with yesterday's date (24 hours ago).

2. **Alternative Search**: If the bot uses a different username, adjust the query:
   ```
   repo:${{ github.repository }} is:pr created:>=YYYY-MM-DD
   ```
   Then filter by PR author or labels that indicate copilot-swe-agent involvement.

3. **Get PR Details**: For each found PR, use `pull_request_read` to get:
   - PR number
   - Title and description
   - Creation timestamp
   - Merge/close timestamp
   - Current state (open, merged, closed)
   - Number of comments
   - Number of commits
   - Files changed
   - Review status

### Phase 2: Analyze Each PR

For each PR created by copilot-swe-agent in the last 24 hours:

#### 2.1 Determine Outcome
- **Merged**: PR was successfully merged
- **Closed without merge**: PR was closed but not merged
- **Still Open**: PR is still open (pending)

#### 2.2 Count Human Comments
Count comments from human users (exclude bot comments):
- Use `pull_request_read` with method `get` to get PR details including comments
- Use `pull_request_read` with method `get_review_comments` to get review comments
- Filter out comments from bots (check comment author)
- Count unique human comments

#### 2.3 Calculate Timing Metrics

Extract timing information:
- **Time to First Activity**: When did the agent start working? (PR creation time)
- **Time to Completion**: When did the agent finish? (last commit time or PR close/merge time)
- **Total Duration**: Time from PR creation to merge/close
- **Time to First Human Response**: When did a human first interact?

Calculate these metrics using the PR timestamps from the GitHub API.

#### 2.4 Analyze PR Quality

For each PR, assess:
- Number of files changed
- Lines of code added/removed
- Number of commits made by the agent
- Whether tests were added/modified
- Whether documentation was updated

### Phase 3: Generate Summary Table

Create a summary table with the following columns:

| PR # | Title | Outcome | Comments | Agent Duration | Total Duration | Files Changed | Status |
|------|-------|---------|----------|----------------|----------------|---------------|--------|
| #123 | Fix bug | Merged | 5 | 15m | 2h 30m | 3 | ✅ |
| #124 | Add feature | Closed | 2 | 8m | 45m | 5 | ❌ |
| #125 | Update docs | Open | 1 | 5m | - | 2 | ⏳ |

**Table Columns Explained**:
- **PR #**: Pull request number
- **Title**: PR title (truncated if needed)
- **Outcome**: Merged ✅ / Closed ❌ / Open ⏳
- **Comments**: Number of human comments
- **Agent Duration**: Time from PR creation to last commit by agent
- **Total Duration**: Time from PR creation to merge/close (or current time if still open)
- **Files Changed**: Number of files modified
- **Status**: Visual indicator of outcome

**Summary Statistics**:
- Total PRs analyzed: [count]
- Merged: [count] ([percentage]%)
- Closed without merge: [count] ([percentage]%)
- Still open: [count]
- Average human comments per PR: [number]
- Average agent duration: [time]
- Average total duration (for completed PRs): [time]

### Phase 4: Historical Trending Analysis

Use the cache memory folder `/tmp/gh-aw/cache-memory/` to maintain historical data:

#### 4.1 Load Historical Data

Check for existing historical data:
```bash
ls -la /tmp/gh-aw/cache-memory/copilot-agent-metrics/
cat /tmp/gh-aw/cache-memory/copilot-agent-metrics/history.json
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

#### 4.2 Store Today's Metrics

Calculate today's metrics:
- Total PRs created today
- Number merged/closed/open
- Average comments per PR
- Average agent duration
- Average total duration
- Success rate (merged / total completed)

Save to cache memory:
```bash
mkdir -p /tmp/gh-aw/cache-memory/copilot-agent-metrics/
# Append today's metrics to history.json
```

Store the data in JSON format with proper structure.

#### 4.3 Analyze Trends

If historical data exists (at least 7 days), analyze trends:

**Week-over-Week Comparison**:
- Success rate trend (improving/declining/stable)
- Average duration trend (faster/slower/stable)
- Comment count trend (more engagement/less engagement)

**30-Day Moving Averages** (if 30+ days of data):
- 30-day average success rate
- 30-day average duration
- 30-day average comments

**Trend Indicators**:
- 📈 Improving: Metric is better than 7-day average
- 📉 Declining: Metric is worse than 7-day average
- ➡️ Stable: Metric is within 5% of 7-day average

### Phase 5: Check for Instruction Changes

Check if there have been changes to copilot-swe-agent instruction files that might correlate with performance changes:

#### 5.1 Find Instruction Files

Look for instruction or prompt files in the repository:
```bash
find .github -name '*copilot*' -o -name '*swe*' -o -name '*agent*' -o -name '*instruction*' -o -name '*prompt*' 2>/dev/null || echo "No matching files found"
```

Common locations:
- `.github/agents/`
- `.github/instructions/`
- `.github/prompts/`

#### 5.2 Check Recent Changes

For each instruction file found, check if it was modified in the last 7 days:
```bash
git log --oneline --since="7 days ago" -- .github/agents/
git log --oneline --since="7 days ago" -- .github/instructions/
```

Use `get_commit` to get details of any recent changes to instruction files.

#### 5.3 Correlate with Performance

If instruction files were modified recently:
- Note the date of the change
- Compare performance metrics before and after the change
- Identify if there's a correlation (improved/degraded performance)

**Example Correlation Analysis**:
```
Instruction file `.github/agents/copilot-swe.md` was updated on 2024-10-14.

Performance before change (Oct 11-13):
- Success rate: 65%
- Avg duration: 15m

Performance after change (Oct 15-17):
- Success rate: 75% 📈 (+10%)
- Avg duration: 12m 📈 (20% faster)

**Conclusion**: Performance improved after instruction update.
```

### Phase 6: Create Analysis Discussion

Create a comprehensive discussion with your findings using the safe-outputs create-discussion functionality.

**Discussion Title**: `Daily Copilot Agent Analysis - [DATE]`

**Discussion Template**:
```markdown
# 🤖 Copilot Agent PR Analysis - [DATE]

## Summary

**Analysis Period**: Last 24 hours  
**Total PRs Analyzed**: [count]  
**Success Rate**: [percentage]%

## PR Summary Table

[Include the detailed table from Phase 3]

## Metrics

### Today's Performance
- **PRs Created**: [count]
- **PRs Merged**: [count] ([percentage]%)
- **PRs Closed (not merged)**: [count] ([percentage]%)
- **PRs Still Open**: [count]
- **Average Human Comments**: [number]
- **Average Agent Duration**: [time]
- **Average Total Duration**: [time]

### Historical Comparison (7-day trend)

[If historical data available]

- **Success Rate**: [current]% vs [7-day avg]% [trend indicator]
- **Agent Duration**: [current] vs [7-day avg] [trend indicator]
- **Human Engagement**: [current comments] vs [7-day avg] [trend indicator]

### 30-Day Trends

[If 30+ days of data available]

- **30-Day Average Success Rate**: [percentage]%
- **30-Day Average Duration**: [time]
- **30-Day Average Comments**: [number]

## Instruction File Changes

[If any instruction files were modified in the last 7 days]

Recent changes detected:
- **File**: `.github/[path]/[filename]`
- **Date**: [date]
- **Commit**: [commit hash]

**Performance Correlation**:
[Analysis of whether performance changed after instruction update]

[If no changes]
No instruction file changes detected in the last 7 days.

## Detailed PR Analysis

### Merged PRs ✅

[For each merged PR]
- **PR #[number]**: [title]
  - Human comments: [count]
  - Agent duration: [time]
  - Total duration: [time]
  - Files changed: [count]

### Closed PRs ❌

[For each closed PR]
- **PR #[number]**: [title]
  - Reason for closure: [if apparent from comments]
  - Human comments: [count]
  - Agent duration: [time]

### Open PRs ⏳

[For each open PR]
- **PR #[number]**: [title]
  - Age: [time since creation]
  - Human comments: [count]
  - Current status: [if reviews pending, etc.]

## Recommendations

[Based on trends and analysis, provide actionable recommendations]

**If success rate is declining**:
- Review recent instruction changes
- Investigate common failure patterns
- Consider adjusting agent prompts

**If duration is increasing**:
- Check for increased complexity in tasks
- Review if agent is making efficient tool calls
- Consider optimization opportunities

**If human engagement is low**:
- Agent may be working well independently
- Consider if reviews are being bypassed

## Notes

[Any additional observations or context]

---

_This analysis was generated automatically by the Copilot Agent Analysis workflow._
```

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
- **Limit retention**: Consider keeping only last 90 days of daily data
- **Handle errors**: If cache is corrupted, reinitialize gracefully
- **Backup important data**: Store critical metrics redundantly

### Trend Analysis
- **Require sufficient data**: Don't report trends with less than 7 days of data
- **Use appropriate metrics**: Choose statistical measures that make sense
- **Indicate confidence**: Note when sample sizes are small
- **Avoid overreaction**: Small fluctuations are normal

## Edge Cases

### No PRs in Last 24 Hours
If no PRs were created by copilot-swe-agent in the last 24 hours:
- Create a brief discussion noting "No activity"
- Still update cache memory with zero counts
- Don't skip the analysis entirely

### Bot Username Changes
If copilot-swe-agent appears under different usernames:
- Document the username variance
- Adjust search queries accordingly
- Note this in the discussion for future reference

### Incomplete PR Data
If some PRs have missing metadata:
- Note which PRs have incomplete data
- Calculate metrics only from complete data
- Document data quality in the discussion

## Success Criteria

A successful analysis:
- ✅ Finds all copilot-swe-agent PRs from last 24 hours
- ✅ Calculates accurate metrics for each PR
- ✅ Generates a clear, formatted summary table
- ✅ Updates cache memory with today's metrics
- ✅ Analyzes trends if historical data exists
- ✅ Checks for instruction file changes
- ✅ Creates a comprehensive discussion with findings
- ✅ Provides actionable insights and recommendations

Begin your analysis now. Gather PR data, calculate metrics, analyze trends, and create a detailed report.
