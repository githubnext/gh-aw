---
title: Cross-Repository Issue Tracking
description: Centralize issue tracking across multiple repositories with automated tracking issue creation and status synchronization.
sidebar:
  badge: { text: 'Multi-Repo', variant: 'note' }
---

Cross-repository issue tracking enables organizations to maintain a centralized view of work across multiple component repositories. When issues are created in component repos, tracking issues are automatically created in a central repository, providing visibility without requiring direct access to all repositories.

## When to Use

- **Component-based architecture** - Track work across microservices or component repositories
- **Multi-team coordination** - Centralize visibility for issues spanning multiple teams
- **External dependencies** - Track upstream issues that affect your projects
- **Cross-project initiatives** - Coordinate work that touches multiple repositories
- **Reporting aggregation** - Collect metrics and status from distributed repositories

## How It Works

Workflows in component repositories create tracking issues in a central repository when local issues are opened, updated, or closed. The central repository maintains references to all component issues, enabling organization-wide visibility and reporting.

## Basic Tracking Issue Creation

Create tracking issues in central repository when component issues are opened:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "myorg/central-tracker"
    title-prefix: "[component-alpha] "
    labels: [from-component-alpha, tracking-issue]
---

# Create Central Tracking Issue

When an issue is opened in component-alpha, create a corresponding
tracking issue in the central tracker.

**Original issue:** ${{ github.event.issue.html_url }}
**Issue number:** ${{ github.event.issue.number }}
**Content:** "${{ needs.activation.outputs.text }}"

**Tracking issue should include:**
- Link to original issue in component repo
- Component identifier clearly marked
- Summary of the issue with key details
- Suggested priority based on labels from original
- Any cross-component dependencies identified

**Labels to apply:**
- `from-component-alpha` (component identifier)
- `tracking-issue` (marks as tracking issue)
- Priority label based on original issue labels
```

## Status Synchronization

Update tracking issues when component issues change status:

```aw wrap
---
on:
  issues:
    types: [closed, reopened, labeled, unlabeled]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  add-comment:
    target-repo: "myorg/central-tracker"
    target: "*"  # Find related tracking issue
---

# Update Central Tracking Issue Status

When this component issue changes status, update the central tracking issue.

**Original issue:** ${{ github.event.issue.html_url }}
**Action:** ${{ github.event.action }}

**Process:**
1. Search for tracking issue in `myorg/central-tracker` that references this issue URL
2. Add comment with status update:
   - If closed: "✅ Component issue resolved"
   - If reopened: "🔄 Component issue reopened"
   - If labeled: "🏷️ Labels updated: [list]"
   - If unlabeled: "🏷️ Label removed: [name]"
3. Include link back to component issue
4. Add timestamp for tracking

**Comment format:**
```
**Status Update from component-alpha**
Action: [closed|reopened|labeled|unlabeled]
Issue: [link to component issue]
Time: [timestamp]
Details: [specific changes]
```
```

## Multi-Component Tracking

Track issues that span multiple component repositories:

```aw wrap
---
on:
  issues:
    types: [opened]
    # Triggered when issue has 'cross-component' label
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    max: 3  # May create issues in multiple tracking repos
    target-repo: "myorg/central-tracker"
    title-prefix: "[cross-component] "
    labels: [cross-component, needs-coordination]
---

# Track Cross-Component Issues

When an issue is marked as cross-component, create coordinated tracking issues.

**Original issue:** ${{ github.event.issue.html_url }}

**Analysis:**
1. Identify which components are affected (check issue description)
2. Create primary tracking issue in central tracker
3. If specific components mentioned, also create child issues in those repos

**Tracking issue should include:**
- Clear description of cross-component nature
- List of affected components
- Coordination requirements
- Suggested approach for resolution
- Links to all related issues

**Follow-up:**
- Tag relevant team leads
- Set up coordination meeting if high priority
```

## External Dependency Tracking

Track issues from external/upstream repositories:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      external_issue_url:
        description: 'URL of external issue to track'
        required: true
        type: string
permissions:
  contents: read
tools:
  github:
    toolsets: [issues]
  web-fetch:
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "myorg/dependency-tracker"
    title-prefix: "[upstream] "
    labels: [external-dependency, upstream-issue]
---

# Track External Dependency Issue

Create tracking issue for external dependency problem.

**External issue URL:** ${{ github.event.inputs.external_issue_url }}

**Process:**
1. Fetch issue details from external URL
2. Extract issue title, description, and current status
3. Identify which internal projects are affected
4. Create tracking issue in dependency tracker

**Tracking issue should include:**
- Link to external issue
- External project/repository name
- Current status of external issue
- Impact assessment on our projects
- List of affected internal repositories
- Workaround status (if applicable)
- Monitoring plan

**Follow-up actions:**
- Set reminder to check external issue weekly
- Notify affected project teams
```

## Automated Triage and Routing

Triage component issues and route to appropriate trackers:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    max: 2
    title-prefix: "[auto-triaged] "
---

# Triage and Route to Tracking Repos

Analyze new issues and create tracking issues in appropriate repositories.

**Original issue:** ${{ github.event.issue.html_url }}
**Content:** "${{ needs.activation.outputs.text }}"

**Triage process:**
1. Analyze issue content for severity and type
2. Determine target tracking repository:
   - Security issues → `myorg/security-tracker`
   - Feature requests → `myorg/feature-tracker`
   - Bug reports → `myorg/bug-tracker`
   - Infrastructure → `myorg/ops-tracker`

3. Create tracking issue in appropriate repo with:
   - Severity assessment
   - Recommended priority
   - Affected components
   - Suggested owner/team

**Include in tracking issue:**
- Original issue link
- Triage reasoning
- Recommended next steps
- SLA targets based on severity
```

## Aggregated Reporting

Create weekly summary of tracked issues:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM
permissions:
  contents: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-discussion:
    target-repo: "myorg/central-tracker"
    category: "Status Reports"
    title-prefix: "[weekly] "
---

# Weekly Cross-Repo Issue Summary

Generate weekly summary of tracked issues across all component repositories.

**Repositories to summarize:**
- `myorg/component-alpha`
- `myorg/component-beta`
- `myorg/component-gamma`
- `myorg/shared-library`

**For each repository:**
1. Count open issues by priority
2. List issues opened this week
3. List issues closed this week
4. Identify stale issues (>30 days no activity)
5. Highlight blockers or critical issues

**Create discussion with:**
- Executive summary (key metrics)
- Per-repository breakdown
- Trending analysis (increasing/decreasing activity)
- Action items requiring attention
- Links to individual tracking issues

**Format as markdown table with status indicators**
```

## Bidirectional Linking

Maintain references between component and tracking issues:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  actions: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "myorg/central-tracker"
    title-prefix: "[linked] "
  add-comment:
    max: 1
---

# Create Tracking Issue with Bidirectional Links

Create tracking issue and add comment to original with link.

**Original issue:** ${{ github.event.issue.html_url }}

**Process:**
1. Create tracking issue in `myorg/central-tracker`
2. Add comment to original issue with tracking issue link
3. Include reference syntax for automatic linking

**Tracking issue creation:**
- Title: "[linked] ${{ github.event.issue.title }}"
- Body includes: "Tracks ${{ github.event.issue.html_url }}"
- Labels: component identifier + tracking-issue

**Comment on original issue:**
```
🔗 **Tracking Issue Created**

This issue is being tracked in the central repository:
[Tracking Issue #XXX](https://github.com/myorg/central-tracker/issues/XXX)

Updates to this issue will be reflected in the tracking issue.
```

**Benefits:**
- Easy navigation between related issues
- GitHub automatic reference detection
- Clear audit trail
```

## Priority-Based Routing

Route issues to different trackers based on priority:

```aw wrap
---
on:
  issues:
    types: [opened, labeled]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [issues]
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    max: 1
    title-prefix: "[priority-routed] "
---

# Route Issues Based on Priority

Route issues to appropriate tracking repository based on priority level.

**Original issue:** ${{ github.event.issue.html_url }}
**Labels:** Check for priority labels (P0, P1, P2, P3)

**Routing logic:**
- P0 (Critical): → `myorg/incidents`
- P1 (High): → `myorg/priority-tracker`
- P2 (Medium): → `myorg/central-tracker`
- P3 (Low): → `myorg/backlog`

**Tracking issue includes:**
- Original issue link and content
- Priority level clearly marked
- SLA expectations based on priority
- Escalation path
- Required approvers/reviewers

**For P0 (Critical):**
- Alert on-call team
- Set immediate response SLA
- Include incident response checklist
```

## Authentication Setup

Cross-repo issue tracking requires appropriate authentication:

### PAT Configuration

```bash
# Create PAT with issues and repository read permissions
gh secret set CROSS_REPO_PAT --body "ghp_your_token_here"
```

**Required Permissions:**
- `repo` (for private repositories)
- `public_repo` (for public repositories)

### GitHub App Configuration

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.TRACKER_APP_ID }}
    private-key: ${{ secrets.TRACKER_APP_PRIVATE_KEY }}
    repositories: ["central-tracker", "security-tracker", "feature-tracker"]
  create-issue:
    target-repo: "myorg/central-tracker"
```

## Best Practices

### Issue Organization

1. **Use consistent prefixes** - Identify component source in tracking issue titles
2. **Apply component labels** - Enable filtering by source component
3. **Include references** - Always link back to original issues
4. **Standardize format** - Use templates for tracking issue descriptions

### Status Management

1. **Automate status sync** - Update tracking issues when component issues change
2. **Set up webhooks** - Consider webhook-based updates for real-time sync
3. **Handle closures** - Update tracking issue when component issue closes
4. **Track reopens** - Alert when issues are reopened

### Search and Discovery

1. **Use GitHub issue search** - Leverage search to find related tracking issues
2. **Implement tagging strategy** - Consistent labels for cross-repo queries
3. **Create saved searches** - Share common queries across team
4. **Document conventions** - Clear documentation of tracking patterns

### Scaling Considerations

1. **Rate limiting** - Space out tracking issue creation to avoid rate limits
2. **Batch updates** - Group status updates when possible
3. **Archive old trackers** - Close stale tracking issues periodically
4. **Monitor volume** - Alert on unusual tracking issue creation rates

## Related Documentation

- [Multi-Repo Operations Guide](/gh-aw/guides/multi-repo-ops/) - Complete multi-repo overview
- [Feature Synchronization](/gh-aw/examples/multi-repo/feature-sync/) - Code sync patterns
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Issue creation configuration
- [GitHub Tools](/gh-aw/reference/tools/#github-tools-github) - API access configuration
