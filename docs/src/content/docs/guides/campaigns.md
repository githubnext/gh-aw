---
title: "Agentic campaigns"
description: "Run structured, visible automation initiatives with GitHub Agentic Workflows and GitHub Projects."
---

An agentic campaign is a finite **initiative** with explicit ownership, review gates, and clear tracking. It helps you run large automation efforts—migrations, upgrades, and rollouts—in a way that is structured and visible.

Agentic workflows still do the hands-on work. Agentic campaigns sit above them and add the *initiative layer*: a shared definition of scope, consistent tracking, and standard progress reporting.

If you are deciding whether you need an agentic campaign, start here.

## When to use agentic campaigns

Use an agentic campaign when you need to run a finite initiative and you want it to be easy to review, operate, and report on.

Example: "Upgrade a dependency across 50 repositories over two weeks, with an approval gate, daily progress updates, and a final summary."

| If you care about… | Use… |
|---|---|
| The result of each run (success/failure, logs, artifacts) | A regular workflow |
| The overall outcome across many runs, repos, and days/weeks | An agentic campaign |

Why just-a-label stops being enough at scale: it does not define scope, it is easy to apply inconsistently, and it does not give you a standard status view.

Use an agentic campaign when any of these are true:

- The work runs for days/weeks and needs handoffs and a durable status view.
- The scope spans many repos/teams and you need a single source of truth.
- You need approvals, staged rollouts, or other explicit decision points.
- You want repeatability: baselines + metrics + learnings for the next run.

What agentic campaigns add:

- An agentic campaign spec file declares the initiative (Project dashboard URL, tracker label, referenced workflows, and optional memory/metrics locations).
- `gh aw compile` validates the spec and can generate an orchestrator workflow (`.campaign.g.md`).
- The CLI gives consistent inventory and status (`gh aw campaign`, `gh aw campaign status`).

You do not need agentic campaigns just to run a workflow across many repositories (or org boundaries). That is primarily an authentication/permissions problem. Agentic campaigns solve definition, validation, and consistent tracking.

## How agentic campaigns work

Once you decide to use an agentic campaign, most implementations follow the same shape:

- **Orchestrator workflow (generated)**: maintains the campaign dashboard by syncing tracker-labeled issues/PRs to the GitHub Project board, updating status fields, and posting periodic reports. The orchestrator handles both initial discovery and ongoing synchronization.
- **Worker workflows (optional)**: process campaign-labeled issues to do the actual work (open PRs, apply fixes, etc.). Workers include a `tracker-id` so the orchestrator can discover their created assets.

You can track agentic campaigns with just labels and issues, but agentic campaigns become much more reusable when you also store baselines, metrics, and learnings in repo-memory (a git branch used for machine-generated snapshots).

### Orchestrator and Worker Coordination

Agentic campaigns use a **tracker-id** mechanism to coordinate between orchestrators and workers. This architecture maintains clean separation of concerns: workers execute tasks without campaign awareness, while orchestrators manage coordination and tracking.

#### The Coordination Pattern

1. **Worker workflows** include a `tracker-id` in their frontmatter (e.g., `tracker-id: "daily-file-diet"`). This identifier is automatically embedded in all assets created by the workflow (issues, PRs, discussions, comments) as an XML comment marker: `<!-- agentic-workflow: WorkflowName, tracker-id: daily-file-diet, ... -->`

2. **Orchestrator workflows** discover work created by workers by searching for issues containing the worker's tracker-id. For example, to find issues created by a worker with `tracker-id: "daily-file-diet"`:
   ```
   repo:owner/repo "tracker-id: daily-file-diet" in:body
   ```

3. The orchestrator then adds discovered issues to the agentic campaign's GitHub Project board and updates their status as work progresses.

This design allows workers to operate independently without knowledge of the agentic campaign, while orchestrators maintain a centralized view of all campaign work by searching for tracker-id markers.

#### Orchestrator Workflow Phases

Generated orchestrator workflows follow a four-phase execution model each time they run:

**Phase 1: Read State (Discovery)**
- Query for tracker-labeled issues/PRs matching the campaign
- Query for worker-created issues using tracker-id search (if workers are configured)
- Read current state of the GitHub Project board
- Compare discovered items against board state to identify gaps

**Phase 2: Make Decisions (Planning)**
- Decide which new items to add to the board (respecting governance limits)
- Determine status updates for existing items (respecting governance rules like no-downgrade)
- Check campaign completion criteria

**Phase 3: Write State (Execution)**
- Add new items to project board via `update-project` safe output
- Update status fields for existing board items
- Record completion state if campaign is done

**Phase 4: Report (Output)**
- Generate status report summarizing execution
- Record metrics: items discovered, added, updated, skipped
- Report any failures encountered

#### Core Design Principles

The orchestrator/worker pattern enforces these principles:

- **Workers are immutable** - Worker workflows never change based on campaign state
- **Workers are campaign-agnostic** - Workers execute the same way regardless of campaign context
- **Campaign logic is external** - All orchestration happens in the orchestrator, not workers
- **Single source of truth** - The GitHub Project board is the authoritative campaign state
- **Idempotent operations** - Re-execution produces the same result without corruption
- **Governed operations** - Orchestrators respect pacing limits and opt-out policies

These principles ensure workers can be reused across agentic campaigns and remain simple, while orchestrators handle all coordination complexity.

## Next Steps

- **[Campaign Specs](/gh-aw/guides/campaigns/specs/)** - Learn about spec files and configuration
- **[Getting Started](/gh-aw/guides/campaigns/getting-started/)** - Quick start guide and walkthrough
- **[Project Management](/gh-aw/guides/campaigns/project-management/)** - Using GitHub Projects with roadmap views
- **[CLI Commands](/gh-aw/guides/campaigns/cli-commands/)** - Command reference for campaign management

## Related Patterns

- **[ResearchPlanAssign](/gh-aw/guides/researchplanassign/)** - Research → generate coordinated work
- **[ProjectOps](/gh-aw/examples/issue-pr-events/projectops/)** - Project board integration for campaigns
- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Cross-repository operations
- **[Cache & Memory](/gh-aw/reference/memory/)** - Persistent storage for campaign data
- **[Safe Outputs](/gh-aw/reference/safe-outputs/)** - `create-issue`, `add-comment` for campaigns
