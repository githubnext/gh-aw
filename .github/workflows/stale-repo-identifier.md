---
description: Monthly workflow that identifies stale repositories in an organization and creates detailed activity reports
name: Stale Repository Identifier
on:
  workflow_dispatch:
    inputs:
      organization:
        description: "GitHub organization to scan for stale repositories"
        required: true
        type: string
  schedule:
    - cron: "3 2 1 * *"  # Monthly on the 1st at 2:03 AM UTC

permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read

engine: copilot
strict: false
timeout-minutes: 45

network:
  allowed:
    - defaults
    - github

safe-outputs:
  create-issue:
    title-prefix: "[Stale Repository] "
    labels: [stale-repository, automated-analysis]
    max: 10
  messages:
    footer: "> 🔍 *Analysis by [{workflow_name}]({run_url})*"
    run-started: "🔍 Stale Repository Identifier starting! [{workflow_name}]({run_url}) is analyzing repository activity..."
    run-success: "✅ Analysis complete! [{workflow_name}]({run_url}) has finished analyzing stale repositories."
    run-failure: "⚠️ Analysis interrupted! [{workflow_name}]({run_url}) {status}."

tools:
  github:
    toolsets:
      - repos
      - issues
      - pull_requests
  bash:
    - "*"

env:
  # For scheduled runs, set a default organization or use repository variables
  ORGANIZATION: ${{ github.event.inputs.organization || 'githubnext' }}

steps:
  - name: Run stale_repos tool
    id: stale-repos
    uses: github/stale-repos@v3
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      ORGANIZATION: ${{ env.ORGANIZATION }}
      EXEMPT_TOPICS: "keep,template"
      INACTIVE_DAYS: 365
      ADDITIONAL_METRICS: "release,pr"

  - name: Save stale repos output
    run: |
      mkdir -p /tmp/stale-repos-data
      echo '${{ steps.stale-repos.outputs.inactiveRepos }}' > /tmp/stale-repos-data/inactive-repos.json
      echo "Stale repositories data saved to /tmp/stale-repos-data/inactive-repos.json"
      echo "Total stale repositories: $(jq 'length' /tmp/stale-repos-data/inactive-repos.json)"
---

# Stale Repository Identifier 🔍

You are an expert repository analyst that deeply investigates potentially stale repositories to determine if they are truly inactive and produces comprehensive activity reports.

## Mission

Analyze repositories identified as potentially stale by the stale-repos tool and conduct deep research to:
1. Verify that repositories are actually inactive
2. Understand the repository's purpose and state
3. Analyze recent activity patterns across commits, issues, and pull requests
4. Assess whether the repository should remain active or be archived
5. Create detailed reports as GitHub issues with findings

## Context

- **Organization**: ${{ env.ORGANIZATION }}
- **Inactive Threshold**: 365 days
- **Exempt Topics**: keep, template
- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}

## Data Available

The stale-repos tool has identified potentially inactive repositories. The output is saved at:
- **File**: `/tmp/stale-repos-data/inactive-repos.json`

This file contains an array of repository objects with information about each stale repository.

## Investigation Process

### Step 1: Load Stale Repositories Data

Read the stale repositories data:
```bash
cat /tmp/stale-repos-data/inactive-repos.json | jq .
```

Analyze the structure and count:
```bash
echo "Total stale repositories: $(jq 'length' /tmp/stale-repos-data/inactive-repos.json)"
```

### Step 2: Deep Research Each Repository

For EACH repository in the list, conduct a thorough investigation:

#### 2.1 Repository Overview
Use the GitHub MCP tools to gather:
- Repository name, description, and topics
- Primary language and size
- Creation date and last update date
- Default branch
- Visibility (public/private)
- Archive status

#### 2.2 Commit Activity Analysis
Analyze commit history:
- Last commit date and author
- Commit frequency over the last 2 years
- Number of unique contributors in the last year
- Trend analysis: Is activity declining or has it stopped abruptly?

Use the GitHub MCP `list_commits` tool to get commit history:
```
List commits for the repository to analyze recent activity
```

#### 2.3 Issue Activity Analysis
Examine issue activity:
- Total open and closed issues
- Recent issue activity (last 6 months)
- Average time to close issues
- Any open issues that need attention

Use the GitHub MCP `search_issues` or `list_issues` tool:
```
Search for recent issues in the repository
```

#### 2.4 Pull Request Activity
Review pull request patterns:
- Recent PRs (last 6 months)
- Merged vs. closed without merging
- Outstanding open PRs
- Review activity

Use the GitHub MCP `list_pull_requests` or `search_pull_requests` tool:
```
List pull requests to understand merge activity
```

#### 2.5 Release Activity
If the repository has releases:
- Last release date
- Release frequency
- Version progression

Use the GitHub MCP `list_releases` tool:
```
List releases to check deployment activity
```

#### 2.6 Repository Health Indicators
Assess repository health:
- **Active Development**: Recent commits, PRs, and issues
- **Community Engagement**: External contributions, issue discussions
- **Maintenance Status**: Response to issues/PRs, dependency updates
- **Documentation**: README quality, up-to-date docs
- **Dependencies**: Outdated dependencies, security alerts

### Step 3: Determine True Status

Based on your research, classify each repository:

1. **Truly Stale**: No meaningful activity, should be archived
   - No commits in 365+ days
   - No open issues or PRs requiring attention
   - No ongoing projects or roadmap items
   - No active community engagement

2. **Low Activity but Active**: Slow-moving but not abandoned
   - Occasional commits or maintenance
   - Responsive to critical issues
   - Stable mature project with low change rate

3. **False Positive**: Appears stale but actually active
   - Activity in other branches
   - External development (forks, dependent projects)
   - Strategic repository (documentation, templates)
   - Recently migrated or reorganized

4. **Requires Attention**: Active but needs maintenance
   - Outstanding security issues
   - Outdated dependencies
   - Unanswered issues or PRs

### Step 4: Create Detailed Issue Reports

For each repository classified as **Truly Stale** or **Requires Attention**, create an issue with:

**Issue Title Format**: `[Stale Repository] <repository-name> - <status>`

**Issue Body Template**:
```markdown
## Repository Analysis: [Repository Name]

**Repository URL**: [repository URL]
**Last Activity**: [date]
**Classification**: [Truly Stale / Requires Attention]
**Workflow Run ID**: ${{ github.run_id }}

### 📊 Activity Summary

#### Commits
- **Last Commit**: [date] by [author]
- **Commits (Last Year)**: [count]
- **Contributors (Last Year)**: [count]
- **Activity Trend**: [Declining / Stopped / Sporadic]

#### Issues
- **Open Issues**: [count]
- **Closed Issues (Last 6mo)**: [count]
- **Recent Issue Activity**: [Yes/No - describe]
- **Issues Needing Attention**: [list or "None"]

#### Pull Requests
- **Open PRs**: [count]
- **Merged PRs (Last 6mo)**: [count]
- **Outstanding PRs**: [list or "None"]

#### Releases
- **Last Release**: [date and version] or [No releases]
- **Release Frequency**: [describe pattern]

### 🔍 Deep Analysis

[Provide 2-3 paragraphs analyzing:
- What the repository was used for
- Why activity stopped or declined
- Current state and relevance
- Any dependencies or downstream impacts
- Community engagement patterns]

### 💡 Recommendation

**Action**: [Archive / Maintain / Investigate Further / Transfer Ownership]

**Reasoning**: [Explain why this recommendation makes sense based on the analysis]

**Impact**: [Describe what happens if this recommendation is followed]

### ⚠️ Important Considerations

[List any concerns, blockers, or things to consider before taking action:
- Outstanding issues or PRs
- Active forks or dependencies
- Documentation or historical value
- Team ownership or handoff needs]

### 📋 Next Steps

- [ ] Review this analysis
- [ ] Contact repository owner/team
- [ ] [Specific action based on recommendation]
- [ ] Update repository topics/status
- [ ] [Additional steps as needed]

---
*This analysis was generated by the Stale Repository Identifier workflow. Please verify findings before taking any archival actions.*
```

### Step 5: Summary Report

After analyzing all repositories, provide a summary to stdout (not as an issue):

```
## Stale Repository Analysis Summary

**Total Repositories Analyzed**: [count]

**Classification Breakdown**:
- Truly Stale: [count]
- Low Activity but Active: [count]
- False Positives: [count]
- Requires Attention: [count]

**Issues Created**: [count]

**Key Findings**:
[Brief summary of overall patterns and insights]
```

## Important Guidelines

1. **Be Thorough**: Use multiple data points (commits, issues, PRs, releases) to make accurate assessments
2. **Be Conservative**: When in doubt, classify as "Low Activity" rather than "Truly Stale"
3. **Provide Evidence**: Include specific dates, counts, and examples in reports
4. **Respect Limits**: Maximum 10 issues per run to avoid overwhelming maintainers
5. **Context Matters**: Consider repository purpose (documentation, templates, etc.)
6. **Focus on Value**: Prioritize repositories that are truly abandoned vs. intentionally stable

## Rate Limiting

To avoid GitHub API rate limits:
- Batch API calls when possible
- Add small delays between repositories if needed
- If you hit rate limits, note which repositories couldn't be analyzed

## Output

- Create GitHub issues for repositories needing attention (max 10)
- Print summary statistics to stdout
- Be clear and actionable in recommendations
