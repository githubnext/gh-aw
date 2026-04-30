---
title: Project Tracking
description: Automatically track issues and pull requests in GitHub Projects boards using safe-outputs configuration
sidebar:
  badge: { text: 'Project', variant: 'tip' }
---

Use `update-project` and `create-project-status-update` safe-outputs to automatically track workflow items in GitHub Projects boards, with support for field updates and status reporting.

## Quick Start

Add project configuration to your workflow's `safe-outputs` section:

```yaml
---
on:
  issues:
    types: [opened]
safe-outputs:
  create-issue:
    max: 3
  update-project:
    project: https://github.com/orgs/github/projects/123
    max: 10
  create-project-status-update:
    project: https://github.com/orgs/github/projects/123
    max: 1
---
```

## Configuration

### Update Project Configuration

Configure `update-project` in the `safe-outputs` section:

```yaml
safe-outputs:
  update-project:
    project: https://github.com/orgs/github/projects/123  # Default project URL
    max: 20                                                # Max operations per run (default: 10)
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    views:                                                 # Optional: auto-create views
      - name: "Sprint Board"
        layout: board
        filter: "is:issue is:open"
      - name: "Task Tracker"
        layout: table
```

Supported operations: `add` (add items), `update` (update fields), `create_fields` (custom fields), `create_views` (project views).

### Project Status Update Configuration

Configure `create-project-status-update` to post project status updates:

```yaml
safe-outputs:
  create-project-status-update:
    project: https://github.com/orgs/github/projects/123  # Default project URL
    max: 1                                                 # Max status updates per run (default: 1)
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

See [Safe Outputs: Project Board Updates](/gh-aw/reference/safe-outputs/#project-board-updates-update-project) for complete configuration details.

### Authentication

See [Project token authentication](/gh-aw/patterns/project-ops/#project-token-authentication).

## Example: Issue Triage

Automatically add new issues to a project board with intelligent categorization:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
  issues: read
tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
safe-outputs:
  add-comment:
    max: 1
  update-project:
    project: https://github.com/orgs/myorg/projects/1
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
---

# Smart Issue Triage

When a new issue is created, analyze it and add to the project board.

## Task

Examine the issue title and description to determine its type:
- **Bug reports** → Add to project, set status="Needs Triage", priority="High"
- **Feature requests** → Add to project, set status="Backlog", priority="Medium"
- **Documentation** → Add to project, set status="Todo", priority="Low"

After adding to the project board, comment on the issue confirming where it was added.
```

## Example: Pull Request Tracking

Track pull requests through the development workflow:

```aw wrap
---
on:
  pull_request:
    types: [opened, review_requested]
permissions:
  contents: read
  actions: read
  pull-requests: read
tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
safe-outputs:
  update-project:
    project: https://github.com/orgs/myorg/projects/2
    max: 5
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
---

# PR Project Tracker

Track pull requests in the development project board.

## Task

When a pull request is opened or reviews are requested:
1. Add the PR to the project board
2. Set status based on PR state:
   - Just opened → "In Progress"
   - Reviews requested → "In Review"
3. Set priority based on PR labels:
   - Has "urgent" label → "High"
   - Has "enhancement" label → "Medium"
   - Default → "Low"
```

## Common Patterns

### Progressive Status Updates

```aw
Analyze the issue and determine its current state:
- If new and unreviewed → status="Needs Triage"
- If reviewed and accepted → status="Todo"
- If work started → status="In Progress"
- If PR merged → status="Done"

Update the project item with the appropriate status.
```

### Priority Assignment

```aw
Examine the issue for urgency indicators:
- Contains "critical", "urgent", "blocker" → priority="High"
- Contains "important", "soon" → priority="Medium"
- Default → priority="Low"

Update the project item with the assigned priority.
```

### Field-Based Routing

```aw
Determine the team that should handle this issue:
- Security-related → team="Security"
- UI/UX changes → team="Design"
- API changes → team="Backend"
- Default → team="General"

Update the project item with the team field.
```

## Troubleshooting

| Issue | Symptom | Solution |
|-------|---------|----------|
| Items not added | Items don't appear despite a successful run | Verify project URL, confirm token has Projects: Read & Write permission, review `safe_outputs` job logs |
| Permission errors | "Resource not accessible" or "Insufficient permissions" | Use fine-grained PAT with org Projects permission, or classic PAT with `project` scope; verify secret name and repository settings |
| Token not resolved | "invalid token" or literal `${{...}}` string in output | Use `${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}` syntax without quotes; check secret name is exact (case-sensitive) |

## See Also

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe-outputs documentation
- [Project token authentication](/gh-aw/patterns/project-ops/#project-token-authentication) - Token setup guide
- [Projects & Monitoring](/gh-aw/patterns/monitoring/) - Design pattern guide
- [Orchestration](/gh-aw/patterns/orchestration/) - Design pattern guide
