---
title: "Campaigns"
description: "Run structured, visible automation initiatives with GitHub Agentic Workflows and GitHub Projects."
---

A campaign is a finite **initiative** with explicit ownership, review gates, and clear tracking. It helps you run large automation efforts—migrations, upgrades, and rollouts—in a way that is structured and visible.

Agentic workflows still do the hands-on work. Campaigns sit above them and add the *initiative layer*: a shared definition of scope, consistent tracking, and standard progress reporting.

If you are deciding whether you need a campaign, start here.

## When to use campaigns

Use a campaign when you need to run a finite initiative and you want it to be easy to review, operate, and report on.

Example: “Upgrade a dependency across 50 repositories over two weeks, with an approval gate, daily progress updates, and a final summary.”

| If you care about… | Use… |
|---|---|
| The result of each run (success/failure, logs, artifacts) | A regular workflow |
| The overall outcome across many runs, repos, and days/weeks | A campaign |

Why just-a-label stops being enough at scale: it does not define scope, it is easy to apply inconsistently, and it does not give you a standard status view.

Use a campaign when any of these are true:

- The work runs for days/weeks and needs handoffs and a durable status view.
- The scope spans many repos/teams and you need a single source of truth.
- You need approvals, staged rollouts, or other explicit decision points.
- You want repeatability: baselines + metrics + learnings for the next run.

What campaigns add:

- A campaign spec file declares the initiative (Project dashboard URL, tracker label, referenced workflows, and optional memory/metrics locations).
- `gh aw compile` validates the spec and can generate an orchestrator workflow (`.campaign.g.md`).
- The CLI gives consistent inventory and status (`gh aw campaign`, `gh aw campaign status`).

You do not need campaigns just to run a workflow across many repositories (or org boundaries). That is primarily an authentication/permissions problem. Campaigns solve definition, validation, and consistent tracking.

## How campaigns work

Once you decide to use a campaign, most implementations follow the same shape:

- **Launcher workflow (required)**: finds work and creates tracking artifacts (issues/Project items), plus (optionally) a baseline in repo-memory.
- **Worker workflows (optional)**: process campaign-labeled issues to do the actual work (open PRs, apply fixes, etc.).
- **Monitor/orchestrator (recommended for multi-day work)**: posts periodic status updates and stores metrics snapshots.

You can track campaigns with just labels and issues, but campaigns become much more reusable when you also store baselines, metrics, and learnings in repo-memory (a git branch used for machine-generated snapshots).

### Orchestrator and Worker Coordination

Campaigns use a **tracker-id** mechanism to coordinate between orchestrators and workers. This architecture maintains clean separation of concerns: workers execute tasks without campaign awareness, while orchestrators manage coordination and tracking.

#### The Coordination Pattern

1. **Worker workflows** include a `tracker-id` in their frontmatter (e.g., `tracker-id: "daily-file-diet"`). This identifier is automatically embedded in all assets created by the workflow (issues, PRs, discussions, comments) as an XML comment marker: `<!-- agentic-workflow: WorkflowName, tracker-id: daily-file-diet, ... -->`

2. **Orchestrator workflows** discover work created by workers by searching for issues containing the worker's tracker-id. For example, to find issues created by a worker with `tracker-id: "daily-file-diet"`:
   ```
   repo:owner/repo "tracker-id: daily-file-diet" in:body
   ```

3. The orchestrator then adds discovered issues to the campaign's GitHub Project board and updates their status as work progresses.

This design allows workers to operate independently without knowledge of the campaign, while orchestrators maintain a centralized view of all campaign work by searching for tracker-id markers.

#### Orchestrator Workflow Phases

Generated orchestrator workflows follow a four-phase execution model each time they run:

**Phase 1: Read State (Discovery)**
- Query for worker-created issues using tracker-id search
- Read current state of the GitHub Project board
- Compare discovered issues against board state to identify gaps

**Phase 2: Make Decisions (Planning)**
- Decide which new issues to add to the board
- Determine status updates for existing items
- Check campaign completion criteria

**Phase 3: Write State (Execution)**
- Add new issues to project board via `update-project` safe output
- Update status fields for existing board items
- Record completion state if campaign is done

**Phase 4: Report (Output)**
- Generate status report summarizing execution
- Record metrics: issues discovered, added, updated
- Report any failures encountered

#### Core Design Principles

The orchestrator/worker pattern enforces these principles:

- **Workers are immutable** - Worker workflows never change based on campaign state
- **Workers are campaign-agnostic** - Workers execute the same way regardless of campaign context
- **Campaign logic is external** - All orchestration happens in the orchestrator, not workers
- **Single source of truth** - The GitHub Project board is the authoritative campaign state
- **Idempotent operations** - Re-execution produces the same result without corruption

These principles ensure workers can be reused across campaigns and remain simple, while orchestrators handle all coordination complexity.


Next: how gh-aw represents that “initiative layer” as a file you can review and version.

## Campaign spec files

In this repository, campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. Each file has a YAML frontmatter block describing the campaign.

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

Common fields you’ll reach for as the initiative grows:

- `project-url`: the GitHub Project URL used as the primary campaign dashboard
- `tracker-label`: the label that ties issues/PRs back to the campaign
- `memory-paths` / `metrics-glob`: where baselines and metrics snapshots live on your repo-memory branch
- `approval-policy`: the expectations for human approval (required approvals/roles)

Once you have a spec, the remaining question is consistency: what should every campaign produce so people can follow along?

## Recommended default wiring

To keep campaigns consistent and easy to read, most teams use a predictable set of primitives:

- **Tracker label** (for example, `campaign:<id>`) applied to every issue/PR in the campaign.
- **Epic issue** (often also labeled `campaign-tracker`) as the human-readable command center.
- **GitHub Project** as the dashboard (primary campaign dashboard).
- **Repo-memory metrics** (daily JSON snapshots) to compute velocity/ETAs and enable trend reporting.
- **Tracker IDs in worker workflows** (e.g., `tracker-id: "worker-name"`) to enable orchestrator discovery of worker-created assets.
- **Monitor/orchestrator** to aggregate and post periodic updates.
- **Custom date fields** (optional, for roadmap views) like `Start Date` and `End Date` to visualize campaign timeline.

If you want to try this end-to-end quickly, start with the minimal steps below.

## Quick start

1. Create a campaign spec: `.github/workflows/<id>.campaign.md`.
2. Reference one or more workflows in `workflows:`.
3. Set `project-url` to the org Project v2 URL you use as the campaign dashboard.
4. Add a `tracker-label` so issues/PRs can be queried consistently.
5. Run `gh aw compile` to validate campaign specs and compile workflows.

## Lowest-friction walkthrough (recommended)

The simplest, least-permissions way to run a campaign is:

1. **Create the campaign spec (in a PR)**
  - **Option A (No-code)**: Use the "🚀 Start a Campaign" issue form in the GitHub UI to capture intent with structured fields. The form creates a campaign issue, and an agent can scaffold the spec file for you.
  - **Option B (CLI)**: Use `gh aw campaign new <id>` to generate a campaign spec file locally.
  - **Option C (Manual)**: Author `.github/workflows/<id>.campaign.md` manually.

2. **Create the org Project board once (manual)**
  - Create an org Project v2 in the GitHub UI and copy its URL into `project-url`.
  - This avoids requiring a PAT or GitHub App setup just to provision boards.
  - Minimum clicks (one-time setup):
    - In GitHub: your org 0 **Projects** 0 **New project**.
    - Give it a name (for example: `Code Health: <Campaign Name>`).
    - Choose any starting layout (Table/Board). You can change views later.
    - Copy the Project URL and set it as `project-url` in the campaign spec.
  - Optional but recommended for “kanban lanes”:
    - Create a **Board** view and set **Group by** to a single-select field (commonly `Status`).
    - Note: workflows can create/update fields and single-select options, but they do not currently create or configure Project views.

3. **Have workflows keep the board in sync using `GITHUB_TOKEN`**
  - Enable the `update-project` safe output in the launcher/monitor workflows.
  - Default behavior is **update-only**: if the board does not exist, the project job fails with instructions.

4. **Opt in to auto-creating the board only when you intend to**
  - If you want workflows to create missing boards, explicitly set `create_if_missing: true` in the `update_project` output.
  - For many orgs, you may also need a token override (`safe-outputs.update-project.github-token`) with sufficient org Project permissions.

When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), `gh aw compile` will also generate an orchestrator workflow named `.github/workflows/<id>.campaign.g.md` and compile it to a corresponding `.lock.yml`.

See [Campaign specs and orchestrators](/gh-aw/setup/cli/#campaign-specs-and-orchestrators) for details.

## Using Project Roadmap Views with Custom Date Fields

GitHub Projects offers a [Roadmap view](https://docs.github.com/en/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/customizing-the-roadmap-layout) that visualizes work items along a timeline. To use this view with campaigns, you need to add custom date fields to track when work items start and end.

### Setting Up Custom Date Fields

**One-time manual setup** (in the GitHub Projects UI):

1. Open your campaign's Project board
2. Click the **+** button in the header row to add a new field
3. Create a **Date** field named `Start Date`
4. Create another **Date** field named `End Date`
5. Create a **Roadmap** view from the view dropdown
6. Configure the roadmap to use your date fields

Once these fields exist, orchestrator workflows can automatically populate them when adding or updating project items.

### Orchestrator Configuration for Date Fields

To have orchestrators set date fields automatically, modify the orchestrator's instructions or use the `fields` parameter in `update-project` outputs.

**Example workflow instruction:**

```markdown
When adding issues to the project board, set these custom fields:
- `Start Date`: Set to the issue's creation date
- `End Date`: Set to estimated completion date based on issue size and priority
  - Small issues: 3 days from start
  - Medium issues: 1 week from start
  - Large issues: 2 weeks from start
```

**Example agent output for update-project:**

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/myrepo/issues/123"
  fields:
    status: "In Progress"
    priority: "High"
    start_date: "2025-12-19"
    end_date: "2025-12-26"
```

### Best Practices for Campaign Date Fields

**Recommended field names:**
- `Start Date` or `start_date` - When work begins
- `End Date` or `end_date` - Expected or actual completion date
- `Target Date` - Optional milestone or deadline

**Date assignment strategies:**

- **For new issues**: Set `start_date` to current date, calculate `end_date` based on estimated effort
- **For in-progress work**: Keep original `start_date`, adjust `end_date` if needed
- **For completed work**: Update `end_date` to actual completion date

**Roadmap view benefits:**

- **Visual timeline**: See all campaign work laid out chronologically
- **Dependency identification**: Spot overlapping or sequential work items
- **Capacity planning**: Identify periods with too much concurrent work
- **Progress tracking**: Compare planned vs actual completion dates

### Example: Campaign with Roadmap Tracking

```yaml
# .github/workflows/migration-q1.campaign.md
id: migration-q1
name: "Q1 Migration Campaign"
project-url: "https://github.com/orgs/myorg/projects/15"
workflows:
  - migration-worker
tracker-label: "campaign:migration-q1"
```

The orchestrator can set date fields when adding issues:

```markdown
## Campaign Orchestrator

When adding discovered issues to the project board:

1. Query issues with tracker-id: "migration-worker"
2. For each issue:
   - Add to project board
   - Set `status` to "Todo" (or "Done" if closed)
   - Set `start_date` to the issue creation date
   - Set `end_date` based on labels:
     - `size:small` → 3 days from start
     - `size:medium` → 1 week from start  
     - `size:large` → 2 weeks from start
   - Set `priority` based on issue labels

Generate a report showing timeline distribution of all work items.
```

### Limitations and Considerations

- **Manual field creation**: Workflows cannot create custom fields; they must exist before workflows can update them
- **Field name matching**: Custom field names are case-sensitive; use exact names as defined in the project
- **Date format**: Use ISO 8601 format (YYYY-MM-DD) for date values
- **No automatic recalculation**: Date fields don't auto-update; orchestrators must explicitly update them
- **View configuration**: Roadmap views must be configured manually in the GitHub UI

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
gh aw campaign validate --no-strict
```

## Start a Campaign with GitHub Issue Forms

For a low-code/no-code approach, you can create a campaign using the GitHub UI with the "🚀 Start a Campaign" issue form:

1. **Go to the repository's Issues tab** and click "New issue"
2. **Select "🚀 Start a Campaign"** from the available templates
3. **Fill in the structured form fields**:
   - **Campaign Name** (required): Human-readable name (e.g., "Framework Upgrade Q1 2025")
   - **Campaign Identifier** (required): Unique ID using lowercase letters, digits, and hyphens (e.g., "framework-upgrade-q1-2025")
   - **Campaign Version** (required): Version string (defaults to "v1")
   - **Project Board URL** (required): URL of the GitHub Project you created to serve as the campaign dashboard
   - **Campaign Type** (optional): Select from Migration, Upgrade/Modernization, Security Remediation, etc.
   - **Scope** (optional): Define what repositories, components, or areas will be affected
   - **Constraints** (optional): List any constraints or requirements (deadlines, approvals, etc.)
   - **Prior Learnings** (optional): Share relevant learnings from past similar campaigns
4. **Submit the form** to create the campaign issue

### What happens after submission

When you submit the issue form:

1. **A campaign issue is created** - This becomes your campaign's central hub with the `campaign` and `campaign-tracker` labels
2. **An agent validates your project board** - Ensures the URL is accessible and properly configured
3. **A campaign spec is generated** - Creates `.github/workflows/<id>.campaign.md` with your inputs as a PR
4. **The spec is linked to the issue** - So you can track the technical implementation
5. **Your project board is configured** - The agent sets up tracking labels and fields

You manage the campaign from the issue. The generated workflow files are implementation details and should not be edited directly.

### Benefits of the issue form approach

- **Captures intent, not YAML**: Focus on what you want to accomplish, not technical syntax
- **Structured validation**: Form fields ensure required information is provided
- **Lower barrier to entry**: No need to understand campaign spec file format
- **Traceable**: Issue serves as the campaign's command center with full history
- **Agent-assisted scaffolding**: Automated generation of spec files and workflows

## Related Patterns

- **[ResearchPlanAssign](/gh-aw/guides/researchplanassign/)** - Research → generate coordinated work
- **[ProjectOps](/gh-aw/examples/issue-pr-events/projectops/)** - Project board integration for campaigns
- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Cross-repository operations
- **[Cache & Memory](/gh-aw/reference/memory/)** - Persistent storage for campaign data
- **[Safe Outputs](/gh-aw/reference/safe-outputs/)** - `create-issue`, `add-comment` for campaigns
