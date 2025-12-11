---
id: code-health-file-diet
version: "v1"
name: "Code Health: File Diet"
description: "Reduce oversized Go files across the codebase with tracked refactors."

workflows:
  - daily-file-diet

memory-paths:
  - "memory/campaigns/code-health-file-diet-*/**"

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

tracker-label: "campaign:code-health-file-diet"

metrics-glob: "memory/campaigns/code-health-file-diet-*/metrics/*.json"

allowed-safe-outputs:
  - "create-issue"
  - "add-comment"

approval-policy:
  required-approvals: 1
  required-roles:
    - "platform-eng-lead"
  change-control: false
---

# Code Health: File Diet Campaign

This campaign reduces oversized Go files over time and makes
maintainability measurable.

- **Goal**: No Go file over 800 LOC by a target date
- **Scope**: All non-test files under `pkg/`
- **Tracking**: Issues labeled `campaign:code-health-file-diet` and
  `campaign-tracker` epic
- **Metrics**: Daily snapshots of largest-file size and count of
  over-threshold files stored under
  `memory/campaigns/code-health-file-diet-*/metrics/`

Use this spec as the enterprise-facing description of the campaign for
owners, sponsors, and reporting.
