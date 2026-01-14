---
title: ProjectOps
description: Automate GitHub Projects board management with AI-powered workflows (triage, routing, and field updates)
sidebar:
  badge: { text: 'Event-triggered', variant: 'success' }
---

ProjectOps keeps [GitHub Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/learning-about-projects/about-projects) up to date using AI.

When a new issue or pull request arrives, the agent reads it and decides where it belongs, what status to start in, and which fields to set (priority, effort, etc.).

Then the [`update-project`](/gh-aw/reference/safe-outputs/#project-board-updates-update-project) safe output applies those choices in a separate, scoped job—the agent job never sees the Projects token so everything remains secure.

## Prerequisites

1. **Create a Project**: Before you wire up a workflow, you must first create the Project in the GitHub UI (user or organization level). Keep the Project URL handy (you'll need to reference it in your workflow instructions).

2. **Create a token**: The kind of token you need depends on whether the Project you created is **user-owned** or **organization-owned**.

#### User-owned Projects (v2)

Use a **classic PAT** with scopes:
- `project` (required for user Projects)
- `repo` (required if accessing private repositories)

#### Organization-owned Projects (v2)

Use a **fine-grained** PAT with scopes:
- Repository access: Select specific repos that will use the workflow
- Repository permissions:
  - Contents: Read
  - Issues: Read (if workflow is triggered by issues)
  - Pull requests: Read (if workflow is triggered by pull requests)
- Organization permissions:
  - Projects: Read & Write (required for updating projects)

### 3) Store the token as a secret

After creating your token, add it to your repository:

```bash
gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "YOUR_PROJECT_TOKEN"
```

See the [GitHub Projects v2 token reference](/gh-aw/reference/tokens/#gh_aw_project_github_token-github-projects-v2) for complete details.

## When to Use ProjectOps

ProjectOps complements [GitHub's built-in Projects automation](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-built-in-automations) with AI-powered intelligence:

- **Content-based routing** - Analyze issue content to determine which project board and what priority (native automation only supports label/status triggers)
- **Multi-issue coordination** - Add a set of related issues/PRs to an existing initiative project and apply consistent tracking labels
- **Dynamic field assignment** - Set priority, effort, and custom fields based on AI analysis of issue content

## How It Works

While GitHub's native project automation can move items based on status changes and labels, ProjectOps adds **AI-powered content analysis** to determine routing and field values. The AI agent reads the issue description, understands its type and priority, and makes intelligent decisions about project assignment and field values.

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
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
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
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

The `update-project` tool provides intelligent project management:

- **Update-only**: Does not create Projects (create the Project in the GitHub UI first)
- **Auto-adds items**: Checks if issue/PR is already on the board before adding (prevents duplicates)
- **Updates fields**: Sets status, priority, and other custom fields
- **Applies a tracking label**: When adding a new item, it can apply a consistent tracking label to the underlying issue/PR
- **Returns outputs**: Exposes the Project item ID (`item-id`) for downstream steps

## Organization-Owned Project Configuration

For workflows that interact with organization-owned projects and need to query GitHub information, use the following configuration:

```yaml wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
---

# Smart Issue Triage for Organization Project

Analyze the issue and add it to the organization project board...
```

This configuration ensures:
1. The GitHub Model Context Protocol (MCP) toolset can query repository and project information
2. The `update-project` safe output can modify the organization project
3. Both operations use the same token with appropriate permissions

## Accessing Issue Context

ProjectOps workflows can access sanitized issue content through the `needs.activation.outputs.text` variable, which combines the issue title and description while removing security risks:

```yaml wrap
# In your workflow instructions:
Analyze this issue to determine priority: "${{ needs.activation.outputs.text }}"
```

**Security Note**: Always treat user content as potentially untrusted and design workflows to be resilient against prompt injection attempts.


## Project Management Features

The `update-project` safe output provides intelligent automation:

- **Update-only** - Expects the Project to already exist (creates no Projects)
- **Duplicate prevention** - Checks if issue already on board before adding
- **Custom field support** - Set status, priority, effort, sprint, team, or any custom fields
- **Tracking** - Can apply a consistent tracking label when adding new items
- **Cross-repo support** - Works with organization-level projects spanning multiple repositories
- **Automatic view creation** - Configure project views directly in workflow frontmatter

## Creating Project Views

Project views can be created automatically by declaring them in the `views` array. Views are created when the workflow runs, after processing update_project items from the agent.

### View Configuration

Views are configured in workflow frontmatter using the `views` property:

```yaml wrap
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    views:
      - name: "Sprint Board"
        layout: board
        filter: "is:issue is:open"
      - name: "Task Tracker"  
        layout: table
        filter: "is:issue,is:pull_request"
      - name: "Timeline"
        layout: roadmap
```

**View properties:**

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | View name (e.g., "Sprint Board", "Task Tracker") |
| `layout` | string | Yes | View layout: `table`, `board`, or `roadmap` |
| `filter` | string | No | Filter query (e.g., `is:issue is:open`, `label:bug`) |

**Layout types:**
- **`table`** — List view with customizable columns for detailed tracking
- **`board`** — Kanban-style cards grouped by status or custom field
- **`roadmap`** — Timeline visualization with date-based swimlanes

**Filter syntax examples:**
- `is:issue is:open` — Open issues only
- `is:pull_request` — Pull requests only  
- `is:issue,is:pull_request` — Both issues and PRs
- `label:bug` — Items with bug label
- `assignee:@me` — Items assigned to viewer

### View Creation Examples

**Bug Triage Board:**
```yaml wrap
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    views:
      - name: "Triage Board"
        layout: board
        filter: "is:issue label:bug"
      - name: "Bug List"
        layout: table
        filter: "is:issue label:bug is:open"
```

**Feature Planning:**
```yaml wrap
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    views:
      - name: "Feature Roadmap"
        layout: roadmap
        filter: "is:issue label:enhancement"
      - name: "Feature Backlog"
        layout: table
        filter: "is:issue label:enhancement"
```

**Sprint Management:**
```yaml wrap
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    views:
      - name: "Current Sprint"
        layout: board
        filter: "is:issue,is:pull_request is:open"
      - name: "Sprint Timeline"
        layout: roadmap
      - name: "All Items"
        layout: table
```

Views are created automatically during workflow execution. The workflow must include at least one `update_project` operation to provide the target project URL.

## Cross-Repository Considerations

Project boards can span multiple repositories, but the `update-project` tool operates on the current repository's context. To manage cross-repository projects:

1. Use organization-level projects accessible from all repositories
2. Ensure the workflow's GitHub token has `projects: write` permission
3. Consider using a PAT for broader access across repositories

## Best Practices

**Use descriptive project names** that clearly indicate purpose and scope. Prefer "Performance Optimization Q1 2025" over "Project 1".

**Leverage a tracking label** for grouping related work across issues and PRs.

**Set meaningful field values** like status, priority, and effort to enable effective filtering and sorting on boards.

**Combine with issue creation** for initiative workflows that generate multiple tracked tasks automatically.

**Update status progressively** as work moves through stages (Todo → In Progress → In Review → Done).

**Archive completed initiatives** rather than deleting them to preserve historical context and learnings.

## Common Challenges

**Permission Errors**: Project operations require `projects: write` permission. For organization-level projects, a PAT may be needed.

**Field Name Mismatches**: Custom field names are case-sensitive. Use exact field names as defined in the project settings.

**Cross-Repo Limitations**: The tool operates in the context of the triggering repository. Use organization-level projects for multi-repo tracking.

**Token Scope**: Default `GITHUB_TOKEN` may have limited project access. Use a PAT stored in secrets for broader permissions.

## Additional Resources

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration
- [Update Project API](/gh-aw/reference/safe-outputs/#project-board-updates-update-project) - Detailed API reference
- [Trigger Events](/gh-aw/reference/triggers/) - Event trigger configuration
- [IssueOps Guide](/gh-aw/examples/issue-pr-events/issueops/) - Related issue automation patterns
