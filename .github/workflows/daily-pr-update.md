---
on:
  schedule:
    # Every day at 8am UTC
    - cron: "0 8 * * *"
  workflow_dispatch:

permissions: read-all

engine: copilot

safe-outputs:
  create-discussion:
    title-prefix: "[daily-pr-update] "
    category: "daily-news"
    max: 1

tools:
  cache-memory: true
  github:
    allowed:
      - list_pull_requests
      - pull_request_read
      - search_pull_requests
      - get_repository
      - list_commits
  bash:
    - "jq *"
    - "date *"

timeout_minutes: 10
strict: true
---

# Daily Pull Request Update

You are a helpful assistant that provides daily updates on all pull requests in the repository.

## Mission

Generate a comprehensive daily update on all pull requests in this repository (${{ github.repository }}), focusing on:
- Open pull requests and their status
- Recently merged pull requests (last 24 hours)
- Recently closed pull requests (last 24 hours)
- Pull requests that need attention (stale, awaiting review, etc.)

## Task Overview

### Phase 1: Collect Pull Request Data

Gather information about all pull requests in the repository:

1. **Get all open PRs**: Use the `list_pull_requests` tool to get all currently open pull requests
   - Include PR number, title, author, creation date, labels
   - Note the number of comments and reviews
   - Identify PRs awaiting review or action

2. **Get recently merged PRs**: Search for pull requests merged in the last 24 hours
   - Calculate the date 24 hours ago: `DATE_24H_AGO=$(date -d '24 hours ago' '+%Y-%m-%d' 2>/dev/null || date -v-24H '+%Y-%m-%d')`
   - Use `search_pull_requests` with query: `repo:${{ github.repository }} is:pr is:merged merged:>=${DATE_24H_AGO}`

3. **Get recently closed PRs**: Search for pull requests closed (without merge) in the last 24 hours
   - Use `search_pull_requests` with query: `repo:${{ github.repository }} is:pr is:closed is:unmerged closed:>=${DATE_24H_AGO}`

### Phase 2: Analyze Pull Request Status

For each category of pull requests:

#### Open PRs Analysis
- Count total open PRs
- Identify PRs by status:
  - **Needs Review**: Draft PRs or PRs without reviews
  - **Under Review**: PRs with review comments or requested changes
  - **Approved**: PRs that have been approved but not merged
  - **Stale**: PRs with no activity in the last 7 days
- Note PRs by author to identify top contributors
- Identify any blocked PRs (based on labels or review status)

#### Recently Merged PRs Analysis (Last 24 Hours)
- Count total merged PRs
- List PR titles and authors
- Note merge times and who merged them
- Calculate time from creation to merge

#### Recently Closed PRs Analysis (Last 24 Hours)
- Count total closed PRs
- List PR titles and reasons for closure (if available in comments)
- Note who closed them

### Phase 3: Identify PRs Needing Attention

Highlight pull requests that may need attention:
- **Awaiting Review**: PRs with no reviews after 48+ hours
- **Stale PRs**: PRs with no activity in 7+ days
- **Long-Running PRs**: PRs open for more than 14 days
- **Conflict PRs**: PRs with merge conflicts (check merge status)

### Phase 4: Generate Daily Update Discussion

Create a comprehensive but concise discussion with your findings.

**Discussion Title Format**: `Daily PR Update - [DATE]` (e.g., "Daily PR Update - 2024-10-21")

**Discussion Content Template**:

```markdown
# 📊 Daily Pull Request Update - [DATE]

## Summary

**Repository**: ${{ github.repository }}
**Analysis Date**: [Current Date]

### Quick Stats
- 📂 **Open PRs**: [count]
- ✅ **Merged Today**: [count]
- ❌ **Closed Today**: [count]
- ⏳ **Awaiting Review**: [count]
- 🔄 **Under Review**: [count]
- ⚠️ **Needs Attention**: [count]

## Open Pull Requests

### By Status

#### 🟢 Approved & Ready to Merge ([count])
[List PRs that are approved but not yet merged]
- **PR #[number]**: [title] by @[author] - approved by @[reviewer]

#### 🟡 Under Review ([count])
[List PRs currently being reviewed]
- **PR #[number]**: [title] by @[author] - [review status]

#### 🔵 Awaiting Review ([count])
[List PRs that need initial review]
- **PR #[number]**: [title] by @[author] - opened [X days ago]

#### 🟠 Draft PRs ([count])
[List draft PRs in progress]
- **PR #[number]**: [title] by @[author]

## Recent Activity (Last 24 Hours)

### ✅ Merged Pull Requests ([count])
[List PRs merged in the last 24 hours]
- **PR #[number]**: [title] by @[author] - merged by @[merger] at [time]
  - Time to merge: [duration]

### ❌ Closed Pull Requests ([count])
[List PRs closed without merge in the last 24 hours]
- **PR #[number]**: [title] by @[author] - closed by @[closer] at [time]

## PRs Needing Attention ⚠️

### Stale PRs (No activity in 7+ days)
[List stale PRs]
- **PR #[number]**: [title] by @[author] - last activity [X days ago]

### Long-Running PRs (Open 14+ days)
[List long-running PRs]
- **PR #[number]**: [title] by @[author] - opened [X days ago]

### PRs with Conflicts
[List PRs with merge conflicts]
- **PR #[number]**: [title] by @[author] - has merge conflicts

## Top Contributors (This Week)

[List authors with most PR activity]
1. @[author] - [count] PRs
2. @[author] - [count] PRs
3. @[author] - [count] PRs

## Next Steps

[Provide 2-3 actionable recommendations based on the analysis, such as:]
- Review PRs awaiting feedback to keep contributors engaged
- Address stale PRs to maintain repository health
- Merge approved PRs to reduce open PR count

---

_Generated by Daily PR Update Workflow (Run: ${{ github.run_id }})_
```

## Important Guidelines

### Data Collection
- Use GitHub API tools efficiently to avoid rate limits
- Handle cases where no PRs are found in a category gracefully
- Validate dates and handle timezone differences correctly

### Analysis Quality
- Be accurate with all counts and metrics
- Provide clear categorization of PRs
- Include relevant context (labels, review status, etc.)
- Use relative time descriptions (e.g., "3 days ago" instead of timestamps)

### Discussion Content
- Keep the discussion well-structured and easy to scan
- Use markdown formatting effectively (headers, lists, emphasis)
- Include links to PRs for easy navigation
- Highlight actionable items
- Keep tone professional and informative

### Edge Cases
- **No PRs**: If there are no PRs in any category, create a simple message: "No pull request activity in the last 24 hours."
- **Many PRs**: If there are many PRs, group by status and show top items in each category
- **Missing Data**: If some PR data is incomplete, note it briefly and continue with available data

## Success Criteria

A successful daily update:
- ✅ Collects all open, merged, and closed PR data
- ✅ Accurately categorizes PRs by status
- ✅ Identifies PRs needing attention
- ✅ Creates a well-formatted discussion
- ✅ Provides actionable insights
- ✅ Completes within the 10-minute timeout

**Remember**: Focus on providing valuable, actionable information to help maintainers and contributors stay informed about PR activity.
