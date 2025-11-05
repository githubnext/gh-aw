---
title: DailyOps
description: Learn how to implement DailyOps workflows that make incremental improvements through scheduled, bite-sized changes at regular intervals.
---

DailyOps workflows use scheduled automation to make incremental progress toward large goals through small, daily changes. Instead of overwhelming the team with major changes, work happens automatically in manageable pieces that are easy to review and integrate. This pattern transforms ambitious long-term goals into achievable daily tasks.

## The DailyOps Pattern

### Scheduled Execution

Workflows run on weekday schedules (avoiding weekends) with `workflow_dispatch` enabled for manual testing:

```aw wrap
---
on:
  schedule:
    - cron: "0 2 * * 1-5"  # 2am UTC, Monday-Friday
  workflow_dispatch:
---
```

### Phased Approach

DailyOps workflows typically organize work into three sequential phases: **Research** (analyze state, create discussion with findings), **Configuration** (define steps, create config PR), and **Execution** (make small improvements, verify, create draft PRs). Each phase waits for maintainer approval before proceeding to the next.

### Progress Tracking

Use GitHub discussions to maintain continuity across runs. The workflow creates a discussion (if none exists) and adds progress comments on subsequent runs:

```aw wrap
safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"
```

### Persistent Memory

Enable `cache-memory` to maintain state at `/tmp/gh-aw/cache-memory/` across runs, useful for tracking progress, storing metrics, and building knowledge bases over time:

```aw wrap
tools:
  cache-memory: true
```

## Common DailyOps Workflows

This repository implements several DailyOps workflows demonstrating different use cases:

- **daily-test-improver.md** - Systematically adds tests to improve coverage incrementally
- **daily-perf-improver.md** - Identifies and implements performance optimizations
- **daily-doc-updater.md** - Keeps documentation synchronized with merged code changes
- **daily-news.md** - Creates engaging daily status reports with trend analysis
- **daily-repo-chronicle.md** - Produces newspaper-style repository updates
- **daily-firewall-report.md** - Analyzes and reports on firewall activity

All follow the phased approach with discussions for tracking and draft pull requests for review.

## Implementation Guide

**1. Define Your Goal** - Identify a large, ongoing goal (test coverage, performance, documentation sync, quality monitoring).

**2. Design the Workflow** - Set weekday schedule, plan phases (research/config/execution), configure `safe-outputs` for discussions and PRs.

**3. Start with Research** - First run analyzes current state, creates discussion with findings and strategy, exits for human review.

**4. Configure and Test** - Once approved, create necessary configuration files, test them, submit config PR, exit.

**5. Execute Daily** - With setup complete, select small focused areas, make improvements, verify, create draft PRs, update discussion.

## Best Practices

Keep daily changes reviewable in 5-10 minutes. Use draft pull requests to signal human review needed. Track overall plan, daily progress, and issues in discussions. Handle failures gracefully by creating issues and exiting cleanly. Always enable `workflow_dispatch` for manual testing and debugging. Schedule for weekdays only to avoid change buildup during weekends.


## Related Patterns

- **IssueOps** - Trigger workflows from issue creation or comments
- **ChatOps** - Trigger workflows from slash commands in comments
- **LabelOps** - Trigger workflows when labels change on issues or pull requests
- **Planning Workflow** - Use `/plan` command to split large discussions into actionable work items, then assign sub-tasks to Copilot for execution

DailyOps complements these patterns by providing scheduled automation that doesn't require manual triggers.
