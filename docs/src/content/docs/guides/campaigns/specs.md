---
title: Campaign specs
description: Define and configure agentic campaigns with spec files, tracker labels, and recommended wiring
---

Campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. The YAML frontmatter is the campaign “contract”; the body can contain optional narrative context.

## What a campaign is (in gh-aw)

In GitHub Agentic Workflows, a campaign is not "a special kind of workflow." The `.campaign.md` file is a specification: a reviewable contract that wires together agentic workflows around a shared initiative.

In a typical setup:

- Worker workflows do the work. They run an agent and use safe-outputs (for example `create_pull_request`, `add_comment`, or `update_issues`) for write operations.
- A generated orchestrator workflow keeps the campaign coherent over time. It discovers and tracks work, executes workflows, and drives progress.
- Repo-memory (optional) makes the campaign repeatable. It lets you store a cursor checkpoint and append-only metrics snapshots so each run can pick up where the last one left off.
- GitHub Project dashboard serves as the canonical source of membership and progress tracking.

**Note:** The `.campaign.g.md` file is a local debug artifact generated during compilation to help developers review the orchestrator structure. It is not committed to git (it's in `.gitignore`). Only the source `.campaign.md` and the compiled `.campaign.lock.yml` are version controlled.

> [!TIP]
> Worker workflows remain campaign-agnostic and don't need modification. The orchestrator handles all campaign coordination.

## Minimal spec

```yaml
# .github/workflows/framework-upgrade.campaign.md
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

# Optional: Custom GitHub token for Projects v2 operations
# project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"

# Required: Repositories this campaign can operate on
allowed-repos:
  - "myorg/service-a"
  - "myorg/service-b"

# Optional: Organizations this campaign can operate on
# allowed-orgs:
#   - "myorg"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    priority: primary
    direction: "increase"
    target: 50
  - id: incidents
    name: "Incidents caused"
    direction: "decrease"
    target: 0

workflows:
  - framework-upgrade

state: "active"
owners:
  - "platform-team"
```

## Core fields (what they do)

- `id`: stable identifier used for file naming, reporting, and (if used) repo-memory paths.
- `allowed-repos` (required): list of repositories (in `owner/repo` format) that this campaign is allowed to discover and operate on. Defines the campaign scope as a reviewable contract for security and governance. Must include at least one repository.
- `allowed-orgs` (optional): list of GitHub organizations that this campaign is allowed to discover and operate on. Provides additional scope control when operating across multiple repositories in an organization.
- `project-url` (optional): the GitHub Project that acts as the campaign dashboard and canonical source of campaign membership. If not provided, the campaign generator will automatically create a new project board with custom fields and views.
- `project-github-token` (optional): a GitHub token expression (e.g., `${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}`) used for GitHub Projects v2 operations. When specified, this token is passed to the `update-project` safe output configuration in the generated orchestrator workflow. Use this when the default `GITHUB_TOKEN` doesn't have sufficient permissions for project board operations.
- `tracker-label` (optional): an ingestion hint label that helps discover issues and pull requests created by workers (commonly `campaign:<id>`). When provided, the orchestrator's discovery precomputation step can discover work across runs. The project board remains the canonical source of truth.
- `objective`: a single sentence describing what “done” means.
- `kpis`: the measures you use to report progress. Use `priority: primary` to mark exactly one KPI as the primary measure (not `primary: true`).
- `workflows`: the participating workflow IDs. These refer to workflows in the repo (commonly `.github/workflows/<workflow-id>.md`). When workflows are configured, the orchestrator will execute them sequentially and can create missing workflows if needed.

> [!IMPORTANT]
> Use `priority: primary` (not `primary: true`) to mark your primary KPI.

## KPIs (recommended shape)

Keep KPIs small and crisp:

- Use 1 primary KPI + a few supporting KPIs.
- Use `direction: increase|decrease|maintain` to describe the desired trend.
- Use `target` when there is a clear threshold.

If you define `kpis`, also define `objective` (and vice versa). It keeps the spec reviewable and makes reports consistent.

## Durable state (repo-memory)

If you use repo-memory for campaigns, standardize on one layout so runs are comparable:

- `memory/campaigns/<campaign-id>/cursor.json`
- `memory/campaigns/<campaign-id>/metrics/<date>.json`

Typical wiring in the spec:

```yaml
memory-paths:
  - "memory/campaigns/framework-upgrade/cursor.json"
metrics-glob: "memory/campaigns/framework-upgrade/metrics/*.json"
```

Campaign tooling enforces the durability contract at push time: a campaign repo-memory write must include a cursor and at least one metrics snapshot.

## Governance (pacing & safety)

Use `governance` to keep orchestration predictable and reviewable:

```yaml
governance:
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  opt-out-labels: ["campaign:skip"]
  do-not-downgrade-done-items: true
  max-project-updates-per-run: 50
  max-comments-per-run: 10
```

> [!TIP]
> Start conservative with low limits (e.g., `max-project-updates-per-run: 10`) for your first campaign, then increase as you gain confidence.

### Governance fields

- `max-new-items-per-run`: Maximum number of new items to add to the project board per run (applies to agent write phase)
- `max-discovery-items-per-run`: Maximum number of candidate items the discovery precomputation step will scan per run (default: 100)
- `max-discovery-pages-per-run`: Maximum number of API result pages the discovery step will fetch per run (default: 10)
- `opt-out-labels`: Labels that exclude an item from campaign tracking. Common values include `["campaign:skip", "no-bot", "no-campaign"]`. Items with these labels will not be discovered by campaign orchestrators, and other workflows (like issue-monster) will also respect these labels.
- `do-not-downgrade-done-items`: Prevent moving items backwards from "Done" status
- `max-project-updates-per-run`: Maximum number of project board updates per run (default: 10)
- `max-comments-per-run`: Maximum number of comments the orchestrator can post per run (default: 10)

These governance policies are enforced during the discovery precomputation step (for read budgets) and during the agent coordination phase (for write budgets), ensuring sustainable API usage and manageable workload.

## Compilation and orchestrators

`gh aw compile` validates campaign specs. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), it also generates an orchestrator and compiles it to `.campaign.lock.yml`.

During compilation, a `.campaign.g.md` file is generated locally as a debug artifact to help developers understand the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

See [Agentic campaign specs and orchestrators](/gh-aw/setup/cli/#compile).
