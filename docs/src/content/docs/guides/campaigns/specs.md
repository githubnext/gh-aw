---
title: Campaign specs
description: Campaign specification format and configuration reference
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

Campaign specs are YAML frontmatter configuration files at `.github/workflows/<id>.campaign.md`. The frontmatter defines campaign metadata, goals, workers, and governance. The body can contain narrative context.

## Spec structure

```yaml
---
id: framework-upgrade
name: "Framework Upgrade"
description: "Move services to Framework vNext"
state: active

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

discovery-repos:
  - "myorg/service-a"
  - "myorg/service-b"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    priority: primary
    unit: count
    baseline: 0
    target: 50
    time-window-days: 30
    direction: "increase"

workflows:
  - framework-upgrade-scanner

governance:
  max-project-updates-per-run: 10
  max-discovery-items-per-run: 50

owners:
  - "platform-team"
---

Additional narrative context about the campaign...
```

## Required fields

### Identity

**id** - Stable identifier for file naming and reporting
- Format: lowercase letters, digits, hyphens only
- Example: `security-audit-2025`

**name** - Human-friendly display name
- Example: `"Security Audit 2025"`

**project-url** - GitHub Project board URL for tracking
- Format: `https://github.com/orgs/ORG/projects/N`
- Example: `https://github.com/orgs/mycompany/projects/1`

### Discovery scope

At least one of these is required when using `workflows` or `tracker-label`:

**discovery-repos** - Specific repositories to search
- Format: List of `owner/repo` strings
- Example: `["myorg/api", "myorg/web"]`

**discovery-orgs** - Organizations to search (all repos)
- Format: List of organization names
- Example: `["myorg"]`

## Common fields

**objective** - One-sentence success definition
- Example: `"Eliminate all critical security vulnerabilities"`

**kpis** - Key performance indicators (1-3 maximum)
- See [KPI specification](#kpi-specification) below

**workflows** - Worker workflows to dispatch
- Format: List of workflow IDs (file names without extension)
- Example: `["security-scanner", "dependency-fixer"]`

**tracker-label** - Label for discovering worker outputs
- Format: `campaign:<id>`
- Example: `campaign:security-audit`

**state** - Lifecycle state
- Values: `planned`, `active`, `paused`, `completed`, `archived`
- Default: `planned`

**governance** - Pacing and safety limits
- See [Governance fields](#governance-fields) below

## Optional fields

**description** - Brief campaign description

**version** - Spec format version
- Default: `v1`

**owners** - Primary human owners
- Format: List of team or user names

**allowed-repos** - Repositories campaign can modify
- Default: Repository containing the spec
- Format: List of `owner/repo` strings

**allowed-orgs** - Organizations campaign can modify
- Format: List of organization names

**project-github-token** - Custom token for Projects API
- Format: Token expression like `${{ secrets.TOKEN_NAME }}`
- Use when default `GITHUB_TOKEN` lacks Projects permissions

## KPI specification

Each KPI requires these fields:

```yaml
kpis:
  - id: vulnerabilities_fixed          # Stable identifier
    name: "Vulnerabilities resolved"    # Display name
    priority: primary                    # One KPI must be primary
    unit: count                          # Measurement unit
    baseline: 50                         # Starting value
    target: 0                            # Goal value
    time-window-days: 30                 # Measurement period
    direction: "decrease"                # Improvement direction
```

### Required KPI fields

- **name** - Human-readable name
- **baseline** - Starting value
- **target** - Goal value
- **time-window-days** - Rolling window (7, 14, 30, or 90 days)

### Optional KPI fields

- **id** - Stable identifier (defaults to sanitized name)
- **priority** - `primary` or `supporting` (exactly one primary)
- **unit** - Measurement unit (`count`, `percent`, `days`, `hours`)
- **direction** - `increase` or `decrease`
- **source** - Signal source (`ci`, `pull_requests`, `code_security`, `custom`)

### KPI guidelines

- Define 1 primary KPI + up to 2 supporting KPIs (3 maximum)
- Always pair `objective` with `kpis` (define both or neither)
- Use concrete, measurable targets
- Choose realistic time windows

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

- Missing required fields (`id`, `name`, `project-url`)
- Missing discovery scope when using `workflows` or `tracker-label`
- Invalid KPI configuration (no primary, too many KPIs)
- Invalid `state` value
- Malformed URLs or identifiers

## Example: Security audit campaign

```yaml
---
id: security-audit-q1
version: "v1"
name: "Security Audit Q1 2025"
description: "Quarterly security review and remediation"
state: active

project-url: "https://github.com/orgs/myorg/projects/5"
tracker-label: "campaign:security-audit-q1"

discovery-orgs:
  - "myorg"

objective: "Resolve all high and critical security vulnerabilities"
kpis:
  - id: critical_vulns
    name: "Critical vulnerabilities"
    priority: primary
    unit: count
    baseline: 15
    target: 0
    time-window-days: 90
    direction: "decrease"
  - id: mttr
    name: "Mean time to resolution"
    priority: supporting
    unit: days
    baseline: 14
    target: 3
    time-window-days: 30
    direction: "decrease"

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

This campaign runs weekly to scan for vulnerabilities and track remediation.
Workers create issues with severity labels and automated fix PRs where possible.
```

## Further reading

- [Campaign lifecycle](/gh-aw/guides/campaigns/lifecycle/) - Execution model
- [Getting started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) - Validation and management
