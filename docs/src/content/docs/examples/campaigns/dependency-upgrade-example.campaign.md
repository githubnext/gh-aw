---
id: dependency-upgrade-example
name: Dependency Upgrade Example Campaign
description: Example campaign demonstrating bootstrap and worker metadata features
project-url: https://github.com/orgs/example/projects/1
version: v1
state: planned

# Bootstrap configuration - runs when discovery returns 0 items
bootstrap:
  mode: seeder-worker
  seeder-worker:
    workflow-id: dependency-scanner
    payload:
      scan-type: outdated
      ecosystems:
        - npm
        - pip
        - go
      severity: medium
    max-items: 20

# Worker metadata - defines capabilities and contracts
workers:
  - id: dependency-scanner
    name: Dependency Scanner
    description: Scans repositories for outdated dependencies
    capabilities:
      - scan-dependencies
      - discover-outdated-packages
    payload-schema:
      scan-type:
        type: string
        description: Type of scan (outdated, vulnerable, all)
        required: true
        example: outdated
      ecosystems:
        type: array
        description: Package ecosystems to scan
        required: false
        example: ["npm", "pip"]
      severity:
        type: string
        description: Minimum severity level to report
        required: false
        example: medium
    output-labeling:
      tracker-label: campaign:dependency-upgrade-example
      additional-labels:
        - dependencies
        - automated-scan
      key-in-title: true
      key-format: "scan-{repository}-{scan_type}"
      metadata-fields:
        - Campaign Id
        - Worker Workflow
        - Scan Type
    idempotency-strategy: issue-title-based
    priority: 5

  - id: dependency-updater
    name: Dependency Updater
    description: Creates PRs to update outdated dependencies
    capabilities:
      - update-dependencies
      - create-pull-requests
    payload-schema:
      repository:
        type: string
        description: Target repository in owner/repo format
        required: true
        example: owner/repo
      package_name:
        type: string
        description: Package name to update
        required: true
        example: express
      current_version:
        type: string
        description: Current package version
        required: true
        example: 4.17.1
      target_version:
        type: string
        description: Target package version
        required: true
        example: 4.18.2
      ecosystem:
        type: string
        description: Package ecosystem
        required: true
        example: npm
    output-labeling:
      tracker-label: campaign:dependency-upgrade-example
      additional-labels:
        - dependencies
        - automated-pr
      key-in-title: true
      key-format: "campaign-{campaign_id}-{repository}-{package_name}-{target_version}"
      metadata-fields:
        - Campaign Id
        - Worker Workflow
        - Package Name
        - Current Version
        - Target Version
    idempotency-strategy: pr-title-based
    priority: 10

workflows:
  - dependency-scanner
  - dependency-updater

tracker-label: campaign:dependency-upgrade-example
memory-paths:
  - memory/campaigns/dependency-upgrade-example/**
metrics-glob: memory/campaigns/dependency-upgrade-example/metrics/*.json
cursor-glob: memory/campaigns/dependency-upgrade-example/cursor.json

objective: Systematically upgrade all outdated dependencies to reduce security vulnerabilities and improve maintainability

kpis:
  - name: Outdated Dependencies
    baseline: 150
    target: 20
    unit: packages
    time-window-days: 90
    priority: primary
    direction: decrease
    source: custom

  - name: PR Merge Rate
    baseline: 0.5
    target: 0.8
    unit: ratio
    time-window-days: 30
    priority: supporting
    direction: increase
    source: pull_requests

governance:
  max-new-items-per-run: 5
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  max-project-updates-per-run: 10
  max-comments-per-run: 5
  opt-out-labels:
    - no-campaign
    - skip-dependency-updates

owners:
  - "@engineering-team"
risk-level: medium

tags:
  - dependencies
  - security
  - maintenance

allowed-safe-outputs:
  - create-pull-request
  - add-comment
  - update-project
  - dispatch-workflow
---

# Dependency Upgrade Example Campaign

This example campaign demonstrates the bootstrap and worker metadata features.

## Bootstrap Phase (Discovery = 0)

When this campaign starts with no existing work items:

1. The orchestrator dispatches the `dependency-scanner` worker
2. The scanner discovers outdated dependencies across configured ecosystems
3. The scanner creates issues for each outdated package with proper labels
4. On the next orchestrator run, these issues become the work queue

## Worker Selection

The orchestrator uses worker metadata to make deterministic decisions:

### dependency-scanner (priority: 5)
- **Capabilities**: scan-dependencies, discover-outdated-packages
- **When to use**: Bootstrap phase or when re-scanning is needed
- **Output**: Issues listing outdated packages

### dependency-updater (priority: 10)
- **Capabilities**: update-dependencies, create-pull-requests  
- **When to use**: When an outdated package issue exists
- **Output**: PRs updating packages with proper tracking labels

## Idempotency Guarantees

Both workers ensure idempotent operation:

- **Scanner**: Uses issue title format `[scan-{repository}-{scan_type}]`
- **Updater**: Uses PR title format `[campaign-{campaign_id}-{repository}-{package}-{version}]`

Before creating new items, workers search for existing items with matching keys.

## Project Field Mapping

Workers populate these Project fields:
- Campaign Id: `dependency-upgrade-example`
- Worker Workflow: `dependency-scanner` or `dependency-updater`
- Package Name: Name of the package being updated
- Current Version: Current version number
- Target Version: Target version number

## Success Metrics

Track progress via:
- **Tasks Total**: Number of outdated packages identified
- **Tasks Completed**: Number of packages successfully updated
- **Velocity**: PRs merged per day
- **KPIs**: Reduction in outdated package count
