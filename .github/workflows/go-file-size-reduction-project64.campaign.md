---
id: go-file-size-reduction-project64
version: "v1"
name: "Go File Size Reduction Campaign (Project 64)"
description: "Systematically reduce oversized Go files to improve maintainability. Success: all files ≤800 LOC, maintain coverage, no regressions."

project-url: "https://github.com/orgs/githubnext/projects/64"

workflows:
  - daily-file-diet

memory-paths:
  - "memory/campaigns/go-file-size-reduction-project64-*/**"

owners:
  - "platform-engineering"
  - "developer-experience"

executive-sponsors:
  - "vp-engineering"

risk-level: "medium"
state: "active"
tags:
  - "code-health"
  - "refactoring"
  - "maintainability"
  - "technical-debt"

tracker-label: "campaign:go-file-size-reduction-project64"

metrics-glob: "memory/campaigns/go-file-size-reduction-project64-*/metrics/*.json"

allowed-safe-outputs:
  - "create-issue"
  - "add-comment"
  - "upload-assets"
  - "update-project"

approval-policy:
  required-approvals: 1
  required-roles:
    - "platform-eng-lead"
  change-control: false
---

# Go File Size Reduction Campaign (Project 64)

This campaign systematically reduces oversized Go files in the codebase to improve maintainability and code quality.

## Campaign Objectives

- **Goal**: Reduce all non-test Go files to ≤800 lines of code (LOC)
- **Scope**: All non-test Go files under `pkg/` directory
- **Success Criteria**:
  - All files ≤800 LOC
  - Maintain test coverage
  - No regressions in functionality
  - Improved code maintainability

## Tracking

- **Issues**: Labeled with `campaign:go-file-size-reduction-project64`
- **Project Board**: [GitHub Project #64](https://github.com/orgs/githubnext/projects/64)
- **Metrics**: Daily snapshots stored under `memory/campaigns/go-file-size-reduction-project64-*/metrics/`

## Approach

1. **Identify**: Scan `pkg/` for Go files exceeding 800 LOC
2. **Prioritize**: Focus on largest files first for maximum impact
3. **Refactor**: Break down oversized files into logical, focused modules
4. **Validate**: Ensure all tests pass and coverage is maintained
5. **Track**: Monitor progress through metrics and project board

## Workflow Integration

The `daily-file-diet` worker workflow and campaign orchestrator work together:
- **Worker** (`daily-file-diet`): Scans for oversized Go files independently, creates tracking issues, records daily metrics snapshots
- **Orchestrator**: Monitors worker workflow runs (via `tracker-id`), discovers issues created by workers, adds them to project board, updates board status, reports on campaign progress

## Setup

**One-time manual setup**: Create the GitHub Project in the UI and configure views (board/table, grouping, filters). The workflows will update items and fields but do not create or configure Project views.

## Governance

- **Risk Level**: Medium - refactoring existing code carries moderate risk
- **Approvals**: Requires 1 approval from platform-eng-lead
- **Change Control**: Not required for routine refactoring PRs

Use this specification as the authoritative description of the campaign for owners, sponsors, and reporting purposes.
