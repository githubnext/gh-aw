---
title: "Campaigns"
description: "Run structured, visible automation initiatives with GitHub Agentic Workflows and GitHub Projects."
---

A campaign is a **finite initiative** with explicit ownership, review gates, and clear tracking. It helps you run large automation efforts—migrations, upgrades, and rollouts—in a way that is structured and visible.

Agentic workflows still do the hands-on work. Campaigns sit above them and add the “initiative layer”: a shared definition of scope, consistent tracking, and standard progress reporting.

If you are deciding whether you need a campaign, start here.

## When to use campaigns

Use a campaign when you need to run a finite initiative and you want it to be easy to review, operate, and report on.

Example: “Upgrade a dependency across 50 repositories over two weeks, with an approval gate, daily progress updates, and a final summary.”

| If you care about… | Use… |
|---|---|
| The result of each run (success/failure, logs, artifacts) | A regular workflow |
| The overall outcome across many runs, repos, and days/weeks | A campaign |

Why “just a label” stops being enough at scale: it does not define scope, it is easy to apply inconsistently, and it does not give you a standard status view.

Use a campaign when any of these are true:

- The work runs for days/weeks and needs handoffs and a durable status view.
- The scope spans many repos/teams and you need a single source of truth.
- You need approvals, staged rollouts, or other explicit decision points.
- You want repeatability: baselines + metrics + learnings for the next run.

What campaigns add:

- A campaign spec file declares the initiative (tracker label, referenced workflows, and optional memory/metrics locations).
- `gh aw compile` validates the spec and can generate an orchestrator workflow (`.campaign.g.md`).
- The CLI gives consistent inventory and status (`gh aw campaign`, `gh aw campaign status`).

You do not need campaigns just to run a workflow across many repositories (or org boundaries). That is primarily an authentication/permissions problem. Campaigns solve definition, validation, and consistent tracking.

## How campaigns work

Once you decide to use a campaign, most implementations follow the same shape:

- **Launcher workflow (required)**: finds work and creates tracking artifacts (issues/Project items), plus (optionally) a baseline in repo-memory.
- **Worker workflows (optional)**: process campaign-labeled issues to do the actual work (open PRs, apply fixes, etc.).
- **Monitor/orchestrator (recommended for multi-day work)**: posts periodic status updates and stores metrics snapshots.

You can track campaigns with just labels and issues, but campaigns become much more reusable when you also store baselines, metrics, and learnings in repo-memory (a git branch used for machine-generated snapshots).

Next: how gh-aw represents that “initiative layer” as a file you can review and version.

## Campaign spec files

In this repository, campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. Each file has a YAML frontmatter block describing the campaign.

```yaml
# .github/workflows/framework-upgrade.campaign.md
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

workflows:
  - framework-upgrade

tracker-label: "campaign:framework-upgrade"
state: "active"
owners:
  - "platform-team"
```

Common fields you’ll reach for as the initiative grows:

- `tracker-label`: the label that ties issues/PRs back to the campaign
- `memory-paths` / `metrics-glob`: where baselines and metrics snapshots live on your repo-memory branch
- `approval-policy`: the expectations for human approval (required approvals/roles)

Once you have a spec, the remaining question is consistency: what should every campaign produce so people can follow along?

## Recommended default wiring

To keep campaigns consistent and easy to read, most teams use a predictable set of primitives:

- **Tracker label** (for example, `campaign:<id>`) applied to every issue/PR in the campaign.
- **Epic issue** (often also labeled `campaign-tracker`) as the human-readable command center.
- **GitHub Project** as the dashboard (optional but common for cross-team work).
- **Repo-memory metrics** (daily JSON snapshots) to compute velocity/ETAs and enable trend reporting.
- **Monitor/orchestrator** to aggregate and post periodic updates.

If you want to try this end-to-end quickly, start with the minimal steps below.

## Quick start

1. Create a campaign spec: `.github/workflows/<id>.campaign.md`.
2. Reference one or more workflows in `workflows:`.
3. Add a `tracker-label` so issues/PRs can be queried consistently.
4. Run `gh aw compile` to validate campaign specs and compile workflows.

When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), `gh aw compile` will also generate an orchestrator workflow named `.github/workflows/<id>.campaign.g.md` and compile it to a corresponding `.lock.yml`.

See [Campaign specs and orchestrators](/gh-aw/setup/cli/#campaign-specs-and-orchestrators) for details.

## Try it with the CLI

From the root of the repo:

```bash
gh aw campaign
gh aw campaign status
gh aw campaign new my-campaign-id
gh aw campaign validate
```

For non-failing validation (useful in CI while you iterate):

```bash
gh aw campaign validate --strict=false
```

## Related Patterns

- **[ResearchPlanAssign](/gh-aw/guides/researchplanassign/)** - Research → generate coordinated work
- **[ProjectOps](/gh-aw/examples/issue-pr-events/projectops/)** - Project board integration for campaigns
- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Cross-repository operations
- **[Cache & Memory](/gh-aw/reference/memory/)** - Persistent storage for campaign data
- **[Safe Outputs](/gh-aw/reference/safe-outputs/)** - `create-issue`, `add-comment` for campaigns
