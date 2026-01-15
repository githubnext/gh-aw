---
id: discussion-task-mining
name: "Campaign: Discussion Task Mining for Code Quality"
description: "Systematically extract actionable code quality improvement tasks from AI-generated discussions. Success: continuous identification of high-value refactoring, testing, and maintainability improvements."
version: v1
# Using Claude engine until Copilot is fixed
engine: claude
project-url: "https://github.com/orgs/githubnext/projects/TBD"  # To be updated when project is created
workflows:
  - discussion-task-miner
tracker-label: "campaign:discussion-task-mining"
memory-paths:
  - "memory/campaigns/discussion-task-mining/**"
  - "memory/discussion-task-miner/**"
metrics-glob: "memory/campaigns/discussion-task-mining/metrics/*.json"
cursor-glob: "memory/campaigns/discussion-task-mining/cursor.json"
state: planned
tags:
  - code-quality
  - task-mining
  - automation
  - refactoring
  - technical-debt
risk-level: low
allowed-safe-outputs:
  - create-issue
  - add-comment
objective: "Continuously mine AI-generated discussions for actionable code quality improvement tasks, converting insights into trackable issues that agents can implement"
kpis:
  - name: "Tasks identified per week"
    priority: primary
    unit: count
    baseline: 0
    target: 15
    time-window-days: 7
    direction: increase
    source: custom
  - name: "Task conversion rate (issues created/tasks identified)"
    priority: supporting
    unit: percent
    baseline: 60
    target: 80
    time-window-days: 7
    direction: increase
    source: custom
  - name: "Task completion rate (closed/created issues)"
    priority: supporting
    unit: percent
    baseline: 50
    target: 75
    time-window-days: 30
    direction: increase
    source: pull_requests
governance:
  max-issues-per-run: 5
  max-comments-per-run: 3
  max-discussions-scanned-per-run: 30
---

# Discussion Task Mining Campaign

## Overview

This campaign systematically mines AI-generated discussions (audits, reports, analyses) to extract actionable code quality improvement tasks. By analyzing what AI agents discover during their work, we create a continuous stream of high-value refactoring, testing, and maintainability improvements.

## Objective

**Continuously mine AI-generated discussions for actionable code quality improvement tasks, converting insights into trackable issues that agents can implement**

AI agents generate valuable insights about code quality issues during audits, analyses, and reports. This campaign harvests those insights and converts them into concrete, trackable tasks that can be assigned to agents for implementation.

## Success Criteria

- Identify 15+ actionable tasks per week from discussion mining
- Maintain 80%+ task conversion rate (tasks → issues)
- Achieve 75%+ task completion rate (issues resolved)
- Ensure all created issues are specific, actionable, and well-scoped
- Avoid creating duplicate issues (track processed discussions)
- Focus exclusively on code quality improvements

## Key Performance Indicators

### Primary KPI: Tasks Identified Per Week
- **Baseline**: 0 (campaign not yet running)
- **Target**: 15 tasks per week
- **Time Window**: 7 days (rolling)
- **Direction**: Increase
- **Source**: Custom metrics from task mining workflow

This KPI tracks how effectively we're discovering actionable tasks in AI-generated discussions. Higher numbers indicate we're successfully extracting valuable insights from agent outputs.

### Supporting KPI: Task Conversion Rate
- **Baseline**: 60% (estimated initial conversion)
- **Target**: 80% (high-quality task extraction)
- **Time Window**: 7 days (rolling)
- **Direction**: Increase
- **Source**: Custom metrics (issues created / tasks identified)

This KPI measures what percentage of identified tasks result in created issues. Higher rates indicate better task quality and filtering.

### Supporting KPI: Task Completion Rate
- **Baseline**: 50% (typical issue completion rate)
- **Target**: 75% (well-scoped, actionable tasks)
- **Time Window**: 30 days (rolling)
- **Direction**: Increase
- **Source**: Pull requests closing created issues

This KPI tracks how many created issues actually get resolved. Higher rates validate that we're creating valuable, actionable tasks that agents can complete.

## Associated Workflows

### discussion-task-miner
Scans AI-generated discussions from the last 7 days to extract actionable code quality improvement tasks. Creates GitHub issues for high-value tasks.

**Schedule**: Daily at 9am UTC

**What it does**:
- Queries recent discussions (audits, reports, analyses)
- Extracts specific, actionable code quality tasks
- Filters by impact, scope, and feasibility
- Creates GitHub issues (max 5 per run)
- Maintains memory to avoid duplicates
- Tracks metrics for continuous improvement

**Focus areas**:
- Refactoring opportunities
- Test coverage gaps
- Documentation improvements
- Performance optimizations
- Security enhancements
- Technical debt reduction
- Tooling improvements

## Project Board Setup

**Recommended Custom Fields**:

1. **Source Discussion** (Text): URL of originating discussion
   - Tracks where the task came from
   
2. **Task Type** (Single select): Refactoring, Testing, Documentation, Performance, Security, Tooling, Technical Debt
   - Categorizes the type of quality improvement
   
3. **Priority** (Single select): High, Medium, Low
   - Priority based on impact and urgency
   
4. **Effort** (Single select): Small (1 day), Medium (2-3 days), Large (1 week)
   - Estimated effort for completion
   
5. **Status** (Single select): Todo, In Progress, Review required, Blocked, Done
   - Current work state
   
6. **Impact Area** (Single select): Maintainability, Reliability, Performance, Security, Developer Experience
   - What aspect of quality this improves

The orchestrator automatically populates these fields based on issue content and labels.

## Agent Behavior Guidelines

Agents working on tasks from this campaign should:

### Task Selection
- Prioritize high-impact tasks that improve code quality
- Choose tasks matching their capabilities and expertise
- Start with "Small" effort tasks for quick wins
- Consider dependencies and related tasks

### Implementation Standards
- Follow existing code style and conventions
- Add or update tests to cover changes
- Update documentation as needed
- Run linters and formatters before submitting
- Ensure all existing tests pass

### Pull Request Quality
- Clear title describing the quality improvement
- Reference the original issue and source discussion
- Explain what was improved and why
- Include before/after comparisons if helpful
- Add test results and verification steps

### Communication
- Comment on issues when starting work
- Update status if blocked or needing clarification
- Link PRs to issues for tracking
- Notify when task is complete

## Timeline

- **Start**: When campaign is activated
- **Target completion**: Ongoing (continuous operation)
- **Current state**: Planned

This is a continuous improvement campaign with no end date. It runs indefinitely to maintain code quality.

## Success Metrics

### Discovery Effectiveness
- **Tasks per discussion**: How many actionable tasks extracted per discussion scanned
- **Discussion coverage**: Percentage of agent discussions analyzed
- **Pattern recognition**: Ability to identify recurring quality themes

### Task Quality
- **Specificity score**: Are tasks clearly defined with acceptance criteria?
- **Actionability score**: Can tasks be started immediately?
- **Completion rate**: Percentage of tasks that get resolved
- **Time to completion**: Average time from creation to closure

### Impact Metrics
- **Code quality improvements**: Measurable improvements in quality metrics
- **Technical debt reduction**: Number of legacy issues addressed
- **Test coverage increase**: Percentage increase in test coverage
- **Performance improvements**: Measurable performance gains

## Memory and State Management

### Repo-Memory Structure

```
memory/
├── campaigns/
│   └── discussion-task-mining/
│       ├── metrics/
│       │   └── weekly-stats.json        # Weekly KPI metrics
│       └── cursor.json                   # Campaign orchestration state
└── discussion-task-miner/
    ├── processed-discussions.json        # Discussions already mined
    ├── extracted-tasks.json              # All identified tasks
    └── latest-run.md                     # Most recent run summary
```

### Metrics Tracking

The workflow maintains detailed metrics:
- Discussions scanned per run
- Tasks identified per run
- Issues created per run
- Duplicates avoided
- Task types distribution
- Average task quality scores
- Completion rates by task type

## Governance Policies

### Rate Limits (per run)
- **Max issues created**: 5
- **Max comments**: 3
- **Max discussions scanned**: 30

These limits ensure sustainable operation and prevent overwhelming the system with too many issues at once.

### Quality Standards

All extracted tasks must meet these criteria:
1. **Specific**: Clear scope with well-defined boundaries
2. **Actionable**: Can be completed by an agent or developer
3. **Valuable**: Improves code quality, maintainability, or performance
4. **Scoped**: Completable in 1-3 days of work
5. **Independent**: No blocking dependencies on other tasks
6. **Documented**: Includes files affected and success criteria

### Deduplication Policy

To prevent duplicate issues:
- Track processed discussions in repo-memory
- Check existing issues before creation
- Maintain extracted tasks log
- Use title similarity matching
- Review recently closed issues

### Review Requirements

- All created issues auto-expire after 14 days if not addressed
- High-impact tasks may need human review before starting
- Security-related tasks require security team approval
- Architectural changes require tech lead approval

## Risk Assessment

**Risk Level**: Low

This campaign:
- Only reads discussions (no write operations)
- Creates issues for review (requires human approval to implement)
- Focuses on code quality (not security or production systems)
- Operates with rate limits and governance controls
- Maintains audit trail in repo-memory

The workflow cannot directly modify code - it only creates issues that humans or agents must review and approve.

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/discussion-task-mining.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-id
- Adds new issues to the project board
- Updates issue status and custom fields
- Aggregates metrics from task mining runs
- Reports campaign progress and quality trends
- Identifies high-value tasks for prioritization

## Example Tasks

Good examples of tasks this campaign might extract:

### Refactoring
- "Split 500-line function into smaller, testable functions"
- "Extract duplicate error handling logic into shared utility"
- "Reduce cyclomatic complexity in authentication module"

### Testing
- "Add missing unit tests for API client methods"
- "Fix flaky integration test in payment processing"
- "Increase test coverage for parser package from 45% to 80%"

### Documentation
- "Add godoc comments to exported functions in workflow package"
- "Create troubleshooting guide for common CLI errors"
- "Update README with examples of new safe-output types"

### Performance
- "Optimize O(n²) loop in validation code"
- "Add caching to reduce redundant GitHub API calls"
- "Reduce memory allocation in compiler hot path"

### Technical Debt
- "Replace deprecated YAML library with recommended version"
- "Remove commented-out code in legacy modules"
- "Update TODOs with implementation plans or remove"

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination happens in the orchestrator
- The GitHub Project board is the single source of truth
- Safe outputs include AI-generated footers for transparency
- This campaign complements (not replaces) regular issue triage
- Focus is exclusively on code quality, not features or bugs
