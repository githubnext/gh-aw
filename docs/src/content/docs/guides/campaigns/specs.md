---
title: "Campaign Specs"
description: "Define and configure agentic campaigns with spec files, tracker labels, and recommended wiring"
---

Agentic campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. Each file has a YAML frontmatter block describing the agentic campaign.

## Agentic campaign spec files

```yaml
# .github/workflows/framework-upgrade.campaign.md
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

project-url: "https://github.com/orgs/ORG/projects/1"

workflows:
  - framework-upgrade

tracker-label: "campaign:framework-upgrade"
state: "active"
owners:
  - "platform-team"
```

Common fields you'll reach for as the initiative grows:

- `project-url`: the GitHub Project URL used as the primary campaign dashboard
- `tracker-label`: the label that ties issues/PRs back to the agentic campaign
- `memory-paths` / `metrics-glob`: where baselines and metrics snapshots live on your repo-memory branch
- `approval-policy`: the expectations for human approval (required approvals/roles)

Once you have a spec, the remaining question is consistency: what should every agentic campaign produce so people can follow along?

## Recommended default wiring

To keep agentic campaigns consistent and easy to read, most teams use a predictable set of primitives:

- **Tracker label** (for example, `campaign:<id>`) applied to every issue/PR in the agentic campaign.
- **Epic issue** (often also labeled `campaign-tracker`) as the human-readable command center.
- **GitHub Project** as the dashboard (primary campaign dashboard).
- **Repo-memory metrics** (daily JSON snapshots) to compute velocity/ETAs and enable trend reporting.
- **Tracker IDs in worker workflows** (e.g., `tracker-id: "worker-name"`) to enable orchestrator discovery of worker-created assets.
- **Monitor/orchestrator** to aggregate and post periodic updates.
- **Custom date fields** (optional, for roadmap views) like `Start Date` and `End Date` to visualize campaign timeline.

If you want to try this end-to-end quickly, start with the [Getting Started guide](/gh-aw/guides/campaigns/getting-started/).

## Spec validation and compilation

When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), `gh aw compile` will also generate an orchestrator workflow named `.github/workflows/<id>.campaign.g.md` and compile it to a corresponding `.lock.yml`.

See [Agentic campaign specs and orchestrators](/gh-aw/setup/cli/#compile) for details.
