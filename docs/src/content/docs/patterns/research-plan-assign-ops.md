---
title: ResearchPlanAssignOps
description: Orchestrate deep research, structured planning, and automated assignment to drive AI-powered development cycles from insight to merged PR
sidebar:
  badge: { text: 'Multi-phase', variant: 'caution' }
---

ResearchPlanAssignOps is a four-phase development pattern that moves from automated discovery to merged code with human control at every decision point. A research agent surfaces insights, a planning agent converts them into actionable issues, a coding agent implements the work, and a human reviews and merges.

## The Four Phases

```
Research → Plan → Assign → Merge
```

Each phase produces a concrete artifact consumed by the next, and every transition is a human checkpoint.

### Phase 1: Research

A scheduled workflow investigates the codebase from a specific angle and publishes its findings as a GitHub discussion. The discussion is the contract between the research phase and everything that follows—it contains the analysis, recommendations, and context a planner needs.

The [`go-fan`](https://github.com/github/gh-aw/blob/main/.github/workflows/go-fan.md) workflow is a live example: it runs each weekday, picks one Go dependency, compares current usage against upstream best practices, and creates a `[go-fan]` discussion under the `audits` category.

```aw wrap
---
name: Go Fan
on:
  schedule:
    - cron: "0 7 * * 1-5"
  workflow_dispatch:
engine: claude
safe-outputs:
  create-discussion:
    title-prefix: "[go-fan] "
    category: "audits"
    max: 1
    close-older-discussions: true
tools:
  cache-memory: true
  github:
    toolsets: [default]
---

Analyze today's Go dependency. Compare current usage in this
repository against upstream best practices and recent releases.
Save a summary to scratchpad/mods/ and create a discussion
with findings and improvement recommendations.
```

The research agent uses `cache-memory` to track which modules have been reviewed so it rotates through them systematically across runs.

### Phase 2: Plan

After reading the research discussion, a developer triggers the `/plan` command on it. The [`plan`](https://github.com/github/gh-aw/blob/main/.github/workflows/plan.md) workflow reads the discussion, extracts concrete work items, and creates up to five sub-issues grouped under a parent tracking issue.

```
/plan focus on the quick wins and API simplifications
```

The planner formats each sub-issue for a coding agent: a clear objective, the files to touch, step-by-step implementation guidance, and acceptance criteria. Issues are tagged `[plan]` and `ai-generated`.

> [!TIP]
> The `/plan` command accepts inline guidance. Steer it toward high-priority findings or away from lower-priority ones before it generates issues.

### Phase 3: Assign

With well-scoped issues in hand, the developer [assigns them to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot) for automated implementation. Copilot opens a pull request and posts progress updates as it works.

Issues can be assigned individually through the GitHub UI, or pre-assigned in bulk via an orchestrator workflow:

```aw wrap
---
name: Auto-assign plan issues to Copilot
on:
  issues:
    types: [labeled]
engine: copilot
safe-outputs:
  assign-to-user:
    target: "*"
  add-comment:
    target: "*"
---

When an issue is labeled `plan` and has no assignee,
assign it to Copilot and add a comment indicating
automated assignment.
```

For multi-issue plans, assignments can run in parallel—Copilot works independently on each issue and opens separate PRs.

### Phase 4: Merge

Copilot's pull request is reviewed by a human maintainer. The maintainer checks correctness, runs tests, and merges. The tracking issue created in Phase 2 closes automatically when all sub-issues are resolved.

## End-to-End Example

The following trace shows the full cycle using `go-fan` as the research driver.

**Monday 7 AM** — `go-fan` runs and creates a discussion:

> **[go-fan] Go Module Review: spf13/cobra**
>
> Current usage creates a new `Command` per invocation. cobra v1.8 introduced
> `SetContext` for propagating cancellation. Quick wins: pass context through
> subcommands, use `PersistentPreRunE` for shared setup.

**Monday afternoon** — Developer reads the discussion and types:

```
/plan
```

The planner creates a parent tracking issue `[plan] cobra improvements` with three sub-issues:

- `[plan] Pass context through subcommands using cobra SetContext`
- `[plan] Refactor shared setup into PersistentPreRunE`
- `[plan] Add context cancellation tests`

**Monday afternoon** — Developer assigns the first two issues to Copilot. Both open PRs within minutes.

**Tuesday** — Developer reviews PRs, requests a minor change on one, approves the other. Both merge by end of day. The tracking issue closes.

## Workflow Configuration Patterns

### Research: produce one discussion per run

```aw wrap
safe-outputs:
  create-discussion:
    expires: 1d
    category: "research"
    max: 1
    close-older-discussions: true
```

`close-older-discussions: true` prevents discussion accumulation—only the latest finding stays open for the planner.

### Research: maintain memory across runs

```aw wrap
tools:
  cache-memory: true
```

Use `cache-memory` to track state between scheduled runs—which items have been reviewed, trend data, or historical baselines.

### Plan: issue grouping

```aw wrap
safe-outputs:
  create-issue:
    expires: 2d
    title-prefix: "[plan] "
    labels: [plan, ai-generated]
    max: 5
    group: true
```

`group: true` creates a parent tracking issue automatically. Do not create the parent manually—the workflow handles it.

### Assign: pre-assign via `assignees`

For research workflows that produce self-contained, well-scoped issues, skip the manual plan phase and assign directly:

```aw wrap
safe-outputs:
  create-issue:
    title-prefix: "[fix] "
    labels: [ai-generated]
    assignees: copilot
```

The `duplicate-code-detector` workflow uses this approach—duplication fixes are narrow enough that a planning phase adds no value.

## When to Use ResearchPlanAssignOps

This pattern fits when:

- The scope of work is unknown until analysis runs
- Issues need human prioritization before implementation
- Research findings vary in quality (some runs find nothing actionable)
- Multiple work items can be executed in parallel

Prefer a simpler pattern when:

- The work is already well-defined (use [IssueOps](/gh-aw/patterns/issue-ops/))
- Issues can go directly to Copilot without review (use the `assignees: copilot` shortcut in your research workflow)
- Work spans multiple repositories (use [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/))

## Existing Workflows

| Phase | Workflow | Description |
|-------|----------|-------------|
| Research | [`go-fan`](https://github.com/github/gh-aw/blob/main/.github/workflows/go-fan.md) | Daily Go dependency analysis with best-practice comparison |
| Research | [`copilot-cli-deep-research`](https://github.com/github/gh-aw/blob/main/.github/workflows/copilot-cli-deep-research.md) | Weekly analysis of Copilot CLI feature usage |
| Research | [`static-analysis-report`](https://github.com/github/gh-aw/blob/main/.github/workflows/static-analysis-report.md) | Daily security scan with clustered findings |
| Research | [`duplicate-code-detector`](https://github.com/github/gh-aw/blob/main/.github/workflows/duplicate-code-detector.md) | Daily semantic duplication analysis (auto-assigns) |
| Plan | [`plan`](https://github.com/github/gh-aw/blob/main/.github/workflows/plan.md) | `/plan` slash command—converts issues or discussions into sub-issues |
| Assign | GitHub UI / workflow | [Assign issues to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot) for automated PR creation |

## Related Patterns

- **[TaskOps](/gh-aw/patterns/task-ops/)** — Detailed breakdown of the three-phase Research → Plan → Assign strategy with configuration guidance
- **[Orchestration](/gh-aw/patterns/orchestration/)** — Fan out work across multiple worker workflows
- **[DailyOps](/gh-aw/patterns/daily-ops/)** — Scheduled incremental improvements without a separate planning phase
- **[DispatchOps](/gh-aw/patterns/dispatch-ops/)** — Manually triggered research and one-off investigations
