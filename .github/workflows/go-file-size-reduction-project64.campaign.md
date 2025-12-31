---
id: go-file-size-reduction-project64
name: "Go File Size Reduction Campaign (Project 64)"
description: "Systematically reduce oversized Go files to improve maintainability. Success: all files ≤800 LOC, maintain coverage, no regressions."
version: v1
project-url: "https://github.com/orgs/githubnext/projects/64"
project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"
workflows:
  - daily-file-diet
tracker-label: "campaign:go-file-size-reduction-project64"
memory-paths:
  - "memory/campaigns/go-file-size-reduction-project64/**"
metrics-glob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json"
state: active
tags:
  - code-quality
  - maintainability
  - refactoring
risk-level: low
allowed-safe-outputs:
  - add-comment
  - update-project
objective: "Reduce all Go files to ≤800 lines of code while maintaining test coverage and preventing regressions"
kpis:
  - name: "Files reduced to target size"
    priority: primary
    unit: percent
    baseline: 0
    target: 100
    time-window-days: 90
    direction: increase
    source: custom
  - name: "Test coverage maintained"
    priority: supporting
    unit: percent
    baseline: 80
    target: 80
    time-window-days: 7
    direction: increase
    source: ci
governance:
  max-project-updates-per-run: 10
  max-comments-per-run: 10
  max-new-items-per-run: 5
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 5
---

# Go File Size Reduction Campaign (Project 64)

## Overview

This campaign systematically identifies and refactors oversized Go files in the codebase to improve maintainability, reduce cognitive load, and enhance code quality.

## Objective

**Reduce all Go files to ≤800 lines of code while maintaining test coverage and preventing regressions**

Large files (>800 LOC) are harder to understand, test, and maintain. This campaign breaks them down into focused, cohesive modules following Go best practices.

## Success Criteria

- ✅ All Go files are ≤800 lines of code
- ✅ Test coverage remains at or above baseline (80%)
- ✅ No functionality regressions introduced
- ✅ Code follows established patterns and conventions

## Key Performance Indicators

### Primary KPI: Files Reduced to Target Size
- **Baseline**: 0% (starting point)
- **Target**: 100% (all files under 800 LOC)
- **Time Window**: 90 days
- **Direction**: Increase
- **Source**: Custom metrics from file analysis

### Supporting KPI: Test Coverage Maintained
- **Baseline**: 80%
- **Target**: 80% (maintain or improve)
- **Time Window**: 7 days (rolling)
- **Direction**: Increase
- **Source**: CI metrics

## Associated Workflows

### daily-file-diet
Primary worker workflow that:
- Identifies oversized Go files (>800 LOC)
- Creates issues for refactoring tasks
- Tracks progress on the project board
- Embeds tracker-id for campaign correlation

## Project Board

**URL**: https://github.com/orgs/githubnext/projects/64

The project board serves as the primary campaign dashboard, tracking:
- Open refactoring tasks
- In-progress work
- Completed file reductions
- Overall campaign progress

## Tracker Label

All campaign-related issues and PRs are tagged with: `campaign:go-file-size-reduction-project64`

## Memory Paths

Campaign state and metrics are stored in:
- `memory/campaigns/go-file-size-reduction-project64-*/**`

Metrics snapshots: `memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json`

## Governance Policies

### Rate Limits (per run)
- **Max project updates**: 10
- **Max comments**: 10
- **Max new items added**: 5
- **Max discovery items scanned**: 50
- **Max discovery pages**: 5

These limits ensure gradual, sustainable progress without overwhelming the team or API rate limits.

## Risk Assessment

**Risk Level**: Low

This campaign:
- Does not modify production code directly
- Requires human review for all changes
- Maintains test coverage requirements
- Uses incremental, reversible refactoring approaches

## Campaign Lifecycle

1. **Discovery**: Identify oversized files using automated analysis
2. **Prioritization**: Order files by size, complexity, and impact
3. **Execution**: Create issues for refactoring tasks
4. **Review**: Human developers implement and review changes
5. **Verification**: Automated tests confirm no regressions
6. **Tracking**: Update project board with progress

## Orchestrator

This campaign uses an automatically generated orchestrator workflow:
- **File**: `.github/workflows/go-file-size-reduction-project64.campaign.g.md`
- **Schedule**: Daily at 18:00 UTC (cron: `0 18 * * *`)
- **Purpose**: Coordinate worker outputs and update project board

The orchestrator:
- Discovers worker-created issues via tracker-id
- Adds new issues to the project board
- Updates issue status based on state changes
- Reports campaign progress and metrics

## Notes

- Workers (`daily-file-diet`) remain campaign-agnostic and immutable
- All coordination and decision-making happens in the orchestrator
- The GitHub Project board is the single source of truth for campaign state
- Safe outputs include appropriate AI-generated footers for transparency
