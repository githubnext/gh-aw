---
title: Campaign Workflows
description: Use agentic workflows to plan, execute, and track focused software initiatives with automated project board management and campaign tracking.
---

Campaign workflows enable AI agents to orchestrate focused, time-bounded initiatives by automatically creating project boards, generating tasks, and tracking progress across issues and pull requests.

## Campaigns in Agentic Workflows

A **campaign workflow** is different from a regular task workflow:

| Regular Workflow | Campaign Workflow |
|------------------|-------------------|
| Executes one task | Plans and coordinates multiple tasks |
| Single issue/PR | Creates issues, manages project board |
| Direct action | Strategic orchestration |
| Tactical | Strategic |

**Campaign workflow responsibilities:**
- Analyze codebase/context to identify work needed
- Create GitHub Project board as campaign dashboard
- Generate issues for each task with labels and priorities
- Add all tasks to project board with status tracking
- Return campaign ID for querying and reporting

**Worker workflow responsibilities:**
- Execute individual tasks (triggered by issue labels)
- Update project board status as work progresses
- Reference campaign ID in commits and PRs
- Mark tasks complete when done

## How Campaign Workflows Work

Campaign workflows use two key safe outputs:

```yaml wrap
safe-outputs:
  create-issue: { max: 20 }     # Generate campaign tasks
  update-project: { max: 20 }   # Manage project board
```

### The `update-project` Safe Output

The `update-project` tool provides smart project board management:
- **Auto-creates boards**: Creates if doesn't exist, finds if it does
- **Auto-adds items**: Checks if issue already on board before adding
- **Updates fields**: Sets status, priority, custom fields
- **Returns campaign ID**: Unique identifier for tracking

The agent describes the desired board state, the tool handles all GitHub Projects v2 API complexity.

## Campaign Workflow Example

### Performance Optimization Campaign

**Goal**: Reduce page load time by 30% in 2 weeks

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      performance_target:
        description: "Target improvement percentage"
        default: "30"

engine: copilot

safe-outputs:
  create-issue: { max: 20 }     # Create tasks
  update-project: { max: 20 }   # Manage board
---

# Performance Optimization Campaign

You are managing a performance optimization campaign.

**Goal**: Reduce page load time by {{inputs.performance_target}}% 

**Your tasks**:

1. **Create campaign board**: "Performance Campaign - [Today's Date]"

2. **Analyze current performance**:
   - Review bundle sizes
   - Check critical rendering path
   - Identify slow database queries
   - Look for large images/assets

3. **Create issues for each problem**:
   - Title: Clear description of performance issue
   - Labels: "performance", "campaign"
   - Body: Specific metrics, suggested fixes
   
4. **Add each issue to the campaign board** with:
   - Priority: Critical/High/Medium based on impact
   - Effort: XS/S/M/L based on complexity
   - Status: "To Do"

5. **Track progress** as issues are resolved

The campaign board provides a visual dashboard of all optimization work.
```

### What the Agent Does

1. **Analyzes context**: Reviews codebase for performance bottlenecks
2. **Creates project board**: Establishes campaign dashboard with unique ID
3. **Generates task issues**: One issue per problem with detailed description
4. **Organizes work**: Adds issues to board with priority and effort estimates
5. **Tracks automatically**: Campaign ID links all work together via labels

### What the Team Does

- Reviews generated issues on campaign board
- Assigns issues to team members
- Issues trigger worker workflows when labeled
- Worker workflows execute fixes and update board status
- Campaign board shows real-time progress toward goal

## Campaign Tracking with IDs

Every campaign automatically receives a unique **campaign ID** that links all campaign-related resources together.

### Campaign ID Format

Campaign IDs use a hybrid slug-timestamp format for both readability and uniqueness:

```
[slug]-[timestamp]
```

**Examples:**
- `perf-q1-2025-a3f2b4c8` - Performance Optimization Campaign
- `bug-bash-spring-b9d4e7f1` - Bug Bash Campaign  
- `tech-debt-auth-c2f8a9d3` - Tech Debt Campaign

### How Campaign IDs Work

When creating a campaign board, the `update-project` tool:

1. **Generates campaign ID** from project name if not provided
2. **Stores ID in project description** for reference
3. **Adds campaign label** (`campaign:[id]`) to all issues/PRs added to the board
4. **Returns campaign ID** as output for downstream workflows

### Using Campaign IDs in Workflows

**Automatic generation:**
```javascript
update_project({
  project: "Performance Optimization Q1 2025",
  issue: 123,
  fields: {
    status: "In Progress",
    priority: "High"
  }
  // campaign_id auto-generated from project name
})
```

**Manual specification:**
```javascript
update_project({
  project: "Performance Optimization Q1 2025",
  issue: 123,
  campaign_id: "perf-q1-2025-a3f2b4c8"  // Explicit ID
})
```

### Querying Campaign Work

**Find all issues in a campaign:**
```bash
# Using campaign label
gh issue list --label "campaign:perf-q1-2025-a3f2b4c8"

# Find PRs
gh pr list --label "campaign:perf-q1-2025-a3f2b4c8"
```

**Track campaign metrics:**
```bash
# Count completed tasks
gh issue list --label "campaign:perf-q1-2025-a3f2b4c8" --state closed | wc -l

# View campaign timeline
gh issue list --label "campaign:perf-q1-2025-a3f2b4c8" --json createdAt,closedAt
```

### Benefits of Campaign IDs

| Benefit | Description |
|---------|-------------|
| **Cross-linking** | Connect issues, PRs, and project boards |
| **Reporting** | Query all campaign work by label |
| **History** | Track campaign evolution over time |
| **Uniqueness** | Prevent collisions between similar campaigns |
| **Integration** | Use in external tools and dashboards |

## Campaign Architecture

```
User triggers campaign workflow
         ↓
Agent analyzes codebase/context
         ↓
Agent creates campaign board
         ↓
Agent identifies tasks needed
         ↓
For each task:
  - Create GitHub issue
  - Add to campaign board
  - Set priority/effort/status
         ↓
Issues trigger worker workflows
         ↓
Worker workflows:
  - Execute task (fix bug, optimize code, etc.)
  - Update board status
  - Mark complete
         ↓
Campaign board shows real-time progress
```

## Campaign Workflow Patterns

### Manual Trigger: Launch Campaign on Demand

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      campaign_goal:
        description: "What should this campaign achieve?"
engine: copilot
safe-outputs:
  create-issue: { max: 20 }
  update-project: { max: 20 }
---

# Campaign Planner

Analyze the codebase and plan a campaign for: {{inputs.campaign_goal}}

Create a project board and generate issues for all necessary tasks.
```

**Use case**: Team decides to launch a bug bash or tech debt campaign

### Scheduled: Proactive Campaign Planning

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * MON"  # Monday mornings
engine: copilot
safe-outputs:
  create-issue: { max: 20 }
  update-project: { max: 20 }
---

# Weekly Campaign Analyzer

Review repository health and recommend campaigns for:
- High-priority bugs that need focused attention
- Technical debt exceeding thresholds
- Performance regressions

If critical issues found, create campaign to address them.
```

**Use case**: Automated health monitoring suggests campaigns when needed

### Condition-Triggered: Reactive Campaign Launch

```aw wrap
---
on:
  issues:
    types: [labeled]
engine: copilot
safe-outputs:
  create-issue: { max: 20 }
  update-project: { max: 20 }
---

# Critical Bug Campaign

When 5+ issues labeled "critical", launch emergency bug fix campaign.

Create board, break down issues into actionable tasks, assign priorities.
```

**Use case**: System automatically escalates to campaign mode when thresholds exceeded

## Integrating Campaigns with Worker Workflows

Campaign workflows create the work, worker workflows execute it:

### Campaign Workflow (Orchestrator)
```yaml wrap
safe-outputs:
  create-issue:
    labels: ["performance", "campaign"]
  update-project: { max: 20 }
```

Creates issues with `performance` and `campaign` labels, adds to board.

### Worker Workflow (Executor)
```aw wrap
---
on:
  issues:
    types: [labeled]
engine: copilot
safe-outputs:
  create-pull-request: { max: 1 }
  update-project: { max: 1 }
---

# Performance Optimizer

When issue labeled "performance", fix the performance issue and update campaign board.

Extract campaign ID from issue labels, update board status to "In Progress", 
create PR with fix, update board to "Done" when merged.
```

Worker workflow detects campaign label, executes task, updates same board.

## Best Practices for Campaign Workflows

### For Campaign Planning
1. **Analyze before creating**: Let agent inspect codebase to find real issues
2. **Batch issue creation**: Use `create-issue: { max: 20 }` for multiple tasks
3. **Include campaign ID**: Auto-generated and added as label for tracking
4. **Set clear priorities**: Use custom fields (Critical/High/Medium/Low)
5. **Estimate effort**: Add effort field (XS/S/M/L/XL) for planning

### For Campaign Execution
1. **Worker workflows reference campaign ID**: Extract from labels to update correct board
2. **Update board status**: Move items through To Do → In Progress → Done
3. **Link PRs to issues**: Use "Fixes #123" to auto-close and track progress
4. **Query by campaign label**: `gh issue list --label "campaign:perf-q1-2025-a3f2b4c8"`
5. **Measure results**: Compare metrics before/after campaign completion

### For Campaign Tracking
1. **One board per campaign**: Don't mix campaigns on same board
2. **Descriptive board names**: Include goal and timeframe
3. **Preserve campaign history**: Don't delete boards, archive them
4. **Report with campaign ID**: Use ID in status updates and retrospectives
5. **Learn from campaigns**: Review what worked for future planning

## Quick Start

**Create your first campaign workflow:**

1. Add campaign workflow file (`.github/workflows/my-campaign.md`)
2. Define trigger (manual, scheduled, or condition-based)
3. Configure `create-issue` and `update-project` safe outputs
4. Write agent instructions to analyze and plan campaign
5. Run workflow to generate board and issues
6. Team executes tasks using worker workflows
7. Query campaign progress using campaign ID

The agent handles planning and organization, the team focuses on execution.
