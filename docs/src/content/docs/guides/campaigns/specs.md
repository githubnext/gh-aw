---
title: Campaign specs
description: Campaign specification format and configuration reference
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

Campaign specs are YAML frontmatter configuration files at `.github/workflows/<id>.campaign.md`. The frontmatter defines pure configuration (id, project-url, workflows, governance, etc.), while the markdown body contains narrative context including objectives, KPIs, timelines, and strategy.

## Spec structure

A minimal campaign spec requires only `id`, `project-url`, and `workflows`. Most fields have sensible defaults.

```yaml
---
id: framework-upgrade
name: "Framework Upgrade"
project-url: "https://github.com/orgs/ORG/projects/1"

workflows:
  - framework-upgrade-scanner

governance:
  max-project-updates-per-run: 10
---

# Framework Upgrade Campaign

## Objective

Upgrade all services to Framework vNext with zero downtime.

## Key Performance Indicators (KPIs)

### Primary KPI: Services Upgraded
- **Baseline**: 0 services
- **Target**: 50 services
- **Time Window**: 30 days
- **Direction**: Increase

## Timeline

- **Phase 1** (Weeks 1-2): Discovery and planning
- **Phase 2** (Weeks 3-6): Incremental upgrades
- **Phase 3** (Week 7+): Validation and monitoring
```

## Required fields

### Identity

**id** - Stable identifier for file naming and reporting
- Format: lowercase letters, digits, hyphens only
- Example: `security-audit-2025`
- Auto-generates defaults for: tracker-label, memory-paths, metrics-glob, cursor-glob

**name** - Human-friendly display name
- Example: `"Security Audit 2025"`
- Default: Uses `id` if not specified

**project-url** - GitHub Project board URL for tracking
- Format: `https://github.com/orgs/ORG/projects/N`
- Example: `https://github.com/orgs/mycompany/projects/1`

**workflows** - Worker workflows that implement the campaign
- Format: List of workflow IDs (file names without .md extension)
- Example: `["security-scanner", "dependency-fixer"]`

## Fields with defaults

Many fields have automatic defaults based on the campaign ID:

**state** - Lifecycle stage
- Default: `active`
- Values: `planned`, `active`, `paused`, `completed`, `archived`

**tracker-label** - Label for discovering worker outputs
- Default: `z_campaign_{id}` (e.g., `z_campaign_security-audit`)
- Can be customized if needed

**memory-paths** - Where campaign writes repo-memory
- Default: `["memory/campaigns/{id}/**"]`

**metrics-glob** - Glob for JSON metrics snapshots
- Default: `memory/campaigns/{id}/metrics/*.json`

**cursor-glob** - Glob for durable cursor/checkpoint file
- Default: `memory/campaigns/{id}/cursor.json`

**allowed-repos** - Repositories campaign can operate on
- Default: Current repository (where campaign is defined)

**discovery-repos** - Repositories to search for worker outputs
- Default: Same as `allowed-repos`

### Discovery scope (optional)

Override discovery scope when operating across multiple repositories:

**discovery-repos** - Specific repositories to search
- Format: List of `owner/repo` strings
- Example: `["myorg/api", "myorg/web"]`
- Default: Same as `allowed-repos` (current repository)

**discovery-orgs** - Organizations to search (all repos)
- Format: List of organization names
- Example: `["myorg"]`
- Overrides `discovery-repos` when specified

## Optional fields

**description** - Brief campaign description
- Provides context in listings and dashboards

**version** - Spec format version
- Default: `v1`
- Usually not needed in specs

**owners** - Primary human owners
- Format: List of team or user names
- Example: `["@security-team", "alice"]`

**governance** - Pacing and safety limits
- See [Governance fields](#governance-fields) below

**allowed-orgs** - Organizations campaign can modify
- Format: List of organization names
- Alternative to specifying individual `allowed-repos`

**project-github-token** - Custom token for Projects API
- Format: Token expression like `${{ secrets.TOKEN_NAME }}`
- Use when default `GITHUB_TOKEN` lacks Projects permissions

## Markdown body content

The markdown body contains narrative context and goals. Include:

**Objective** - Clear statement of what the campaign aims to achieve
- Example: "Reduce all critical security vulnerabilities to zero"
- Can include multiple paragraphs with context and rationale

**Key Performance Indicators (KPIs)** - Measurable success metrics
- Define 1 primary KPI + up to 2 supporting KPIs
- Include baseline, target, time window, direction for each
- Example format:
  ```markdown
  ### Primary KPI: Critical Vulnerabilities
  - **Baseline**: 15 issues
  - **Target**: 0 issues
  - **Time Window**: 90 days
  - **Direction**: Decrease
  ```

**Timeline** - Campaign phases and milestones
**Worker Workflows** - Descriptions of automated workflows
**Success Criteria** - Concrete conditions for completion
**Risk Management** - Mitigation strategies and approvals

## Governance fields

Governance controls execution pace and safety:

```yaml
governance:
  max-project-updates-per-run: 10
  max-discovery-items-per-run: 50
  max-discovery-pages-per-run: 5
  max-new-items-per-run: 10
  max-comments-per-run: 10
  do-not-downgrade-done-items: true
  opt-out-labels: ["campaign:skip", "no-bot"]
```

**max-project-updates-per-run** - Maximum project board updates per execution
- Default: Conservative limit
- Start low (10) and increase with confidence

**max-discovery-items-per-run** - Maximum items to discover per execution
- Controls API load
- Remaining items discovered on next run

**max-discovery-pages-per-run** - Maximum API pages to fetch
- Alternative to item limit

**max-new-items-per-run** - Maximum new items to add to project
- Separate from total updates

**max-comments-per-run** - Maximum comments to post
- Prevents notification spam

**do-not-downgrade-done-items** - Prevent moving completed items backward
- Recommended: `true`

**opt-out-labels** - Labels that exclude items from campaign
- Default: `["no-bot", "no-campaign"]`

## Discovery configuration

### Repository-scoped discovery

```yaml
discovery-repos:
  - "myorg/frontend"
  - "myorg/backend"
  - "myorg/api"
```

Searches only specified repositories for issues and pull requests with tracker labels.

### Organization-scoped discovery

```yaml
discovery-orgs:
  - "myorg"
```

Searches all repositories in the organization. Use carefully - can be expensive for large organizations.

### Hybrid approach

```yaml
discovery-repos:
  - "myorg/critical-service"  # Always scan this one
discovery-orgs:
  - "myorg"                    # Scan all others
```

## Validation

Validate campaign specs before committing:

```bash
gh aw campaign validate
```

Common validation errors:

- Missing required fields (`id`, `project-url`, `workflows`)
- Invalid `state` value
- Malformed URLs or identifiers

## Minimal example

The simplest possible campaign:

```yaml
---
id: security-audit-q1
name: "Security Audit Q1 2025"
project-url: "https://github.com/orgs/myorg/projects/5"

workflows:
  - security-scanner
  - dependency-updater
---

# Security Audit Q1 2025 Campaign

Document your objectives, KPIs, timeline, and strategy here...
```

This automatically gets:
- `state: active`
- `tracker-label: z_campaign_security-audit-q1`
- `memory-paths: ["memory/campaigns/security-audit-q1/**"]`
- `metrics-glob: memory/campaigns/security-audit-q1/metrics/*.json`
- `cursor-glob: memory/campaigns/security-audit-q1/cursor.json`
- `allowed-repos` and `discovery-repos`: current repository

## Full example

With governance and multi-org scope:

```yaml
---
id: security-audit-q1
name: "Security Audit Q1 2025"
description: "Quarterly security review and remediation"
project-url: "https://github.com/orgs/myorg/projects/5"

discovery-orgs:
  - "myorg"

workflows:
  - security-scanner
  - dependency-updater

governance:
  max-project-updates-per-run: 20
  max-discovery-items-per-run: 100
  do-not-downgrade-done-items: true

owners:
  - "security-team"
---

# Security Audit Q1 2025 Campaign

## Objective

Resolve all high and critical security vulnerabilities across the organization.

## Key Performance Indicators (KPIs)

### Primary KPI: Critical Vulnerabilities
- **Baseline**: 15 vulnerabilities
- **Target**: 0 vulnerabilities
- **Time Window**: 90 days
- **Direction**: Decrease

### Supporting KPI: Mean Time to Resolution
- **Baseline**: 14 days
- **Target**: 3 days
- **Time Window**: 30 days
- **Direction**: Decrease

## Timeline

This campaign runs weekly to scan for vulnerabilities and track remediation. Workers create issues with severity labels and automated fix PRs where possible.
```

## Further reading

- [Campaign lifecycle](/gh-aw/guides/campaigns/lifecycle/) - Execution model
- [Getting started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) - Validation and management
