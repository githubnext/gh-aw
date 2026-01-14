---
id: lint-quality-2026
name: "Lint Quality Campaign 2026"
description: "Maintain consistent code formatting and proactively identify linting issues across Go, JavaScript, and TypeScript files while detecting security vulnerabilities early through static analysis"
version: v1
state: active

project-url: "https://github.com/orgs/githubnext/projects/TBD"  # Will be created by safe-outputs handler job
tracker-label: "campaign:lint-quality-2026"

objective: "Maintain consistent code formatting across Go, JavaScript, and TypeScript files, identify and fix linting issues proactively before they reach production, detect security vulnerabilities early through static analysis, ensure Markdown documentation quality, and reduce technical debt through automated code quality checks"

kpis:
  - name: "Linting issues resolved"
    priority: primary
    unit: count
    baseline: 0
    target: 50
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Security vulnerabilities detected"
    priority: supporting
    unit: count
    baseline: 0
    target: 20
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Code quality score"
    priority: supporting
    unit: percent
    baseline: 85
    target: 95
    time-window-days: 30
    direction: increase
    source: custom

# Workflows to execute
workflows:
  - super-linter
  - tidy
  - static-analysis-report

# Governance
governance:
  max-project-updates-per-run: 20
  max-comments-per-run: 15
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 10

owners:
  - "githubnext"

risk-level: low

tags:
  - code-quality
  - linting
  - security
  - maintenance
  - formatting

allowed-safe-outputs:
  - create-issue
  - create-pull-request
  - update-project
  - create-discussion
---

# Lint Quality Campaign 2026

## Overview

This campaign maintains high code quality standards across the GitHub Agentic Workflows repository through automated linting, formatting, and security scanning. By running multiple quality-focused workflows, the campaign ensures consistent code style, identifies issues proactively, and reduces technical debt.

## Objective

**Maintain consistent code formatting across Go, JavaScript, and TypeScript files, identify and fix linting issues proactively before they reach production, detect security vulnerabilities early through static analysis, ensure Markdown documentation quality, and reduce technical debt through automated code quality checks**

Quality automation is essential for maintaining a healthy codebase. This campaign coordinates multiple linting and analysis workflows to systematically improve and maintain code excellence.

## Success Criteria

- All Go, JavaScript, and TypeScript code follows consistent formatting standards
- Linting issues are identified and resolved within 48 hours
- Security vulnerabilities are detected and addressed proactively
- Markdown documentation passes quality checks
- Technical debt trends downward over time
- Automated code quality checks run daily

## Key Performance Indicators

### Primary KPI: Linting Issues Resolved
- **Baseline**: 0 (starting point)
- **Target**: 50 (cumulative fixes)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from workflow outputs

This KPI tracks the number of linting issues successfully identified and fixed through the campaign, demonstrating proactive code quality improvement.

### Supporting KPI: Security Vulnerabilities Detected
- **Baseline**: 0 (starting point)
- **Target**: 20 (cumulative detections)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from static-analysis-report

This KPI measures how many security vulnerabilities are caught early through static analysis (zizmor, poutine, actionlint), preventing them from reaching production.

### Supporting KPI: Code Quality Score
- **Baseline**: 85% (current estimated quality)
- **Target**: 95% (excellent quality)
- **Time Window**: 30 days (rolling)
- **Direction**: Increase
- **Source**: Custom metrics from all linting workflows

This KPI represents an overall code quality score derived from linting coverage, issue resolution rate, and security scan results.

## Associated Workflows

### super-linter
Runs Markdown quality checks using Super Linter and creates issues for violations.
- **Schedule**: Weekdays at 2 PM UTC (cron: `0 14 * * 1-5`)
- **Trigger**: workflow_dispatch
- **Purpose**: Ensure Markdown documentation quality

### tidy
Automatically formats and tidies code files (Go, JS, TypeScript) and creates pull requests with fixes.
- **Schedule**: Daily at 7 AM UTC (cron: `0 7 * * *`)
- **Trigger**: workflow_dispatch, slash_command, push to main
- **Purpose**: Maintain consistent code formatting and fix linting issues

### static-analysis-report
Security scanning with zizmor, poutine, and actionlint to detect vulnerabilities.
- **Schedule**: Daily
- **Trigger**: workflow_dispatch
- **Purpose**: Proactively identify security vulnerabilities and code quality issues

## Project Board

**URL**: TBD (will be created by safe-outputs handler job)

The project board serves as the campaign dashboard, tracking:
- Linting issues discovered
- Code formatting PRs
- Security vulnerabilities detected
- Issue resolution progress
- Overall campaign health

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:lint-quality-2026`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 20
- **Max comments**: 15
- **Max new items added**: 10
- **Max discovery items scanned**: 100
- **Max discovery pages**: 10

These limits ensure sustainable progress while preventing API rate limit exhaustion and maintaining manageable workload for reviewers.

### Quality Standards

All code quality improvements must:
1. **Follow project conventions**: Adhere to existing code style and formatting
2. **Be automatically validated**: Pass linting and tests before merging
3. **Include clear descriptions**: Explain what was fixed and why
4. **Be non-breaking**: Maintain backward compatibility
5. **Address root causes**: Fix underlying issues, not just symptoms

### Review Requirements

- Formatting PRs from tidy workflow can be auto-merged after tests pass
- Security vulnerability fixes require human review
- Breaking changes need stakeholder approval

## Risk Assessment

**Risk Level**: Low

This campaign focuses on code quality and formatting improvements that require validation through tests and human review before merging.

## Timeline

- **Start Date**: 2026-01-14
- **Target Completion**: Ongoing (continuous quality maintenance)

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination and decision-making happens in the campaign orchestrator
- The GitHub Project board is the single source of truth for campaign state
- Safe outputs include appropriate AI-generated footers for transparency
- Project board URL will be updated after the safe-outputs handler job creates it
