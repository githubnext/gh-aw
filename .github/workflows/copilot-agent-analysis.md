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

imports:
  - shared/jqschema.md

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
    - "gh pr list *"
    - "gh search prs *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"

steps:
  - name: Fetch Copilot PR data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directory
      mkdir -p /tmp/gh-aw/pr-data
      
      # Calculate date 30 days ago
      DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')
      
      # Search for PRs created by Copilot in the last 30 days using gh CLI
      # Output in JSON format for easy processing with jq
      echo "Fetching Copilot PRs from the last 30 days..."
      gh search prs \
        --repo "${{ github.repository }}" \
        --json number,title,state,createdAt,closedAt,mergedAt,author,comments,additions,deletions,changedFiles,commits,url \
        --created ">=$DATE_30_DAYS_AGO" \
        --limit 1000 \
        > /tmp/gh-aw/pr-data/copilot-prs-raw.json
      
      # Filter to only Copilot author (user.login == "Copilot" and user.id == 198982749)
      jq '[.[] | select(.author.login == "Copilot" or .author.id == 198982749)]' \
        /tmp/gh-aw/pr-data/copilot-prs-raw.json \
        > /tmp/gh-aw/pr-data/copilot-prs.json
      
      # Generate schema for reference
      cat /tmp/gh-aw/pr-data/copilot-prs.json | /tmp/gh-aw/jqschema.sh > /tmp/gh-aw/pr-data/copilot-prs-schema.json
      
      echo "PR data saved to /tmp/gh-aw/pr-data/copilot-prs.json"
      echo "Schema saved to /tmp/gh-aw/pr-data/copilot-prs-schema.json"
      echo "Total PRs found: $(jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json)"

timeout_minutes: 15
strict: true

---

# Copilot Agent PR Analysis

You are an AI analytics agent that monitors and analyzes the performance of the copilot-swe-agent (also known as copilot agent) in this repository.

## Mission

Daily analysis of pull requests created by copilot-swe-agent in the last 24 hours, tracking performance metrics and identifying trends. Provides daily, weekly, and monthly performance summaries.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Period**: Last 24 hours (with weekly and monthly summaries)

## Task Overview

### Phase 1: Collect PR Data

**Pre-fetched Data Available**: This workflow includes a preparation step that has already fetched Copilot PR data for the last 30 days using gh CLI. The data is available at:
- `/tmp/gh-aw/pr-data/copilot-prs.json` - Full PR data in JSON format
- `/tmp/gh-aw/pr-data/copilot-prs-schema.json` - Schema showing the structure

You can use `jq` to process this data directly. For example:
```bash
# Get PRs from the last 24 hours
TODAY=$(date -d '24 hours ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-24H '+%Y-%m-%dT%H:%M:%SZ')
jq --arg today "$TODAY" '[.[] | select(.createdAt >= $today)]' /tmp/gh-aw/pr-data/copilot-prs.json

# Count total PRs
jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json

# Get PR numbers for the last 24 hours
jq --arg today "$TODAY" '[.[] | select(.createdAt >= $today) | .number]' /tmp/gh-aw/pr-data/copilot-prs.json
```

**Alternative Approaches** (if you need additional data not in the pre-fetched file):

Search for pull requests created by Copilot in the last 24 hours.

**Important**: The Copilot coding agent creates PRs under the username `Copilot` (user ID 198982749, a Bot account). GitHub's search API doesn't support searching by bot authors using `author:` filter, so we need alternative approaches.

Use the GitHub tools with one of these strategies:

1. **Search by keywords in title/body (Recommended)**:
   ```
   repo:${{ github.repository }} is:pr "START COPILOT CODING AGENT" created:>=YYYY-MM-DD
   ```
   This searches for PRs containing the signature text that Copilot adds to PR bodies.
   Replace `YYYY-MM-DD` with yesterday's date (24 hours ago).

2. **List all PRs and filter by author**:
   Use `list_pull_requests` tool to get recent PRs, then filter by checking if:
   - `user.login == "Copilot"` 
   - `user.id == 198982749`
   - `user.type == "Bot"`
   
   This is more reliable but requires processing all recent PRs.

3. **Search by common patterns**:
   ```
   repo:${{ github.repository }} is:pr "Original prompt" created:>=YYYY-MM-DD
   ```
   Copilot PRs typically contain "Original prompt" in their body.

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

For each PR created by Copilot in the last 24 hours:

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

**If Historical Data is Missing or Incomplete:**

If the history file doesn't exist or has gaps in the data, rebuild it by querying historical PRs:

1. **Determine Missing Date Range**: Identify which dates need data (up to last 7 days maximum for meaningful trends)

2. **Query PRs One Day at a Time**: To avoid context explosion, query PRs for each missing day separately:
   ```
   repo:${{ github.repository }} is:pr author:copilot-swe-agent created:YYYY-MM-DD..YYYY-MM-DD
   ```
   
3. **Process Each Day**: For each day with missing data:
   - Query PRs created on that specific date
   - Calculate the same metrics as for today (total PRs, merged, closed, success rate, etc.)
   - Store in the history file
   - Limit processing to avoid timeout - prioritize most recent days first

4. **Incremental Approach**: 
   - Process one day at a time in chronological order (oldest to newest)
   - Save after each day to preserve progress
   - If you have 5+ days of data, that's sufficient for basic trend analysis
   - Aim for 7 days maximum for week-over-week trends
   - Do not attempt to collect more than 7 days of historical data

5. **Rate Limiting**: Be mindful of API rate limits - if approaching limits, save what you have and continue next run

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

#### 4.2.1 Rebuild Historical Data (if needed)

**When to Rebuild:**
- History file doesn't exist
- History file has gaps (missing dates in the last 7 days)
- Insufficient data for trend analysis (< 7 days)

**Rebuilding Strategy:**
1. **Assess Current State**: Check how many days of data you have
2. **Target Collection**: Aim for at least 7 days maximum (for weekly trends)
3. **One Day at a Time**: Query PRs for each missing date separately to avoid context explosion

**For Each Missing Day:**
```
# Query PRs for specific date using keyword search
repo:${{ github.repository }} is:pr "START COPILOT CODING AGENT" created:YYYY-MM-DD..YYYY-MM-DD
```

Or use `list_pull_requests` with date filtering and filter results by `user.login == "Copilot"` and `user.id == 198982749`.

**Process:**
- Start with the oldest missing date in your target range (maximum 7 days ago)
- For each date:
  1. Search for PRs created on that date
  2. Analyze each PR (same as Phase 2)
  3. Calculate daily metrics (same as Phase 4.2)
  4. Add to history.json
  5. Save immediately to preserve progress
- Continue until you have sufficient data (up to 7 days) or reach time limits

**Important Constraints:**
- Process dates in chronological order (oldest first)
- Save after processing each day
- If time is running short (> 10 minutes elapsed), stop and save what you have
- Next run will continue from where you left off
- Prioritize data quality over quantity - better to have accurate data for fewer days

#### 4.3 Store Today's Metrics

After ensuring historical data is available (either from existing cache or rebuilt), add today's metrics:
- Total PRs created today
- Number merged/closed/open
- Average comments per PR
- Average agent duration
- Average total duration
- Success rate (merged / total completed)

Append to history.json in the cache memory.

#### 4.4 Analyze Trends

If historical data exists (at least 7 days), analyze trends:

**Week-over-Week Comparison** (last 7 days):
- Success rate trend (improving/declining/stable)
- Average duration trend (faster/slower/stable)
- Comment count trend (more engagement/less engagement)
- Volume trend (more/fewer PRs)

**Trend Indicators**:
- 📈 Improving: Metric is better than comparison period
- 📉 Declining: Metric is worse than comparison period
- ➡️ Stable: Metric is within 5% of comparison period

### Phase 5: Check for Instruction Changes

Check if there have been changes to Copilot coding agent instruction files that might correlate with performance changes:

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

### Weekly Summary (Last 7 Days)

[If at least 7 days of historical data available]

**Performance Metrics:**
- **Total PRs**: [count]
- **Success Rate**: [percentage]%
- **Average Duration**: [time]
- **Average Comments**: [number]

**Daily Breakdown (Last 7 Days):**
| Date | PRs | Merged | Success Rate | Avg Duration |
|------|-----|--------|--------------|--------------|
| [date] | [count] | [count] | [%] | [time] |
| ... | ... | ... | ... | ... |

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
- **Limit retention**: Keep only last 7 days of daily data (maximum)
- **Handle errors**: If cache is corrupted, reinitialize gracefully
- **Backup important data**: Store critical metrics redundantly
- **Progressive data collection**: If historical data is missing, rebuild incrementally
  - Prioritize most recent days first (they're more relevant for trends)
  - Process one day at a time to avoid overwhelming the context
  - Save progress after each day to ensure data persistence
  - Aim for 7 days maximum for weekly trends

### Trend Analysis
- **Require sufficient data**: Don't report trends with less than 7 days of data
- **Use appropriate metrics**: Choose statistical measures that make sense
- **Indicate confidence**: Note when sample sizes are small
- **Avoid overreaction**: Small fluctuations are normal

## Edge Cases

### No PRs in Last 24 Hours
If no PRs were created by Copilot in the last 24 hours:
- Create a brief discussion noting "No activity"
- Still update cache memory with zero counts
- Don't skip the analysis entirely

### Bot Username Changes
If Copilot appears under different usernames:
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
- ✅ Finds all Copilot PRs from last 24 hours
- ✅ Calculates accurate metrics for each PR
- ✅ Generates a clear, formatted summary table
- ✅ Updates cache memory with today's metrics
- ✅ Rebuilds missing historical data if needed (one day at a time, up to 7 days maximum)
- ✅ Analyzes trends with available historical data
- ✅ Provides weekly summary (if 7 days of data available)
- ✅ Checks for instruction file changes
- ✅ Creates a comprehensive discussion with findings
- ✅ Provides actionable insights and recommendations

Begin your analysis now. Gather PR data, calculate metrics, analyze trends, and create a detailed report.
