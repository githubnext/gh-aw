---
title: ProjectOps
description: Automate GitHub Projects board management - organize work, track campaigns, and maintain project state with AI-powered workflows
sidebar:
  badge: { text: 'Event-triggered', variant: 'success' }
---

ProjectOps brings intelligent automation to GitHub Projects, enabling AI agents to automatically create project boards, add items, update status fields, and track campaigns. GitHub Agentic Workflows makes ProjectOps natural through the [`update-project`](/gh-aw/safe-outputs/update-project) safe output that handles all [Projects v2 API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) complexity while maintaining security with minimal permissions.

## When to Use ProjectOps

ProjectOps complements [GitHub Projects' built-in automation](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-built-in-automations) with AI-powered intelligence:

- **Content-based routing** - Analyze issue content to determine which project board and what priority (native automation only supports label/status triggers)
- **Multi-issue coordination** - Create campaign boards with multiple issues and apply campaign labels automatically
- **Dynamic field assignment** - Set priority, effort, and custom fields based on AI analysis of issue content

## How It Works

While GitHub Projects' native automation can move items based on status changes and labels, ProjectOps adds **AI-powered content analysis** to determine routing and field values. The AI agent reads the issue description, understands its type and priority, and makes intelligent decisions about project assignment and field values.

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  update-project:
    max: 1
  add-comment:
    max: 1
---

# Smart Issue Triage with Project Tracking

When a new issue is created, analyze it and add to the appropriate project board.

Examine the issue title and description to determine its type:
- Bug reports → Add to "Bug Triage" project, status: "Needs Triage", priority: based on severity
- Feature requests → Add to "Feature Roadmap" project, status: "Proposed"
- Documentation issues → Add to "Docs Improvements" project, status: "Todo"
- Performance issues → Add to "Performance Optimization" project, priority: "High"

After adding to project board, comment on the issue confirming where it was added.
```

This workflow creates an intelligent triage system that automatically organizes new issues onto appropriate project boards with relevant status and priority fields.

## Safe Output Architecture

ProjectOps workflows use the `update-project` safe output to ensure secure project management with minimal permissions. The main job runs with `contents: read` while project operations happen in a separate job with `projects: write` permissions:

```yaml wrap
safe-outputs:
  update-project:
    max: 10                              # Optional: max project operations (default: 10)
    github-token: ${{ secrets.PROJECTS_PAT }}  # Optional: PAT for cross-repo projects
```

The `update-project` tool provides intelligent project management:
- **Auto-creates boards**: Creates project if it doesn't exist
- **Auto-adds items**: Checks if issue already on board before adding (prevents duplicates)
- **Updates fields**: Sets status, priority, custom fields
- **Applies campaign labels**: Adds `campaign:<id>` label for tracking
- **Returns metadata**: Provides campaign ID, project ID, and item ID as outputs

## Accessing Issue Context

ProjectOps workflows can access sanitized issue content through the `needs.activation.outputs.text` variable, which combines the issue title and description while removing security risks:

```yaml wrap
# In your workflow instructions:
Analyze this issue to determine priority: "${{ needs.activation.outputs.text }}"
```

**Security Note**: Always treat user content as potentially untrusted and design workflows to be resilient against prompt injection attempts.

## Common ProjectOps Patterns

### Campaign Launch and Tracking

Create a project board for a focused initiative and add all related issues with tracking metadata.

**This goes beyond native GitHub automation** by analyzing the codebase to generate campaign issues and coordinating multiple related work items.

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      campaign_name:
        description: "Campaign name"
        required: true
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
    max: 20
  update-project:
    max: 20
---

# Launch Campaign

Create a new campaign project board: "{{inputs.campaign_name}}"

Analyze the repository to identify tasks needed for this campaign.

For each task:
1. Create an issue with detailed description
2. Add the issue to the campaign project board
3. Set status to "Todo"
4. Set priority based on impact
5. Apply campaign label for tracking

The campaign board provides a visual dashboard showing all related work.
```

See the [Campaign Workflows Guide](/gh-aw/guides/campaigns/) for comprehensive campaign patterns and coordination strategies.

### Content-Based Priority Assignment

Analyze issue content to set priority automatically, going beyond what labels can provide:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  update-project:
    max: 1
---

# Intelligent Priority Triage

When an issue is created, analyze its content to set priority and effort.

Analyze the issue description for:
- Security vulnerabilities → Priority: "Critical", add to "Security" project
- Production crashes or data loss → Priority: "High", Effort: "Medium"
- Performance degradation → Priority: "High", Effort: "Large"
- Minor bugs or improvements → Priority: "Low", Effort: "Small"

Add to "Engineering Backlog" project with calculated priority and effort fields.
```

**Why use ProjectOps:** Native GitHub automation can't analyze issue content to determine priority - it only reacts to labels and status changes.



## Project Management Features

The `update-project` safe output provides intelligent automation:

- **Auto-creates boards** - Creates project if it doesn't exist, finds existing boards automatically
- **Duplicate prevention** - Checks if issue already on board before adding
- **Custom field support** - Set status, priority, effort, sprint, team, or any custom fields
- **Campaign tracking** - Auto-generates campaign IDs, applies labels, stores metadata
- **Cross-repo support** - Works with organization-level projects spanning multiple repositories

## Cross-Repository Considerations

Project boards can span multiple repositories, but the `update-project` tool operates on the current repository's context. To manage cross-repository projects:

1. Use organization-level projects accessible from all repositories
2. Ensure the workflow's GitHub token has `projects: write` permission
3. Consider using a PAT for broader access across repositories

## Best Practices

**Use descriptive project names** that clearly indicate purpose and scope. Prefer "Performance Optimization Q1 2025" over "Project 1".

**Leverage campaign IDs** for tracking related work across issues and PRs. Query by campaign label for reporting and metrics.

**Set meaningful field values** like status, priority, and effort to enable effective filtering and sorting on boards.

**Combine with issue creation** for campaign workflows that generate multiple tracked tasks automatically.

**Update status progressively** as work moves through stages (Todo → In Progress → In Review → Done).

**Archive completed campaigns** rather than deleting them to preserve historical context and learnings.

## Common Challenges

**Permission Errors**: Project operations require `projects: write` permission. For organization-level projects, a PAT may be needed.

**Field Name Mismatches**: Custom field names are case-sensitive. Use exact field names as defined in the project settings.

**Cross-Repo Limitations**: The tool operates in the context of the triggering repository. Use organization-level projects for multi-repo tracking.

**Token Scope**: Default `GITHUB_TOKEN` may have limited project access. Use a PAT stored in secrets for broader permissions.

## Additional Resources

- [Campaign Workflows Guide](/gh-aw/guides/campaigns/) - Comprehensive campaign pattern documentation
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [Update Project API](/gh-aw/reference/safe-outputs/#project-board-updates-update-project) - Detailed API reference
- [Trigger Events](/gh-aw/reference/triggers/) - Event trigger configuration
- [IssueOps Guide](/gh-aw/examples/issue-pr-events/issueops/) - Related issue automation patterns
