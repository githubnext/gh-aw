---
id: docs-quality-maintenance-project69
name: "Campaign: Documentation Quality & Maintenance (Project 69)"
description: "Systematically improve documentation quality, consistency, and maintainability. Success: all docs follow Diátaxis framework, maintain accessibility standards, and pass quality checks."
version: v1
engine: claude
project-url: "https://github.com/orgs/githubnext/projects/69"
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
workflows:
  - daily-doc-updater
  - docs-noob-tester
  - daily-multi-device-docs-tester
  - unbloat-docs
  - developer-docs-consolidator
  - technical-doc-writer
tracker-label: "campaign:docs-quality-maintenance-project69"
memory-paths:
  - "memory/campaigns/docs-quality-maintenance-project69/**"
metrics-glob: "memory/campaigns/docs-quality-maintenance-project69/metrics/*.json"
cursor-glob: "memory/campaigns/docs-quality-maintenance-project69/cursor.json"
state: active
tags:
  - documentation
  - quality
  - maintainability
  - accessibility
  - user-experience
risk-level: low
allowed-safe-outputs:
  - add-comment
  - update-project
  - create-pull-request
  - create-discussion
  - upload-asset
objective: "Maintain high-quality, accessible, and consistent documentation following the Diátaxis framework while ensuring all docs are accurate, complete, and user-friendly"
kpis:
  - name: "Documentation coverage of features"
    priority: primary
    unit: percent
    baseline: 85
    target: 95
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Documentation accessibility score"
    priority: supporting
    unit: percent
    baseline: 90
    target: 98
    time-window-days: 30
    direction: increase
    source: custom
  - name: "User-reported documentation issues"
    priority: supporting
    unit: count
    baseline: 15
    target: 5
    time-window-days: 30
    direction: decrease
    source: pull_requests
governance:
  max-project-updates-per-run: 15
  max-comments-per-run: 10
  max-new-items-per-run: 8
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 10
---

# Documentation Quality & Maintenance Campaign (Project 69)

## Overview

This campaign ensures the GitHub Agentic Workflows documentation maintains the highest quality standards, remains accurate and up-to-date, follows the Diátaxis framework, and provides an excellent user experience across all devices and accessibility requirements.

## Objective

**Maintain high-quality, accessible, and consistent documentation following the Diátaxis framework while ensuring all docs are accurate, complete, and user-friendly**

High-quality documentation is critical for user adoption and success. This campaign coordinates multiple documentation workflows to systematically improve and maintain documentation excellence.

## Success Criteria

- All documentation follows the Diátaxis framework (Tutorial, How-to, Reference, Explanation)
- Documentation coverage reaches 95% of user-facing features
- Accessibility score maintains 98% or higher
- User-reported documentation issues decrease to ≤5 per month
- All documentation passes automated quality checks
- Documentation site performs well across mobile, tablet, and desktop devices

## Key Performance Indicators

### Primary KPI: Documentation Coverage of Features
- **Baseline**: 85% (current estimated coverage)
- **Target**: 95% (comprehensive feature documentation)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from feature analysis

This KPI tracks what percentage of user-facing features have complete documentation. Coverage includes CLI commands, workflow configurations, safe outputs, tools, and all major features.

### Supporting KPI: Documentation Accessibility Score
- **Baseline**: 90% (current accessibility compliance)
- **Target**: 98% (near-perfect accessibility)
- **Time Window**: 30 days (rolling)
- **Direction**: Increase
- **Source**: Custom metrics from Playwright accessibility testing

This KPI measures WCAG 2.1 AA compliance across the documentation site, including keyboard navigation, screen reader support, color contrast, and semantic HTML.

### Supporting KPI: User-Reported Documentation Issues
- **Baseline**: 15 per month (current average)
- **Target**: 5 per month (minimal confusion)
- **Time Window**: 30 days (rolling)
- **Direction**: Decrease
- **Source**: Pull requests and issues labeled "documentation"

Lower user-reported issues indicate clearer, more complete documentation that addresses user needs effectively.

## Associated Workflows

### daily-doc-updater
Automatically reviews and updates documentation based on recent code changes. Daily at 6am UTC.

### docs-noob-tester
Tests documentation from a beginner's perspective. Daily.

### daily-multi-device-docs-tester
Tests documentation site across mobile, tablet, and desktop devices. Daily.

### unbloat-docs
Reviews and simplifies documentation by reducing verbosity. Daily, or via `/unbloat` command in PR comments.

### developer-docs-consolidator
Consolidates developer documentation from `specs/` directory. Daily at 3:17 AM UTC.

### technical-doc-writer
Creates or enhances technical documentation for complex features. On-demand or scheduled.

## Project Board

**URL**: https://github.com/orgs/githubnext/projects/69

The project board serves as the campaign dashboard, tracking:
- Documentation gaps and coverage
- Quality improvement tasks
- Accessibility issues
- User-reported problems
- PRs in review
- Completed improvements

### Project Board Views

1. **Roadmap View**: Timeline-based view showing documentation initiatives over time
2. **Task Tracker**: Kanban board with columns: Backlog, In Progress, Review, Done
3. **Progress Board**: Status-based view tracking documentation coverage and quality metrics

### Project Board Fields

- **Worker/Workflow**: Which workflow is handling this task
- **Priority**: High, Medium, Low
- **Status**: Not Started, In Progress, Blocked, Review, Done
- **Start Date**: When work began
- **Target Date**: Expected completion
- **Effort**: Story points (1, 2, 3, 5, 8, 13)

## Workflow Coordination

The campaign orchestrator coordinates these workflows:

1. **Discovery**: Identifies documentation gaps, outdated content, accessibility issues
2. **Assignment**: Routes work items to appropriate workflows based on task type
3. **Tracking**: Updates project board with progress
4. **Metrics**: Collects and reports KPI measurements

## Memory & State Management

The campaign uses repository memory for state persistence:

- **Cursor**: `memory/campaigns/docs-quality-maintenance-project69/cursor.json`
- **Metrics**: `memory/campaigns/docs-quality-maintenance-project69/metrics/*.json`
- **Work Items**: `memory/campaigns/docs-quality-maintenance-project69/items/*.json`

## Governance Rules

- Maximum 8 new work items created per campaign run
- Maximum 100 items discovered per run (with pagination support)
- Maximum 15 project updates per run
- Maximum 10 comments per run
- Items with `no-campaign` or `no-bot` labels are excluded

## Risk Assessment

**Risk Level**: Low

This campaign is low-risk because:
- Documentation changes don't affect production code
- All changes go through PR review process
- Campaign has strict governance limits
- Workflows operate in isolated sandboxes
- Changes are incremental and reversible

## Success Metrics

Track campaign success through:
- Documentation coverage percentage trending toward 95%
- Accessibility score maintaining 98%+
- User-reported documentation issues declining to ≤5/month
- All workflows completing successfully
- Project board items moving through workflow stages

## Campaign Timeline

- **Phase 1** (Weeks 1-4): Audit existing documentation, identify gaps
- **Phase 2** (Weeks 5-8): Fill critical gaps, improve accessibility
- **Phase 3** (Weeks 9-12): Optimize for user experience, reduce verbosity
- **Ongoing**: Continuous monitoring and incremental improvements
