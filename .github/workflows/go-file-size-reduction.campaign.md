---
id: go-file-size-reduction
version: "v1"
name: "Go File Size Reduction Campaign"
description: "Reduce oversized non-test Go files under pkg/ to ≤800 LOC via tracked refactors, with daily metrics snapshots and a GitHub Projects dashboard."

project-url: "https://github.com/orgs/githubnext/projects/60"

workflows:
  - daily-file-diet

memory-paths:
  - "memory/campaigns/go-file-size-reduction-*/**"

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

tracker-label: "campaign:go-file-size-reduction"

metrics-glob: "memory/campaigns/go-file-size-reduction-*/metrics/*.json"

allowed-safe-outputs:
  - "create-issue"
  - "add-comment"
  - "update-project"

approval-policy:
  required-approvals: 1
  required-roles:
    - "platform-eng-lead"
  change-control: false
---

# Go File Size Reduction Campaign

This campaign reduces oversized Go files over time and makes
maintainability measurable.

- **Goal**: No non-test Go file over 800 LOC in `pkg/` by **<YYYY-MM-DD>**
- **Scope**: All non-test files under `pkg/`
- **Tracking**: Issues labeled `campaign:go-file-size-reduction` and
  `campaign-tracker` epic
- **Metrics**: Daily snapshots of largest-file size and count of
  over-threshold files stored under
  `memory/campaigns/go-file-size-reduction-*/metrics/`
- **Project Board**: GitHub Projects board named `Go File
  Size Reduction Campaign` used as the primary campaign dashboard. The
  `daily-file-diet` workflow keeps this board in sync via the
  `update-project` safe-output.

One-time setup (manual): create the Project in the GitHub UI and set its URL in
`project-url`. Views (board/table, grouping, filters) are configured in the UI;
workflows can update items and fields, but they do not currently create or
configure Project views.

Use this spec as the enterprise-facing description of the campaign for
owners, sponsors, and reporting.
