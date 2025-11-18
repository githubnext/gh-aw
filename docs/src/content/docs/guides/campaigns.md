---
title: Campaign Workflows
description: Use agentic workflows to orchestrate multi-issue initiatives with goals, KPIs, resources, and human guidance - campaigns coordinate work rather than execute it.
---

Campaign workflows enable AI agents to orchestrate focused initiatives by coordinating buckets of related issues, tracking progress toward goals, synthesizing resources, and adapting to human guidance. Campaigns are coordination mechanisms, not execution mechanisms.

## What is a Campaign?

A **campaign** is a coordinated initiative with:

- **Bucket of issues** - Related work items grouped by shared purpose
- **Goals and KPIs** - Measurable objectives ("reduce page load by 30%", "achieve 80% test coverage")
- **Resources** - Telemetry data, docs, specs, research, external context
- **Human guidance** - Comments from maintainers, user feedback, design decisions
- **History** - Timeline of PRs, events, discussions causally related to campaign
- **Tracking mechanism** - Project board, epic issue, discussion, or just labels

## Campaign vs Task Workflows

| Regular Workflow | Campaign Workflow |
|------------------|-------------------|
| Executes one task | Coordinates multiple related tasks |
| Single issue/PR | Manages bucket of issues |
| Direct action | Strategic orchestration |
| Tactical | Strategic |

**Campaign workflow responsibilities:**
- Analyze context to identify related work items
- Organize issues into campaign bucket (via labels, project, or epic)
- Define campaign goals, KPIs, and success criteria
- Synthesize resources (telemetry, docs, research)
- Create tracking mechanism (project board, epic issue, or discussion)
- Generate campaign ID for coordination

**Worker workflow responsibilities:**
- Execute individual issues within campaign scope
- Reference campaign ID in commits and PRs
- Update campaign status through labels/comments
- Contribute to campaign goals incrementally

## How Campaign Workflows Work

Campaign workflows use safe outputs flexibly based on coordination needs:

```yaml wrap
safe-outputs:
  create-issue: { max: 20 }      # Generate campaign work items
  update-project: { max: 20 }    # Optional: use project boards
  create-discussion: { max: 1 }  # Optional: use discussions for tracking
  add-comment: { max: 10 }       # Update epic issues or discussions
```

### Campaign Tracking Options

Campaigns are **not tightly coupled to any single GitHub feature**. Choose what fits:

**Project Board (structured dashboards)**
- Visual kanban for status tracking
- Custom fields for priority, effort, phase
- Good for: executive visibility, multi-phase campaigns

**Epic Issue (lightweight tracking)**
- Single issue with task list linking related issues
- Comments for progress updates and guidance
- Good for: simple campaigns, developer-focused coordination

**Discussion Thread (research and planning)**
- Long-form updates with persistent history
- Good for: proto-campaigns, research phases, community input

**Campaign Labels Only (minimal overhead)**
- Just `campaign:<id>` labels on related issues
- Query and report using GitHub CLI
- Good for: informal coordination, fast iteration

Most campaigns use **multiple mechanisms**: labels for querying, discussion for planning, project board or epic issue for tracking.

## Campaign Workflow Example

### AI Triage Campaign

**Goal**: Implement intelligent issue triage to reduce maintainer burden

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      triage_goal:
        description: "What should AI triage accomplish?"
        default: "Auto-label, route, and prioritize all new issues"

engine: copilot

safe-outputs:
  create-issue: { max: 20 }        # Create tasks
  create-discussion: { max: 1 }    # Campaign planning discussion
---

# AI Triage Campaign

You are launching an AI triage campaign.

**Goal**: {{inputs.triage_goal}}

**Your tasks**:

1. **Create campaign discussion**: "AI Triage Campaign - [Today's Date]"
   - Document campaign goals and KPIs
   - Link to relevant resources (existing triage workflows, issue templates)

2. **Analyze current triage process**:
   - Review existing issue labels and their usage
   - Identify common issue types and patterns
   - Check current triage response times
   - Look for triage bottlenecks

3. **Create issues for each improvement**:
   - Title: Clear description of triage enhancement
   - Labels: "triage", "campaign:ai-triage-[timestamp]"
   - Body: Specific metrics, acceptance criteria, implementation approach
   
   Example issues:
   - Auto-label bug reports based on content
   - Route feature requests to appropriate project boards
   - Prioritize security issues automatically
   - Add "needs-reproduction" label when stack traces missing
   - Suggest duplicate issues using semantic search

4. **Track in discussion**:
   - Campaign ID for querying: `campaign:ai-triage-[timestamp]`
   - Success criteria: 80% of issues auto-labeled within 5 minutes
   - Resources: Link to issue templates, label taxonomy, triage docs

Provide campaign summary with issue list and discussion URL.
```

### What the Agent Does

1. **Analyzes context**: Reviews current issue triage process and bottlenecks
2. **Creates campaign discussion**: Establishes planning thread with goals and resources
3. **Generates task issues**: One issue per triage improvement with detailed implementation approach
4. **Organizes work**: Applies campaign labels for coordination and tracking
5. **Documents campaign**: Campaign ID, success criteria, and resources in discussion

### What the Team Does

- Reviews generated issues and prioritizes implementation order
- Assigns issues to team members or labels for worker workflows
- Worker workflows implement triage automation when issues are labeled
- Team monitors triage metrics (auto-labeling rate, response time, accuracy)
- Campaign discussion shows progress updates and learnings

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
User/Scheduled Trigger
         ↓
  Campaign Workflow
         ↓
Analyze codebase/context
         ↓
Identify related work items
         ↓
Create tracking mechanism
(project/epic/discussion/labels)
         ↓
Generate campaign ID
         ↓
Create/organize issues
         ↓
Apply campaign labels
         ↓
    Issue Created
         ↓
  Worker Workflow
  (label-triggered)
         ↓
   Execute Task
         ↓
  Update Status
  (via labels/comments)
         ↓
   Create PR
         ↓
References campaign ID
         ↓
  Campaign Progress
   Tracked & Visible
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
      tracking_method:
        description: "Tracking method: project-board, epic-issue, discussion, or labels-only"
        default: "discussion"
engine: copilot
safe-outputs:
  create-issue: { max: 20 }
  create-discussion: { max: 1 }
  update-project: { max: 20 }
---

# Campaign Launcher

Launch campaign: {{inputs.campaign_goal}}

**Setup tasks:**
1. Generate campaign ID from goal
2. Analyze repository to identify work items needed
3. Create tracking mechanism: {{inputs.tracking_method}}
   - project-board: Create board with status fields
   - epic-issue: Create epic issue with task list
   - discussion: Create planning discussion thread
   - labels-only: Just apply campaign labels to issues
4. Define campaign goals, KPIs, success criteria
5. Generate issues with campaign labels
6. Create initial status report with campaign overview

Provide campaign ID and tracking URL in summary.
```

**Use case**: Team decides to launch a focused initiative (e.g., "Implement AI triage", "Migrate to TypeScript", "Security audit")

### Scheduled: Proactive Campaign Monitoring

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

# Weekly Campaign Health Check

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
1. **Choose appropriate tracking**: Match mechanism to campaign complexity (labels-only for simple, project board for complex)
2. **Define clear goals and KPIs**: Measurable objectives enable progress tracking
3. **Synthesize resources**: Link to telemetry, docs, specs, research
4. **Use campaign ID consistently**: Apply to all related issues, PRs, commits
5. **Analyze before creating**: Let agent inspect codebase to find real issues

### For Campaign Execution
1. **Worker workflows reference campaign ID**: Extract from labels to coordinate updates
2. **Link PRs to issues**: Use "Fixes #123" to track progress automatically
3. **Query by campaign label**: `gh issue list --label "campaign:perf-q1-2025-a3f2b4c8"`
4. **Measure against KPIs**: Compare metrics to campaign goals regularly
5. **Update campaign status**: Comment on epic/discussion with progress

### For Campaign Tracking
1. **One tracking mechanism per campaign**: Keep campaigns clearly separated
2. **Descriptive names**: Include goal and timeframe
3. **Preserve history**: Archive completed campaigns with outcomes and learnings
4. **Report with campaign ID**: Use ID in status updates and retrospectives
5. **Learn from campaigns**: Review what worked for future planning

## Quick Start

**Create your first campaign workflow:**

1. Add campaign workflow file (`.github/workflows/my-campaign.md`)
2. Define trigger (manual, scheduled, or condition-based)
3. Configure `create-issue` and tracking safe outputs (`update-project`, `create-discussion`, etc.)
4. Write agent instructions to analyze and plan campaign
5. Run workflow to generate tracking mechanism and issues
6. Team executes tasks using worker workflows
7. Query campaign progress using campaign ID

The agent handles planning and organization, the team focuses on execution.
