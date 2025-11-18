---
title: Campaigns
description: Coordinate multi-issue initiatives with AI-powered planning, tracking, and orchestration
---

A **campaign** coordinates related work toward a shared goal. Campaigns can be a bundle of workflows (launcher + workers + monitors), a bundle of issues (coordinated via labels, project boards, or epic issues), or both. They coordinate multiple workflows and/or issues with measurable goals (like "reduce page load by 30%"), flexible tracking (project boards, epic issues, discussions, or labels), and a campaign ID linking all work together.

Instead of executing individual tasks, campaigns orchestrate: analyze context, generate work, track progress, adapt to feedback. Compare to regular workflows which execute one task—campaigns **orchestrate multiple related pieces of work**.

## How Campaigns Work

Campaigns use safe outputs to coordinate work:

```yaml wrap
safe-outputs:
  create-issue: { max: 20 }      # Generate work items
  update-project: { max: 20 }    # Optional: project board tracking
  create-discussion: { max: 1 }  # Optional: planning discussion
```

**Tracking options** (choose what fits):
- **Discussion** - Planning thread with updates (research-heavy)
- **Epic issue** - Single issue with task list (simple campaigns)
- **Labels only** - Just `campaign:<id>` labels (minimal overhead)
- **Project board** - Visual dashboard with custom fields (complex campaigns)

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

**What happens**:
1. Agent analyzes triage process and creates discussion with goals
2. Generates 5-10 issues for triage improvements with campaign labels
3. Team reviews and prioritizes issues
4. Worker workflows execute individual improvements
5. Track progress via campaign ID: `gh issue list --label "campaign:ai-triage-[id]"`

## Campaign IDs

Campaign IDs use format `[slug]-[timestamp]` (e.g., `ai-triage-a3f2b4c8`). They're auto-generated and applied as labels to all campaign issues.

**Query campaign work:**
```bash
gh issue list --label "campaign:ai-triage-a3f2b4c8"
gh pr list --label "campaign:ai-triage-a3f2b4c8"
```

## Common Patterns

**Manual launch** - User triggers campaign for specific goal
**Scheduled monitoring** - Weekly checks suggest campaigns when needed  
**Threshold-triggered** - Auto-launch when critical issues accumulate

## Campaign Architecture

A campaign typically involves multiple coordinated workflows:

**Launcher workflow** (orchestrator):
- Analyzes codebase
- Creates multiple issues with campaign labels
- Sets up tracking (board/epic/discussion)
- Defines campaign goals and KPIs

**Worker workflows** (executors):
- Trigger on campaign-labeled issues
- Execute individual tasks
- Reference campaign ID in PRs
- Update campaign status

**Monitor workflows** (optional):
- Track campaign progress on schedule
- Report metrics against KPIs
- Update campaign tracking with status

All workflows in a campaign share the same campaign ID for coordination.

## Best Practices

- **Define clear KPIs** - Make goals measurable ("reduce load time by 30%")
- **Choose right tracking** - Labels for simple, project boards for complex campaigns
- **Link resources** - Include telemetry, docs, specs in campaign tracking
- **Use consistent IDs** - Apply campaign labels to all related issues/PRs
- **Archive when done** - Preserve campaign history and learnings

## Quick Start

1. Create workflow file: `.github/workflows/my-campaign.md`
2. Add safe outputs: `create-issue`, `update-project`, or `create-discussion`
3. Write instructions to analyze context and generate issues
4. Run workflow to launch campaign
5. Team executes via worker workflows
6. Track progress: `gh issue list --label "campaign:<id>"`
