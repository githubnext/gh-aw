---
title: Quick Reference
description: At-a-glance reference for campaign commands, fields, and patterns
---

Quick reference for common campaign tasks, spec fields, and CLI commands.

## Creating a campaign

```bash
# Method 1: Automated (recommended)
# 1. Create GitHub issue
# 2. Apply label: create-agentic-campaign
# 3. Wait 2-3 minutes for PR

# Method 2: Manual (advanced)
gh aw campaign new my-campaign-id
# Edit .github/workflows/my-campaign-id.campaign.md
gh aw compile
```

## Essential CLI commands

```bash
# List all campaigns
gh aw campaign
gh aw campaign security              # Filter by ID/name

# Check campaign status
gh aw campaign status
gh aw campaign status --json

# Validate campaigns
gh aw campaign validate
gh aw campaign validate --no-strict

# Compile campaigns
gh aw compile                        # Compile all
gh aw compile my-campaign           # Compile specific
```

## Minimal campaign spec

```yaml
---
id: my-campaign
name: My Campaign
project-url: https://github.com/orgs/ORG/projects/1
tracker-label: campaign:my-campaign
allowed-repos:
  - "myorg/repo-a"
  - "myorg/repo-b"

objective: "Clear, measurable goal"
kpis:
  - id: main_metric
    name: "Main Metric"
    priority: primary
    target: 100
    direction: increase

workflows:
  - my-worker-workflow
---
```

## Common spec fields

| Field | Required | Description | Example |
|-------|----------|-------------|---------|
| `id` | Yes | Stable identifier | `framework-upgrade` |
| `name` | Yes | Display name | `"Framework Upgrade"` |
| `allowed-repos` | Yes | Repository scope | `["myorg/service-a"]` |
| `project-url` | No* | Project board URL | `https://github.com/orgs/ORG/projects/1` |
| `tracker-label` | No | Discovery label | `campaign:framework-upgrade` |
| `objective` | Recommended | Campaign goal | `"Upgrade all services to Node 20"` |
| `kpis` | Recommended | Success metrics | See KPI section below |
| `workflows` | No | Workflows to coordinate | `["scanner", "upgrader"]` |
| `governance` | No | Rate limits & policies | See Governance section below |

*Auto-created if not provided

## KPI structure

```yaml
kpis:
  - id: services_upgraded          # Unique identifier
    name: "Services Upgraded"      # Display name
    priority: primary              # Mark as primary KPI
    target: 50                     # Success threshold
    direction: increase            # increase | decrease | maintain
```

**Best practices:**
- 1 primary KPI + 2-3 supporting KPIs maximum
- Use clear, measurable metrics
- Set realistic targets based on historical data

## Governance settings

```yaml
governance:
  max-project-updates-per-run: 10           # Items to update per run
  max-discovery-items-per-run: 100          # Items to discover per run
  max-discovery-pages-per-run: 10           # API pages to fetch per run
  max-comments-per-run: 10                  # Comments to post per run
  opt-out-labels: ["no-campaign", "no-bot"] # Skip items with these labels
  do-not-downgrade-done-items: true         # Prevent moving items backward
```

**Default values:**
- `max-project-updates-per-run`: 10
- `max-discovery-items-per-run`: 100
- `max-discovery-pages-per-run`: 10
- `max-comments-per-run`: 10

**Guidance:**
- Start with low limits (10 updates/run)
- Increase gradually as you gain confidence
- Higher limits = faster progress but more API usage

## Campaign modes

| Mode | Trigger | Use Case |
|------|---------|----------|
| **ProjectOps** (default) | Workflows run independently | Track existing workflows, no orchestration |
| **Campaign Orchestration** | `execute-workflows: true` | Orchestrator runs workflows and drives progress |

```yaml
# Enable orchestration mode
execute-workflows: true
```

## Repo-memory paths

```yaml
memory-paths:
  - "memory/campaigns/my-campaign/**"
metrics-glob: "memory/campaigns/my-campaign/metrics/*.json"
cursor-glob: "memory/campaigns/my-campaign/cursor.json"
```

**Standard layout:**
```
memory/campaigns/<campaign-id>/
├── cursor.json                    # Pagination checkpoint
└── metrics/
    ├── 2026-01-17.json           # Daily snapshots
    ├── 2026-01-18.json
    └── 2026-01-19.json
```

## Worker workflow pattern

```yaml
---
on:
  workflow_dispatch:  # REQUIRED for campaign orchestration
# schedule: daily    # DISABLE if campaign orchestrates this workflow

tracker-id: my-worker-workflow  # Used for discovery

safe-outputs:
  create-issue:
    labels:
      - "campaign:my-campaign"  # Campaign tracker label
---

# Worker task description
Perform specific task and create issues with campaign label.
```

**Key points:**
- Use `workflow_dispatch` trigger only if orchestrated by campaign
- Add campaign tracker label to created issues/PRs
- Keep workers campaign-agnostic (no campaign-specific logic)

## Project board fields

Standard custom fields created automatically:

| Field | Type | Purpose |
|-------|------|---------|
| **Worker/Workflow** | Single select | Track workflow ownership |
| **Priority** | Single select | High, Medium, Low |
| **Status** | Single select | Todo, In Progress, Review, Blocked, Done |
| **Start Date** | Date | Timeline start (auto from `created_at`) |
| **End Date** | Date | Timeline end (auto from `closed_at`) |
| **Effort** | Single select | Small, Medium, Large |

## Common patterns

### Migration campaign

```yaml
objective: "Migrate all services to Framework vNext"
kpis:
  - id: services_migrated
    name: "Services Migrated"
    priority: primary
    target: 50
    direction: increase
workflows:
  - framework-scanner      # Discovers services needing migration
  - framework-migrator     # Performs migration
  - framework-validator    # Validates migrated services
```

### Security campaign

```yaml
objective: "Resolve all high-severity vulnerabilities"
kpis:
  - id: vulnerabilities_resolved
    name: "Vulnerabilities Resolved"
    priority: primary
    target: 0
    direction: decrease
  - id: mean_time_to_resolution
    name: "Mean Time to Resolution (days)"
    target: 5
    direction: decrease
workflows:
  - vulnerability-scanner
  - dependency-updater
  - security-reporter
```

### Quality campaign

```yaml
objective: "Increase test coverage to 90%"
kpis:
  - id: test_coverage
    name: "Test Coverage %"
    priority: primary
    target: 90
    direction: increase
workflows:
  - coverage-analyzer
  - test-generator
  - coverage-reporter
```

## Campaign lifecycle states

| State | Description | Orchestrator Behavior |
|-------|-------------|----------------------|
| `planned` | Design phase | Can compile but shouldn't execute |
| `active` | Running normally | Executes on schedule (daily) |
| `paused` | Temporarily suspended | Manual trigger only, not scheduled |
| `completed` | Objectives achieved | Should be disabled |
| `archived` | Historical reference | Workflow deleted/disabled |

```yaml
state: active  # Change to pause/complete campaign
```

> [!CAUTION]
> Changing state doesn't automatically disable the workflow. You must manually disable it in GitHub Actions UI.

## Troubleshooting quick tips

**Campaign not discovering items:**
- Check tracker-label matches what workflows apply
- Verify workflows are creating issues/PRs with correct labels
- Check governance limits aren't preventing discovery

**Orchestrator failing:**
- Check GitHub Actions logs for specific error
- Verify project-url is accessible with provided token
- Check allowed-repos includes all necessary repositories

**Items not appearing on project board:**
- Check max-project-updates-per-run limit
- Verify items have correct tracker label
- Check opt-out-labels aren't excluding items

**Too many API rate limit errors:**
- Reduce max-discovery-items-per-run
- Reduce max-discovery-pages-per-run
- Reduce max-project-updates-per-run

## Next steps

- **[Getting Started](/gh-aw/guides/campaigns/getting-started/)** – Create your first campaign
- **[Campaign Specs](/gh-aw/guides/campaigns/specs/)** – Complete spec reference
- **[Campaign Flow](/gh-aw/guides/campaigns/flow/)** – Understand the lifecycle
- **[Technical Overview](/gh-aw/guides/campaigns/technical-overview/)** – Architecture details
