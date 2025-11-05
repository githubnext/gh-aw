---
title: DailyOps
description: Learn how to implement DailyOps workflows that make incremental improvements through scheduled, bite-sized changes at regular intervals.
---

DailyOps workflows tackle large goals through small, scheduled improvements that run automatically every business day. Instead of implementing major changes all at once, DailyOps breaks work into manageable pieces that the team can review and integrate without disruption.

## Overview

The DailyOps pattern uses scheduled workflows that run daily (typically weekdays) to make incremental progress toward long-term goals. Each workflow focuses on a specific area—documentation, testing, performance, or monitoring—and produces small, reviewable changes.

**Key Benefits:**
- Progress happens automatically without manual intervention
- Changes stay small enough for easy review and integration
- Work continues consistently without overwhelming the team
- Goals that seem too large become achievable through daily increments

## The DailyOps Pattern

### Scheduled Execution

DailyOps workflows run on weekday schedules, avoiding weekends when teams are less available to review changes:

```aw wrap
---
on:
  schedule:
    # Run at 2am UTC, Monday through Friday
    - cron: "0 2 * * 1-5"
  workflow_dispatch:  # Allow manual triggering
---
```

The `workflow_dispatch` trigger allows manual execution for testing or urgent runs outside the normal schedule.

### Phased Approach

Many DailyOps workflows organize work into three phases that run sequentially across multiple days:

**Phase 1: Research and Planning**
- Analyze the current state of the repository
- Identify opportunities for improvement
- Create a discussion with findings and proposed strategy
- Wait for maintainer review before proceeding

**Phase 2: Configuration Setup**
- Define the build/test/analysis steps needed
- Create configuration files or action definitions
- Test the configuration with trial runs
- Create a pull request for review

**Phase 3: Incremental Execution**
- Select a small, focused goal from the plan
- Make improvements following the strategy
- Verify changes work correctly
- Create a draft pull request with results

### Progress Tracking

DailyOps workflows use GitHub discussions to maintain continuity across runs:

```aw wrap
safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"
```

The workflow checks for an existing discussion titled with the workflow name. If none exists, it creates one with the research and plan. Subsequent runs add brief comments documenting progress.

### Persistent Memory

DailyOps workflows benefit from `cache-memory` to store state between runs:

```aw wrap
tools:
  cache-memory: true
```

This enables workflows to maintain persistent files at `/tmp/gh-aw/cache-memory/` across runs, useful for:
- Tracking which areas have been worked on
- Storing progress metrics and trends
- Maintaining incremental state between daily runs
- Building knowledge bases over time

The cache persists across workflow runs and automatically creates restore keys for intelligent fallback.

## Common DailyOps Workflows

### Test Coverage Improvement

The daily test improver systematically increases test coverage by adding tests for uncovered code each day:

```aw wrap
---
on:
  schedule:
    - cron: "0 2 * * 1-5"
permissions:
  all: read
tools:
  cache-memory: true
safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"
  create-pull-request:
    draft: true
---

# Daily Test Coverage Improver

You are an AI test engineer for ${{ github.repository }}.
Your mission: systematically identify and implement test coverage improvements.

[Phase selection logic...]

## Phase 3 - Work selection, work and results

1. Review coverage report
2. Select area with low coverage
3. Write meaningful tests for that area
4. Verify tests pass and coverage improved
5. Create draft PR with improvements
```

**Example from this repository:** `.github/workflows/daily-test-improver.md`

### Performance Optimization

The daily performance improver identifies bottlenecks and implements optimizations incrementally:

```aw wrap
---
on:
  schedule:
    - cron: "0 2 * * 1-5"
timeout_minutes: 60
tools:
  cache-memory: true
safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
  create-pull-request:
    draft: true
---

# Daily Perf Improver

You are an AI performance engineer for ${{ github.repository }}.
Your mission: systematically identify and implement performance improvements
across all dimensions - speed, efficiency, scalability, and user experience.

[Phased approach focusing on measurement and optimization...]
```

**Example from this repository:** `.github/workflows/daily-perf-improver.md`

### Documentation Updates

The daily documentation updater keeps docs current by reviewing merged changes each day:

```aw wrap
---
on:
  schedule:
    - cron: "0 6 * * *"
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  cache-memory: true
safe-outputs:
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation, automation]
---

# Daily Documentation Updater

Scan the repository for merged pull requests and code changes from the last 24 hours,
identify new features or changes that should be documented, and update documentation.

[Steps to analyze changes and update docs...]
```

**Example from this repository:** `.github/workflows/daily-doc-updater.md`

### Status Reporting

Daily news workflows create regular reports on repository activity, helping teams stay informed:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1-5"  # Every weekday at 9am UTC
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  cache-memory: true
safe-outputs:
  create-discussion:
    category: "daily-news"
---

# Daily News

Write an upbeat, friendly summary of recent activity in the repo.

Include:
- Recent issues and pull requests
- Code changes and releases
- Team productivity insights
- Suggestions for improvement

Create a new GitHub discussion with today's date.
```

**Example from this repository:** `.github/workflows/daily-news.md`

## Implementation Guide

### Step 1: Define Your Goal

Identify a large, ongoing goal that would benefit from daily incremental progress:

- Increasing test coverage to 80%
- Improving performance metrics
- Keeping documentation up-to-date
- Monitoring security or quality metrics

### Step 2: Design the Workflow

Create a workflow file following the DailyOps pattern:

1. **Set the schedule:** Choose a weekday time when the team can review results
2. **Plan phases:** Decide if you need research, configuration, and execution phases
3. **Define outputs:** Use `safe-outputs` for discussions and pull requests
4. **Write instructions:** Provide clear guidance for what the AI should do each day

### Step 3: Start with Research

On the first run, have the workflow analyze the current state and create a plan:

```markdown
## Phase 1 - Research

1. Analyze the current state of [target area]
2. Identify opportunities for improvement
3. Create a discussion with:
   - Summary of findings
   - Proposed strategy
   - Specific areas to focus on
   - Questions for maintainers
4. Exit workflow and wait for human review
```

### Step 4: Configure and Test

Once the plan is approved, set up the necessary tools and configuration:

```markdown
## Phase 2 - Configuration

1. Create configuration files needed for [goal]
2. Test the configuration manually
3. Create a pull request with configuration
4. Exit workflow
```

### Step 5: Execute Daily

With research and configuration complete, make incremental progress:

```markdown
## Phase 3 - Execution

1. Check the plan and previous work
2. Select a small, focused area to improve
3. Make the improvement
4. Verify it works correctly
5. Create a draft pull request
6. Add brief progress note to discussion
```

## Best Practices

### Keep Changes Small

Each daily run should produce changes that can be reviewed in 5-10 minutes. If changes are too large, the team won't have time to review and merge them regularly.

### Use Draft Pull Requests

Create pull requests as drafts until maintainers review and approve them. This signals that the changes need human review before merging.

### Track Progress Clearly

Use discussions to maintain a record of:
- The overall plan and strategy
- Progress made each day
- Issues encountered
- Adjustments to the plan

### Handle Failures Gracefully

When a daily run encounters problems:
- Create an issue describing the problem
- Update the discussion with what went wrong
- Exit cleanly rather than making incorrect changes

### Allow Manual Triggering

Always include `workflow_dispatch` so maintainers can:
- Test the workflow before the scheduled run
- Trigger extra runs when needed
- Debug issues by running manually

### Avoid Weekends

Schedule workflows for weekdays only (cron pattern `* * * * 1-5`) so changes don't pile up when the team isn't available to review them.

## Examples from This Repository

The gh-aw repository uses several DailyOps workflows:

- **daily-test-improver.md** - Adds tests to improve coverage incrementally
- **daily-perf-improver.md** - Identifies and implements performance optimizations
- **daily-doc-updater.md** - Keeps documentation synchronized with code changes
- **daily-news.md** - Creates engaging daily status reports with trend analysis
- **daily-repo-chronicle.md** - Produces newspaper-style repository updates
- **daily-firewall-report.md** - Analyzes and reports on firewall activity

Each workflow follows the phased approach, uses discussions for tracking, and creates draft pull requests for human review.

## Related Patterns

- **IssueOps** - Trigger workflows from issue creation or comments
- **ChatOps** - Trigger workflows from slash commands in comments
- **LabelOps** - Trigger workflows when labels change on issues or pull requests
- **Planning Workflow** - Use `/plan` command to split large discussions into actionable work items, then assign sub-tasks to Copilot for execution

DailyOps complements these patterns by providing scheduled automation that doesn't require manual triggers.
