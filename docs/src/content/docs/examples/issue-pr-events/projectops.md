---
title: ProjectOps
description: Automate GitHub Projects board management with AI-powered workflows (triage, routing, and field updates), including monitoring capabilities
sidebar:
  badge: { text: 'Event-triggered', variant: 'success' }
---

ProjectOps automates [GitHub Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/learning-about-projects/about-projects) management using AI-powered workflows.

When a new issue or pull request arrives, the agent analyzes it and determines where it belongs, what status to set, which fields to update (priority, effort, etc.), and whether to create or update project structures.

Safe outputs handle all project operations in separate, scoped jobs with minimal permissions—the agent job never sees the Projects token, ensuring secure automation.

## ProjectOps vs Campaigns

**ProjectOps** is for passive monitoring and board management:
- Track and update project boards based on existing issues/PRs
- Monitor progress without dispatching workflows
- Aggregate metrics from ongoing work
- Keep project boards synchronized with repository state

**Campaigns** are for active orchestration:
- Orchestrate worker workflows toward a goal
- Create new issues/PRs through coordinated workflows
- Make decisions and adapt strategy
- Drive time-bound initiatives with measurable objectives

**Relationship**: Campaigns can use ProjectOps features as components (via tracker-label, discovery), just like they use workflow components. Monitoring is a modular component, not a type of campaign.

See: [Campaigns documentation](/gh-aw/guides/campaigns/getting-started/) for active orchestration.

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

## Example: Smart Issue Triage

This example demonstrates intelligent issue routing to project boards with AI-powered content analysis:

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
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
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

## Pre-Built Component: Project Board Monitor

For a complete, production-ready monitoring solution, use the pre-built **project-board-monitor** component:

```bash
gh aw add githubnext/gh-aw/project-board-monitor
```

This shared component provides comprehensive project board automation out-of-the-box:

**Features:**
- Event-driven triggers for issues and PRs (opened, reopened, labeled, closed)
- AI-powered content analysis for intelligent routing
- Automatic status, priority, and field updates
- Support for multiple project boards and custom routing rules
- Comprehensive embedded documentation

**Quick Setup:**

1. **Add the component**:
   ```bash
   gh aw add githubnext/gh-aw/project-board-monitor
   ```

2. **Configure your project URL** in the workflow file:
   ```markdown
   PROJECT_URL="https://github.com/orgs/myorg/projects/42"
   ```

3. **Set up authentication**:
   ```bash
   gh aw secrets set GH_AW_PROJECT_GITHUB_TOKEN --value "YOUR_TOKEN"
   ```

4. **Compile and deploy**:
   ```bash
   gh aw compile project-board-monitor.md
   git add .github/workflows/project-board-monitor.*
   git commit -m "Add project board monitoring"
   git push
   ```

**Example Use Case: Sprint Board Automation**

The project-board-monitor component can automatically manage a sprint board:

1. **New issues** → Added to "Backlog" with AI-determined priority
2. **Labeled issues** → Moved to appropriate status based on label (e.g., `in-progress` → "In Progress")
3. **Pull requests** → Added to "In Review" with size estimation
4. **Closed items** → Moved to "Done" with completion tracking

**Customization:** Edit the routing logic in the workflow file to match your project structure, field names, and business rules. The component includes detailed examples for common scenarios.

**Updates:** Keep your monitoring workflow current with:
```bash
gh aw update project-board-monitor
```

**Alternative: Custom Generation**

For maximum flexibility, generate a custom monitoring workflow from scratch:

```bash
gh aw monitoring new my-custom-monitor
```

This creates a fully customizable template you can tailor to specific requirements.

## Available Safe Outputs

ProjectOps workflows leverage these safe outputs for project management operations:

### Core Operations

- **[`create-project`](/gh-aw/reference/safe-outputs/#project-creation-create-project)** - Create new GitHub Projects V2 boards with custom configuration
- **[`update-project`](/gh-aw/reference/safe-outputs/#project-board-updates-update-project)** - Add issues/PRs to projects, update fields (status, priority, custom fields), and manage project views
- **[`copy-project`](/gh-aw/reference/safe-outputs/#project-board-copy-copy-project)** - Duplicate project boards with all fields, views, and structure intact
- **[`create-project-status-update`](/gh-aw/reference/safe-outputs/#project-status-updates-create-project-status-update)** - Post status updates to project boards with progress summaries and health indicators

Each safe output operates in a separate job with minimal, scoped permissions. See the [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete configuration options and examples.

## Key Capabilities

**Project Creation and Management**
- Create new Projects V2 boards programmatically
- Copy existing projects to duplicate templates or migrate structures
- Add issues and pull requests to projects with duplicate prevention
- Update project status with automated progress summaries

**Field Management**
- Set status, priority, effort, and sprint fields
- Update custom date fields (start date, end date) for timeline tracking
- Support for TEXT, DATE, NUMBER, ITERATION, and SINGLE_SELECT field types
- Automatic field option creation for single-select fields

**View Configuration**
- Automatically create project views (table, board, roadmap)
- Configure view filters and visible fields
- Support for swimlane grouping by custom fields

**Campaign Integration**
- Automatic tracking label application
- Project status updates with health indicators
- Cross-repository project coordination
- Worker/workflow field population for multi-agent campaigns

See the [Safe Outputs reference](/gh-aw/reference/safe-outputs/#project-board-updates-update-project) for project field and view configuration.

## When to Use ProjectOps

ProjectOps complements [GitHub's built-in Projects automation](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-built-in-automations) with AI-powered intelligence:

- **Content-based routing** - Analyze issue content to determine which project board and what priority (native automation only supports label/status triggers)
- **Multi-issue coordination** - Add related issues/PRs to projects and apply consistent tracking labels
- **Dynamic field assignment** - Set priority, effort, and custom fields based on AI analysis
- **Automated project creation** - Create new project boards programmatically based on campaign needs
- **Status tracking** - Generate automated progress summaries with health indicators
- **Template replication** - Copy existing project structures for new initiatives

## Best Practices

**Create projects programmatically** when launching campaigns to ensure consistent structure and field configuration. Use `create-project` with optional first issue to initialize tracking.

**Use descriptive project names** that clearly indicate purpose and scope. Prefer "Performance Optimization Q1 2026" over "Project 1".

**Leverage tracking labels** (`campaign:<id>`) for grouping related work across issues and PRs, enabling orchestrator discovery.

**Set meaningful field values** like status, priority, and effort to enable effective filtering and sorting on boards.

**Create custom views automatically** using the `views` configuration in frontmatter for consistent board setup across campaigns.

**Post regular status updates** using `create-project-status-update` to keep stakeholders informed of campaign progress and health.

**Duplicate successful templates** with `copy-project` to accelerate new campaign setup and maintain consistency.

**Combine with issue creation** for initiative workflows that generate multiple tracked tasks automatically.

**Update status progressively** as work moves through stages (Todo → In Progress → In Review → Done).

**Archive completed initiatives** rather than deleting them to preserve historical context and learnings.

## Common Challenges

**Permission Errors**: Project operations require `projects: write` permission via a PAT. Default `GITHUB_TOKEN` lacks Projects v2 access.

**Field Name Mismatches**: Custom field names are case-sensitive. Use exact field names as defined in project settings. Field names are automatically normalized (e.g., `story_points` matches `Story Points`).

**Token Scope**: Default `GITHUB_TOKEN` cannot access Projects. Store a PAT with Projects permissions in `GH_AW_PROJECT_GITHUB_TOKEN` secret.

**Project URL Format**: Use full project URLs (e.g., `https://github.com/orgs/myorg/projects/42`), not project numbers alone.

**Field Type Detection**: Ensure field types match expected formats (dates as `YYYY-MM-DD`, numbers as integers, single-select as exact option values).

## Additional Resources

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output configuration and API details
- [Campaign Guides](/gh-aw/guides/campaigns/) - Campaign setup and lifecycle
- [Trigger Events](/gh-aw/reference/triggers/) - Event trigger configuration options
- [IssueOps Guide](/gh-aw/examples/issue-pr-events/issueops/) - Related issue automation patterns
- [Token Reference](/gh-aw/reference/tokens/#gh_aw_project_github_token-github-projects-v2) - GitHub Projects token setup

## Real-World Example: Multi-Team Development Board

Here's a complete example using the project-board-monitor component to automate a multi-team development board:

**Scenario:** You have a project board tracking work across frontend, backend, and DevOps teams. You want automatic routing, prioritization, and status updates.

**Setup:**

1. **Create your project** with custom fields:
   - Status: Backlog | To Do | In Progress | In Review | Done
   - Priority: High | Medium | Low
   - Team: Frontend | Backend | DevOps | Design
   - Size: Small | Medium | Large

2. **Add the monitoring component:**
   ```bash
   gh aw add githubnext/gh-aw/project-board-monitor
   ```

3. **Configure routing logic** in `project-board-monitor.md`:
   ```markdown
   PROJECT_URL="https://github.com/orgs/myorg/projects/123"
   
   ## Routing Rules
   
   ### Issue Routing by Labels
   - Labels contain "frontend" or "ui" or "css" → Set Team: "Frontend", Size: "Small"
   - Labels contain "backend" or "api" or "database" → Set Team: "Backend", Size: "Medium"
   - Labels contain "devops" or "ci" or "deployment" → Set Team: "DevOps", Size: "Small"
   - Labels contain "bug" → Set Priority: "High", Status: "To Do"
   - Labels contain "enhancement" → Set Priority: "Medium", Status: "Backlog"
   
   ### PR Routing by Changed Files
   - Changes to `src/frontend/**` → Set Team: "Frontend", Status: "In Review"
   - Changes to `src/backend/**` → Set Team: "Backend", Status: "In Review"
   - Changes to `.github/workflows/**` → Set Team: "DevOps", Status: "In Review"
   
   ### Size Estimation
   - PRs with 1-10 files changed → Size: "Small"
   - PRs with 11-30 files changed → Size: "Medium"
   - PRs with 30+ files changed → Size: "Large"
   
   ### Status Transitions
   - Issue/PR opened → Status: "Backlog" (unless Priority: High, then "To Do")
   - Label "in-progress" added → Status: "In Progress"
   - PR ready_for_review → Status: "In Review"
   - Issue/PR closed and merged → Status: "Done"
   - Issue/PR closed without merge → Archive (remove from project)
   ```

4. **Deploy:**
   ```bash
   gh aw compile project-board-monitor.md
   git commit -am "Add automated project board monitoring"
   git push
   ```

**Results:**

- **Automatic team routing:** Issues and PRs are instantly assigned to the correct team
- **Smart prioritization:** Bugs get high priority, enhancements get medium priority
- **Size estimation:** PRs are automatically sized based on complexity
- **Status tracking:** Board stays current as work progresses through stages
- **Clean board:** Closed/rejected items are automatically archived

**Monitoring the Monitor:**

Check workflow runs to see the automation in action:
```bash
gh aw logs project-board-monitor
```

**Iterating:**

Adjust routing rules based on team feedback:
```bash
# Edit the workflow
vi .github/workflows/project-board-monitor.md

# Recompile with changes
gh aw compile project-board-monitor.md

# Deploy updates
git commit -am "Refine project board routing rules"
git push
```

This setup eliminates manual project board management while ensuring consistent categorization and tracking across all team work.
