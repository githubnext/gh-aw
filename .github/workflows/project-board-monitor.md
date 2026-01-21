---
on:
  issues:
    types: [opened, reopened, labeled, closed]
  pull_request:
    types: [opened, reopened, ready_for_review, labeled, closed]
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tools:
  github:
    toolsets: [default, projects]
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
safe-outputs:
  update-project:
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
  add-comment:
    max: 1
---

# Project Board Monitor - Automated ProjectOps

Automatically track and update GitHub Projects v2 boards based on issue and PR activity.

This is a **monitoring workflow** (ProjectOps) - it passively tracks existing issues/PRs
and updates project boards without creating new work or orchestrating other workflows.

For active orchestration of worker workflows toward goals, use campaigns instead.

## What This Workflow Does

When an issue or pull request is opened, reopened, labeled, or closed:

1. **Analyzes content** using AI to determine appropriate routing and classification
2. **Updates project boards** automatically based on content analysis  
3. **Sets project fields** intelligently:
   - **Status**: Based on item state (Backlog, To Do, In Progress, In Review, Done)
   - **Priority**: Determined from severity indicators and labels (High/Medium/Low)
   - **Size/Effort**: Estimated based on scope and complexity
   - **Type**: Classified as bug, feature, documentation, etc.
4. **Posts confirmation** comment briefly noting where the item was added

## Project Configuration

Before using this workflow, you need to configure the project URL below.

**Replace the placeholder** with your actual GitHub Project URL:
- **User project**: `https://github.com/users/USERNAME/projects/PROJECT_NUMBER`
- **Organization project**: `https://github.com/orgs/ORG/projects/PROJECT_NUMBER`

**Example configuration:**
```
PROJECT_URL="https://github.com/orgs/myorg/projects/42"
```

## Routing Logic

Customize the routing logic below to match your project structure. The AI will analyze
issue/PR content and apply these rules:

### Issue Routing Examples

- **Bug reports** (indicated by "bug", "error", "crash", "broken" in title/body)
  → Add to project, set status: "Needs Triage", priority: based on severity
  
- **Feature requests** (indicated by "feature", "enhancement", "add support")
  → Add to project, set status: "Proposed", priority: "Medium"
  
- **Documentation issues** (indicated by "docs", "documentation", "readme")
  → Add to project, set status: "Todo", size: "Small"
  
- **Performance issues** (indicated by "slow", "performance", "optimization")
  → Add to project, priority: "High", type: "Performance"

### Pull Request Routing Examples

- **Bug fixes** (title starts with "Fix", links to issue with "Fixes #")
  → Add to project, set status: "In Review", link to related issue
  
- **Features** (title starts with "Add", "Implement", "Feature")
  → Add to project, set status: "In Review", type: "Feature"
  
- **Documentation** (changes only .md files or docs/ directory)
  → Add to project, set status: "In Review", type: "Documentation"
  
- **Refactoring** (title contains "refactor", "cleanup", "improve")
  → Add to project, type: "Technical Debt", size: estimated from files changed

## Intelligence Guidelines

When analyzing issues and PRs to route and classify them:

1. **Read the full context**: Title, body, labels, linked issues, changed files (for PRs)

2. **Determine project and status**:
   - Check if PROJECT_URL is configured above (if not, report error)
   - For new items: Set initial status based on type (e.g., "Backlog", "To Do")
   - For reopened items: Set status to "In Progress" if previously done
   - For closed items: Set status to "Done" or "Cancelled" based on resolution

3. **Set priority intelligently**:
   - **High**: Security issues, critical bugs, production outages, urgent features
   - **Medium**: Standard bugs, planned features, moderate impact
   - **Low**: Nice-to-have features, minor improvements, documentation updates

4. **Estimate size** (for features and enhancements):
   - **Small**: 1-2 days, single file changes, simple additions
   - **Medium**: 3-5 days, multiple files, moderate complexity
   - **Large**: 1+ weeks, architectural changes, complex features

5. **Classify type** based on content:
   - Bug, Feature, Documentation, Performance, Security, Technical Debt, etc.

## After Board Update

After successfully adding or updating the item on the project board, post a **brief** comment
confirming the action taken. Keep it concise and informative.

**Good example:**
```
✅ Added to [Project Name] with status 'To Do' and priority 'High'
```

**Avoid**:
- Long explanations of what the workflow does
- Repeating information already visible in the issue/PR
- Marketing language or unnecessary details

---

## Configuration Required

⚠️ **Before using this workflow, you must:**

1. **Create a GitHub Project** (v2) in your organization or user account
2. **Set the PROJECT_URL** in the routing logic above
3. **Create a GitHub token** with Projects permissions:
   - For user projects: Classic PAT with `project` scope
   - For org projects: Fine-grained PAT with `Projects: Read & Write` permission
4. **Store the token** as a repository secret:
   ```bash
   gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "YOUR_TOKEN"
   ```

See the [GitHub Projects v2 token documentation](https://githubnext.github.io/gh-aw/reference/tokens/#gh_aw_project_github_token-github-projects-v2) for detailed setup instructions.

## Rate Limits

The workflow is configured to update up to **10 items per run** via the `safe-outputs.update-project.max` setting. This prevents overwhelming the GitHub API and keeps board updates manageable.

Adjust the `max` value if you need to process more or fewer items per run.

## Monitoring Only

This is a **monitoring workflow** - it only tracks existing issues/PRs and updates project boards.
It does **not**:
- Create new issues or PRs
- Dispatch other workflows
- Make code changes
- Orchestrate toward a goal

For active orchestration of worker workflows toward measurable objectives, use **campaigns** instead:
- Campaigns actively dispatch and coordinate worker workflows
- Campaigns drive toward explicit goals with KPIs
- Campaigns can build workflows dynamically and adapt strategy

See the [campaigns documentation](https://githubnext.github.io/gh-aw/guides/campaigns/getting-started/) for more information.

---

## Troubleshooting

**Project board not updating?**
- Verify PROJECT_URL is set correctly in the routing logic
- Check that GH_AW_PROJECT_GITHUB_TOKEN secret exists and has correct permissions
- Ensure the token has access to the specified project
- Check workflow run logs for permission errors

**Wrong status/priority being set?**
- Review the routing logic rules above
- Add more specific keywords or patterns to guide classification
- Check for conflicting rules that might override each other

**Too many/too few updates?**
- Adjust `safe-outputs.update-project.max` value
- Modify event triggers to be more/less selective
- Add label-based filtering in the routing logic
