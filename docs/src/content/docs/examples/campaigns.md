---
title: Campaign Examples
description: Example campaign workflows demonstrating worker orchestration with standardized contracts
sidebar:
  badge: { text: 'Examples', variant: 'note' }
---

This section contains example campaign workflows that demonstrate how to use first-class campaign workers with standardized input contracts and idempotency.

## Security Audit Campaign

[**Security Audit 2026**](/gh-aw/examples/campaigns/security-auditcampaign/) - A comprehensive security audit campaign that demonstrates:

- **Worker Discovery**: Finding security-related issues and PRs via tracker labels
- **Dispatch-Only Workers**: Workers designed specifically for campaign orchestration
- **Standardized Contract**: All workers accept `campaign_id` and `payload` inputs
- **Idempotency**: Workers check for existing work before creating duplicates
- **KPI Tracking**: Measuring vulnerability reduction over time

### Key Features

- 3 dispatch-only worker workflows (scanner, fixer, reviewer)
- Governance policies for pacing and opt-out
- Deterministic work item keys to prevent duplicates
- Quarterly timeline with weekly status updates
- Executive sponsorship and risk management

### Worker Example

[**Security Scanner**](/gh-aw/examples/campaigns/security-scanner/) - An example security scanner workflow that:

- Accepts `campaign_id` and `payload` inputs via workflow_dispatch
- Uses deterministic keys for branch names and PR titles
- Checks for existing PRs before creating new ones
- Labels all created items with `campaign:{id}` for tracking
- Reports completion status back to orchestrator

## Using These Examples

### 1. Campaign Spec Structure

Campaign specs (`.campaign.md` files) define:
- Campaign goals and KPIs
- Worker workflows to reference (by name)
- Discovery scope (repos/orgs to search)
- Memory paths for state persistence
- Governance and pacing policies

### 2. Worker Workflow Pattern (Dispatch-Only)

Worker workflows MUST:
- Use `workflow_dispatch` as the ONLY trigger (no schedule/push/pull_request)
- Accept standardized inputs: `campaign_id` (string) and `payload` (string; JSON)
- Implement idempotency via deterministic work item keys
- Label all created items with `campaign:{campaign_id}`
- Focus on specific, repeatable tasks

Example:
```yaml
on:
  workflow_dispatch:
    inputs:
      campaign_id:
        description: 'Campaign identifier'
        required: true
        type: string
      payload:
        description: 'JSON payload with work item details'
        required: true
        type: string
```

### 3. Idempotency Requirements

Workers prevent duplicates by:
1. Computing deterministic keys: `campaign-{campaign_id}-{repository}-{work_item_id}`
2. Using keys in branch names, PR titles, issue titles
3. Checking for existing work with the key before creating
4. Skipping or updating existing items rather than creating duplicates

### 4. Folder Organization

```
.github/workflows/
├── my-campaign.campaign.md         # Campaign spec
├── my-worker.md                    # Worker workflow (dispatch-only)
└── my-campaign.campaign.lock.yml   # Compiled orchestrator

docs/
└── campaign-workers.md             # Worker pattern documentation
```

Workers are stored alongside regular workflows, not in campaign-specific folders. The dispatch-only trigger makes ownership clear.

## Learn More

- [Campaign Workers Guide](/gh-aw/campaign-workers/) - Complete worker pattern documentation
- [Campaign Guides](/gh-aw/guides/campaigns/) - Campaign setup and configuration
- [Flow & lifecycle](/gh-aw/guides/campaigns/flow/) - How the orchestrator runs
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - dispatch_workflow configuration

## Pattern Analysis

These examples demonstrate best practices for campaign workers:
- **Explicit ownership**: Workers are dispatch-only, clearly orchestrated
- **Standardized contract**: All workers use the same input format
- **Idempotent behavior**: Workers avoid duplicate work across runs
- **Deterministic keys**: Enable reliable duplicate detection
- **Simple units**: Workers are focused, stateless, deterministic

The dispatch-only pattern eliminates confusion about trigger precedence and makes orchestration explicit.
