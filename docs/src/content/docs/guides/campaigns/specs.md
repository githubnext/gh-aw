---
title: Campaign specs
description: Campaign specification format and configuration reference
banner:
  content: '<strong>⚠️ Deprecated:</strong> The <code>.campaign.md</code> file format is deprecated. Use the <code>project</code> field in workflow frontmatter instead. See <a href="/gh-aw/reference/frontmatter/#project-tracking-project">Project Tracking</a> for the current approach.'
---

:::caution[File format deprecated]
The `.campaign.md` standalone file format described in this document is **deprecated** and has been removed from gh-aw. 

**Migration:** Use the `project` field in workflow frontmatter instead:
```yaml
---
on:
  schedule:
    - cron: "0 0 * * *"
project:
  url: https://github.com/orgs/myorg/projects/1
  workflows:
    - worker-workflow-name
---
```
See [Project Tracking documentation](/gh-aw/reference/frontmatter/#project-tracking-project) for details.
:::

Campaign specs were YAML frontmatter configuration files at `.github/workflows/<id>.campaign.md`. The frontmatter defined pure configuration (id, project-url, workflows, governance, etc.), while the markdown body contained narrative context including objectives, KPIs, timelines, and strategy.

## Spec structure (deprecated)

:::note[Historical reference]
This section documents the deprecated `.campaign.md` file format for historical reference only.
:::

A minimal campaign spec was a `.github/workflows/<id>.campaign.md` file with YAML frontmatter plus a markdown body. Most fields had sensible defaults.

```markdown
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
- If omitted, defaults to the filename basename (e.g. `security-audit.campaign.md` → `security-audit`)

**name** - Human-friendly display name
- Example: `"Security Audit 2025"`
- Default: Uses `id` if not specified

**project-url** - GitHub Project board URL for tracking
- Format: `https://github.com/orgs/ORG/projects/N`
- Example: `https://github.com/orgs/mycompany/projects/1`

**workflows** - Worker workflows that implement the campaign
- Format: List of workflow IDs (file names without .md extension)
- Example: `["security-scanner", "dependency-fixer"]`

`workflows` is strongly recommended for most campaigns (and `gh aw campaign validate` will flag empty workflows). It can be omitted for campaigns that only do coordination/discovery work.

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

**scope** - Repositories and organizations this campaign can operate on
- Default: Current repository (where campaign is defined)

Campaign scope is defined once and used for both discovery and execution:

**scope** - Scope selectors
- Repository selector: `owner/repo`
- Organization selector: `org:<name>`
- Example: `["myorg/api", "myorg/web", "org:myorg"]`

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

No other scope fields are needed; use `scope`.

Campaign orchestrators are **dispatch-only by design**:
- The orchestrator can make decisions and coordinate work.
- The orchestrator may only *act* by dispatching allowlisted worker workflows via `safe-outputs.dispatch-workflow`.
- All side effects (Projects, issues/PRs, comments) happen in worker workflows with their own safe-outputs.

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
Campaign discovery uses the same `scope` as execution.

```yaml
scope:
  - "myorg/frontend"
  - "myorg/backend"
  - "myorg/api"
  - "org:myorg"        # optional org-wide scope
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

```markdown
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
- `scope`: current repository

## Full example

With governance and org scope:

```yaml
---
id: security-audit-q1
name: "Security Audit Q1 2025"
description: "Quarterly security review and remediation"
project-url: "https://github.com/orgs/myorg/projects/5"

scope:
  - "org:myorg"

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
