---
title: Campaign Examples
description: Example campaign workflows demonstrating worker orchestration and pattern analysis
sidebar:
  badge: { text: 'Examples', variant: 'note' }
---

This section contains example campaign workflows that demonstrate how to use campaign worker orchestration, workflow discovery, and the dispatch_workflow safe output.

## Security Audit Campaign

[**Security Audit 2026**](/gh-aw/examples/campaigns/security-auditcampaign/) - A comprehensive security audit campaign that demonstrates:

- **Worker Discovery**: Finding existing security-related workflows
- **Workflow Fusion**: Adapting workflows with `workflow_dispatch` triggers
- **Orchestration**: Using `dispatch_workflow` to coordinate multiple workers
- **KPI Tracking**: Measuring vulnerability reduction over time
- **Pattern Analysis**: Organizing workers in campaign-specific folders

### Key Features

- 3 worker workflows (scanner, updater, reporter)
- Governance policies for pacing and opt-out
- Quarterly timeline with weekly status updates
- Executive sponsorship and risk management

### Worker Example

[**Security Scanner**](/gh-aw/examples/campaigns/security-scanner/) - An example security scanner workflow that:

- Runs on a schedule (weekly)
- Creates issues for vulnerabilities
- Uses tracker-id for campaign discovery
- Can be dispatched by campaign orchestrators

## Using These Examples

### 1. Campaign Spec Structure

Campaign specs (`.campaign.md` files) define:
- Campaign goals and KPIs
- Worker workflows to orchestrate
- Memory paths for state persistence
- Governance and pacing policies

### 2. Worker Workflow Pattern

Worker workflows should:
- Support workflow_dispatch for orchestration
- Focus on specific, repeatable tasks
- Be campaign-agnostic (reusable)
- Optionally include tracker-id in frontmatter to add tracking metadata to created issues/PRs

### 3. Folder Organization

```
docs/src/content/docs/examples/campaigns/
├── security-audit.campaign.md      # Campaign spec
└── security-scanner.md             # Example worker workflow

.github/workflows/campaigns/
└── security-audit-2026/            # Fused workers at runtime
    ├── security-scanner-worker.md
    └── ...
```

## Learn More

- [Campaign Guides](/gh-aw/guides/campaigns/) - Complete campaign documentation
- [Flow & lifecycle](/gh-aw/guides/campaigns/flow/) - How the orchestrator runs
- [Dispatch Workflow](/gh-aw/guides/dispatchops/) - Using workflow_dispatch
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - dispatch_workflow configuration

## Pattern Analysis

These examples are organized to enable future pattern analysis:
- Which workflows work best for security campaigns?
- What KPIs are most effective for different campaign types?
- How should workers be organized for optimal results?

The separate folder structure allows tracking and learning from campaign outcomes over time.
