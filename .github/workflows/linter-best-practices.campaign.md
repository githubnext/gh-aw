---
id: linter-best-practices
name: "Campaign: Linter Best Practices"
description: "Enforce consistent code quality and best practices across all file types, automatically identify and address code simplification opportunities, and track security vulnerabilities through static analysis. Success: reduce linting violations by 80% and increase code quality score to 95%."
version: v1
# Using Claude engine until Copilot is fixed
engine: claude
project-url: "https://github.com/orgs/githubnext/projects/TBD"  # To be updated when project is created
workflows:
  - super-linter
  - code-simplifier
  - static-analysis-report
  - repository-quality-improver
  - cli-consistency-checker
tracker-label: "campaign:linter-best-practices"
memory-paths:
  - "memory/campaigns/linter-best-practices/**"
metrics-glob: "memory/campaigns/linter-best-practices/metrics/*.json"
cursor-glob: "memory/campaigns/linter-best-practices/cursor.json"
state: planned
tags:
  - code-quality
  - linting
  - best-practices
  - security
  - maintainability
risk-level: low
allowed-safe-outputs:
  - create-issue
  - add-comment
  - create-pull-request
  - create-discussion
  - update-project
objective: "Reduce linting violations by 80% within 30 days, increase code quality score from 85% to 95% within 90 days, and reduce pull requests with linting issues from 30% to 5%"
kpis:
  - name: "Linting violations"
    priority: primary
    unit: count
    baseline: 100
    target: 20
    time-window-days: 30
    direction: decrease
    source: custom
  - name: "Code quality score"
    priority: supporting
    unit: percent
    baseline: 85
    target: 95
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Pull requests with linting issues"
    priority: supporting
    unit: percent
    baseline: 30
    target: 5
    time-window-days: 30
    direction: decrease
    source: pull_requests
governance:
  max-project-updates-per-run: 15
  max-comments-per-run: 10
  max-new-items-per-run: 8
  max-issues-per-run: 5
  max-pull-requests-per-run: 3
---

# Linter Best Practices Campaign

## Overview

This campaign systematically enforces consistent code quality and best practices across all file types in the GitHub Agentic Workflows repository. By coordinating multiple linting and quality analysis workflows, we reduce technical debt, improve maintainability, and ensure code consistency.

## Objective

**Reduce linting violations by 80% within 30 days, increase code quality score from 85% to 95% within 90 days, and reduce pull requests with linting issues from 30% to 5%**

Consistent code quality is critical for maintainability, collaboration, and long-term project health. This campaign coordinates multiple workflows to systematically identify and address linting violations, code simplification opportunities, and security vulnerabilities.

## Success Criteria

- Reduce linting violations from ~100 to 20 or fewer within 30 days
- Increase overall code quality score from 85% to 95% within 90 days
- Reduce pull requests with linting issues from 30% to 5%
- All code passes automated linting checks
- Security vulnerabilities identified and tracked
- Consistent code style across all file types

## Key Performance Indicators

### Primary KPI: Linting Violations
- **Baseline**: 100 violations (current estimated count)
- **Target**: 20 violations (80% reduction)
- **Time Window**: 30 days
- **Direction**: Decrease
- **Source**: Custom metrics from linting workflows

This KPI tracks the total number of linting violations across all file types (Markdown, Go, JavaScript, YAML, etc.). Lower numbers indicate cleaner, more maintainable code.

### Supporting KPI: Code Quality Score
- **Baseline**: 85% (current estimated quality)
- **Target**: 95% (high-quality codebase)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from quality analysis

This KPI measures overall code quality including complexity, test coverage, documentation, and maintainability. Higher scores indicate better long-term project health.

### Supporting KPI: Pull Requests with Linting Issues
- **Baseline**: 30% (current PR failure rate)
- **Target**: 5% (minimal linting issues)
- **Time Window**: 30 days (rolling)
- **Direction**: Decrease
- **Source**: Pull request CI status

Lower percentages indicate developers are catching issues earlier and following best practices consistently.

## Associated Workflows

### super-linter
Runs comprehensive Markdown quality checks using Super Linter and creates issues for violations.

**Schedule**: Triggered on pull requests and manually

**What it does**:
- Validates Markdown formatting and style
- Checks for broken links
- Enforces consistent documentation standards
- Creates issues for violations

### code-simplifier
Analyzes recently modified code and creates pull requests with simplifications.

**Schedule**: Daily or on-demand

**What it does**:
- Identifies overly complex code patterns
- Suggests refactoring opportunities
- Creates pull requests with simplifications
- Reduces cyclomatic complexity

### static-analysis-report
Scans agentic workflows for security vulnerabilities using zizmor, poutine, and actionlint.

**Schedule**: Daily and on pull requests

**What it does**:
- Scans workflows for security vulnerabilities
- Detects insecure patterns and practices
- Checks for GitHub Actions best practices
- Creates detailed security reports

### repository-quality-improver
Daily analysis and improvement of repository quality focusing on different lifecycle areas.

**Schedule**: Daily

**What it does**:
- Analyzes repository structure and organization
- Identifies quality improvement opportunities
- Creates issues for tracked improvements
- Monitors overall repository health

### cli-consistency-checker
Ensures CLI commands, flags, and help text follow consistent patterns.

**Schedule**: Daily or on-demand

**What it does**:
- Validates CLI command structure
- Checks for consistent help text
- Ensures proper flag handling
- Creates issues for inconsistencies

## Project Board

**URL**: https://github.com/orgs/githubnext/projects/TBD

The project board serves as the campaign dashboard, tracking:
- Active linting violations
- Code quality improvement tasks
- Security vulnerabilities
- Pull requests in review
- Completed improvements
- Overall campaign progress

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:linter-best-practices`

## Memory Paths

Campaign state and metrics are stored in:
- `memory/campaigns/linter-best-practices/**`

Metrics snapshots: `memory/campaigns/linter-best-practices/metrics/*.json`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 15
- **Max comments**: 10
- **Max new items added**: 8
- **Max issues created**: 5
- **Max pull requests**: 3

These limits ensure sustainable progress while preventing API rate limit exhaustion and maintaining manageable workload for reviewers.

### Quality Standards

All code changes must:
1. **Pass linting**: All automated linters must pass
2. **Follow style guides**: Consistent with existing code style
3. **Maintain tests**: All existing tests must pass
4. **Include documentation**: Update docs as needed
5. **Security first**: No new security vulnerabilities
6. **Reviewable**: Clear, focused changes

### Review Requirements

- All pull requests require human review before merge
- Security issues require immediate attention
- Breaking changes need stakeholder approval

## Workflow Enhancements (Future)

The campaign suggests enhancing these existing workflows:

### ci.yml
Potential additions:
- Go linting with golangci-lint
- Formatting checks (gofmt, prettier)
- Complexity analysis
- actionlint integration

### security-scan.yml
Potential additions:
- AI-powered vulnerability prioritization
- Automated remediation suggestions
- Dependency vulnerability tracking

## Risk Assessment

**Risk Level**: Low

This campaign:
- Focuses on code quality (not production systems)
- Requires human review for all changes
- Operates with rate limits and governance controls
- Cannot directly modify protected branches
- Maintains audit trail in repo-memory

## Timeline

- **Start Date**: TBD (upon activation)
- **Initial Sprint**: 30 days to reduce violations by 80%
- **Target Completion**: Ongoing (continuous improvement)
- **Estimated Duration**: Continuous

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/linter-best-practices.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-label
- Adds new issues to the project board
- Updates issue status based on state changes
- Aggregates metrics from all linting workflows
- Reports campaign progress and quality trends

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination and decision-making happens in the orchestrator
- The GitHub Project board is the single source of truth for campaign state
- Safe outputs include appropriate AI-generated footers for transparency
- This campaign complements (not replaces) existing CI checks
- Focus on incremental improvement rather than massive refactoring
