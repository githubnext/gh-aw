---
id: docs-quality-maintenance-project73
name: "Campaign: Documentation Quality & Maintenance (Project 73)"
description: "Systematically improve documentation quality, consistency, and maintainability. Success: all docs follow Diátaxis framework, maintain accessibility standards, and pass quality checks."
version: v1
# Using Claude engine until Copilot is fixed
engine: claude
project-url: "https://github.com/orgs/githubnext/projects/73"
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
workflows:
  - daily-doc-updater
  - docs-noob-tester
  - daily-multi-device-docs-tester
  - unbloat-docs
  - developer-docs-consolidator
  - technical-doc-writer
tracker-label: "campaign:docs-quality-maintenance-project73"
memory-paths:
  - "memory/campaigns/docs-quality-maintenance-project73/**"
metrics-glob: "memory/campaigns/docs-quality-maintenance-project73/metrics/*.json"
cursor-glob: "memory/campaigns/docs-quality-maintenance-project73/cursor.json"
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

# Documentation Quality & Maintenance Campaign (Project 73)

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

**URL**: https://github.com/orgs/githubnext/projects/73

The project board serves as the campaign dashboard, tracking:
- Documentation gaps and coverage
- Quality improvement tasks
- Accessibility issues
- User-reported problems
- PRs in review
- Completed improvements
- Overall campaign progress

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:docs-quality-maintenance-project73`

## Memory Paths

Campaign state and metrics are stored in:
- `memory/campaigns/docs-quality-maintenance-project73/**`

Metrics snapshots: `memory/campaigns/docs-quality-maintenance-project73/metrics/*.json`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 15
- **Max comments**: 10
- **Max new items added**: 8
- **Max discovery items scanned**: 100
- **Max discovery pages**: 10

These limits ensure sustainable progress while preventing API rate limit exhaustion and maintaining manageable workload for reviewers.

### Quality Standards

All documentation changes must:
1. **Follow Diátaxis framework**: Clearly categorize content as Tutorial, How-to, Reference, or Explanation
2. **Maintain accessibility**: Pass WCAG 2.1 AA standards
3. **Use proper formatting**: Follow Astro Starlight markdown conventions
4. **Include examples**: Provide practical code samples where appropriate
5. **Be technically accurate**: Match current codebase behavior
6. **Maintain consistent tone**: Neutral, technical, not promotional

### Review Requirements

- All documentation PRs require human review before merge
- Accessibility issues require immediate attention
- Breaking documentation changes need stakeholder approval

## Risk Assessment

**Risk Level**: Low

This campaign does not modify production code and requires human review for all changes.

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/docs-quality-maintenance-project73.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-id
- Adds new issues to the project board
- Updates issue status based on state changes
- Aggregates metrics from all documentation workflows
- Reports campaign progress and quality trends

## Notes

- Workers remain campaign-agnostic and immutable
- All coordination and decision-making happens in the orchestrator
- The GitHub Project board is the single source of truth for campaign state
- Safe outputs include appropriate AI-generated footers for transparency
