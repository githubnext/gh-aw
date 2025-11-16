---
description: Provides daily team status updates summarizing activity, progress, and blockers across the team
on:
  schedule:
    # Every day at 9am UTC, all days except Saturday and Sunday
    - cron: "0 9 * * 1-5"
  workflow_dispatch:
  # workflow will no longer trigger after 30 days. Remove this and recompile to run indefinitely
  stop-after: +30d 

permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
  actions: read

campaign: daily-team-status
engine: copilot

timeout-minutes: 30

network:
  allowed:
    - defaults
    - python
    - node
  firewall: true

safe-outputs:
  upload-assets:
  create-discussion:
    title-prefix: "[team-status] "
    category: "announcements"

tools:
  cache-memory:
  edit:
  bash:
    - "*"
  github:
    toolsets:
      - default
      - discussions
  web-fetch:

# Pre-download GitHub data in steps to avoid excessive MCP calls
# Uses cache-memory to persist data across runs and avoid re-fetching
steps:
  - name: Download team activity data
    id: download-data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e
      
      # Create directories
      mkdir -p /tmp/gh-aw/team-status-data
      mkdir -p /tmp/gh-aw/cache-memory/team-status-data
      
      # Check if cached data exists and is recent (< 24 hours old)
      CACHE_VALID=false
      CACHE_TIMESTAMP_FILE="/tmp/gh-aw/cache-memory/team-status-data/.timestamp"
      
      if [ -f "$CACHE_TIMESTAMP_FILE" ]; then
        CACHE_AGE=$(($(date +%s) - $(cat "$CACHE_TIMESTAMP_FILE")))
        # 24 hours = 86400 seconds
        if [ $CACHE_AGE -lt 86400 ]; then
          echo "✅ Found valid cached data (age: ${CACHE_AGE}s, less than 24h)"
          CACHE_VALID=true
        else
          echo "⚠ Cached data is stale (age: ${CACHE_AGE}s, more than 24h)"
        fi
      else
        echo "ℹ No cached data found, will fetch fresh data"
      fi
      
      # Use cached data if valid, otherwise fetch fresh data
      if [ "$CACHE_VALID" = true ]; then
        echo "📦 Using cached data from previous run"
        cp -r /tmp/gh-aw/cache-memory/team-status-data/* /tmp/gh-aw/team-status-data/
        echo "✅ Cached data restored to working directory"
      else
        echo "🔄 Fetching fresh data from GitHub API..."
        
        # Fetch issues (open and recently closed)
        echo "Fetching issues..."
        gh api graphql -f query="
          query(\$owner: String!, \$repo: String!) {
            repository(owner: \$owner, name: \$repo) {
              openIssues: issues(first: 50, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
                nodes {
                  number
                  title
                  state
                  createdAt
                  updatedAt
                  author { login }
                  labels(first: 10) { nodes { name } }
                  comments { totalCount }
                }
              }
              closedIssues: issues(first: 30, states: CLOSED, orderBy: {field: UPDATED_AT, direction: DESC}) {
                nodes {
                  number
                  title
                  state
                  createdAt
                  updatedAt
                  closedAt
                  author { login }
                  labels(first: 10) { nodes { name } }
                }
              }
            }
          }
        " -f owner="${GITHUB_REPOSITORY_OWNER}" -f repo="${GITHUB_REPOSITORY#*/}" > /tmp/gh-aw/team-status-data/issues.json
        
        # Fetch pull requests (open and recently merged/closed)
        echo "Fetching pull requests..."
        gh api graphql -f query="
          query(\$owner: String!, \$repo: String!) {
            repository(owner: \$owner, name: \$repo) {
              openPRs: pullRequests(first: 30, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
                nodes {
                  number
                  title
                  state
                  createdAt
                  updatedAt
                  author { login }
                  additions
                  deletions
                  changedFiles
                  reviews(first: 10) { totalCount }
                }
              }
              mergedPRs: pullRequests(first: 30, states: MERGED, orderBy: {field: UPDATED_AT, direction: DESC}) {
                nodes {
                  number
                  title
                  state
                  createdAt
                  updatedAt
                  mergedAt
                  author { login }
                  additions
                  deletions
                }
              }
            }
          }
        " -f owner="${GITHUB_REPOSITORY_OWNER}" -f repo="${GITHUB_REPOSITORY#*/}" > /tmp/gh-aw/team-status-data/pull_requests.json
        
        # Fetch recent commits (last 50)
        echo "Fetching commits..."
        gh api "repos/${GITHUB_REPOSITORY}/commits?per_page=50" \
          --jq '[.[] | {sha, author: .commit.author, message: .commit.message, date: .commit.author.date, html_url}]' \
          > /tmp/gh-aw/team-status-data/commits.json
        
        # Fetch discussions
        echo "Fetching discussions..."
        gh api graphql -f query="
          query(\$owner: String!, \$repo: String!) {
            repository(owner: \$owner, name: \$repo) {
              discussions(first: 20, orderBy: {field: UPDATED_AT, direction: DESC}) {
                nodes {
                  number
                  title
                  createdAt
                  updatedAt
                  author { login }
                  category { name }
                  comments { totalCount }
                  url
                }
              }
            }
          }
        " -f owner="${GITHUB_REPOSITORY_OWNER}" -f repo="${GITHUB_REPOSITORY#*/}" > /tmp/gh-aw/team-status-data/discussions.json
        
        # Cache the freshly downloaded data for next run
        echo "💾 Caching data for future runs..."
        cp -r /tmp/gh-aw/team-status-data/* /tmp/gh-aw/cache-memory/team-status-data/
        date +%s > "$CACHE_TIMESTAMP_FILE"
        
        echo "✅ Data download and caching complete"
      fi
      
      ls -lh /tmp/gh-aw/team-status-data/

imports:
  - shared/reporting.md
  - shared/trends.md

source: githubnext/agentics/workflows/daily-team-status.md@1e366aa4518cf83d25defd84e454b9a41e87cf7c
---

# Daily Team Status

Create an upbeat daily status report for the team as a GitHub discussion.

## 📁 Pre-Downloaded Data Available

**IMPORTANT**: All GitHub data has been pre-downloaded to `/tmp/gh-aw/team-status-data/` to avoid excessive MCP calls. Use these files instead of making GitHub API calls:

- **`issues.json`** - Open and recently closed issues (50 open + 30 closed)
- **`pull_requests.json`** - Open and merged pull requests (30 open + 30 merged)
- **`commits.json`** - Recent commits (last 50)
- **`discussions.json`** - Recent discussions (last 20)

**Load and analyze these files** instead of making repeated GitHub MCP calls. All data is in JSON format.

## 💾 Cache Memory Available

**Cache-memory is enabled** - You have access to persistent storage at `/tmp/gh-aw/cache-memory/` that persists across workflow runs:

- Use it to **store intermediate analysis results** that might be useful for future runs
- Store **processed data, statistics, or insights** that take time to compute
- Cache **team metrics, velocity data, or trend analysis** for historical comparison
- Files stored here will be available in the next workflow run (cached for 24 hours)

**Example use cases**:
- Save team productivity metrics (e.g., `/tmp/gh-aw/cache-memory/team-metrics.json`)
- Cache velocity calculations for trend analysis
- Store historical data for week-over-week comparisons

## What to Include

Using the pre-downloaded data from `/tmp/gh-aw/team-status-data/`, create a comprehensive status report including:

### Team Activity Overview
- Recent repository activity (from issues.json, pull_requests.json)
- Active contributors and their focus areas
- Progress on key initiatives (based on PR titles, issue labels)
- Recently completed work (merged PRs, closed issues)

### Team Health Indicators
- Open issue count and trends
- PR review velocity (time from open to merge)
- Discussion engagement levels
- Code contribution patterns (from commits.json)

### Productivity Insights
- Suggested process improvements based on bottlenecks
- Ideas for reducing cycle time or improving workflows
- Community engagement opportunities
- Feature prioritization recommendations

### Looking Ahead
- Upcoming priorities (based on open issues, in-progress PRs)
- Areas needing attention or support
- Investment opportunities for team growth

## Report Structure

Follow these guidelines for your report (see shared/reporting.md for detailed formatting):

1. **Overview**: Start with 1-2 paragraphs summarizing the key highlights and team status
2. **Detailed Sections**: Use collapsible `<details>` sections for comprehensive content
3. **Data References**: In a note at the end, include:
   - Files read from `/tmp/gh-aw/team-status-data/`
   - Summary statistics (issues/PRs/commits/discussions analyzed)
   - Any data limitations encountered

## Data Processing Workflow

1. **Load pre-downloaded files** from `/tmp/gh-aw/team-status-data/`
2. **Parse JSON data** to extract relevant metrics and insights
3. **Check cache-memory** for historical data to identify trends
4. **Analyze activity patterns** to understand team dynamics
5. **Generate actionable insights** based on the data
6. **Create discussion** with your findings

## Style

- Be positive, encouraging, and helpful 🌟
- Use emojis moderately for engagement
- Keep it concise - adjust length based on actual activity
- Focus on actionable insights rather than just data listing
- Celebrate wins and progress
- Frame challenges as opportunities for improvement

Create a new GitHub discussion with a title containing today's date (e.g., "Team Status - 2024-11-16") containing a markdown report with your findings. Use links where appropriate.

Only a new discussion should be created, do not close or update any existing discussions.
