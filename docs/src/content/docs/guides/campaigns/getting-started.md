---
title: Getting Started with Campaigns
description: Create your first agentic campaign in minutes using the automated creation flow
---

This guide shows you how to create and launch your first campaign. You'll go from idea to running campaign in about 5 minutes.

> [!IMPORTANT]
> **Use the automated creation flow** described below. It's the only supported way to create campaigns and handles all the complexity automatically.

## Prerequisites

Before creating a campaign, ensure you have:

- **GitHub repository access** with ability to create issues and workflows
- **GitHub Actions enabled** in your repository
- **At least one agentic workflow** (or a plan to create one)
- **Clear campaign goal** you can describe in 1-2 sentences

> [!TIP]
> Don't have workflows yet? That's okay! The campaign generator can help identify which workflows you need.

## Best practices

Keep these principles in mind when planning your campaign:

**Start small and focused:**
- One clear goal per campaign (e.g., "Upgrade Node.js to v20" not "Improve everything")
- 2-3 workflows maximum for your first campaign
- Simple KPIs that are easy to measure

**Leverage existing work:**
- Search for existing workflows before creating new ones
- Reuse workflows from the [agentics collection](https://github.com/githubnext/agentics)
- Campaign-agnostic workflows are easier to maintain

**Design for safety:**
- Grant minimal permissions (issues/draft PRs, not automatic merges)
- Use standardized output patterns (consistent labels, issue formats)
- Start with low governance limits, increase as you gain confidence

**Plan for human oversight:**
- Workflows should create issues for review, not make final decisions
- Build in approval gates for risky operations
- Escalate to humans when uncertain

## Create your first campaign

### Step 1: Create a campaign issue

1. Navigate to your repository on GitHub
2. Go to **Issues** → **New Issue**
3. Write a descriptive title that captures your goal
   - ✅ Good: "Upgrade all services to Node.js 20"
   - ❌ Avoid: "Node upgrade" or "Fix things"
4. In the issue body, describe:
   - **What** you want to accomplish
   - **Why** this campaign is important
   - **Scope**: Which repositories or services are included
   - **Success criteria**: How you'll know it's done
5. Apply the **`create-agentic-campaign`** label

> [!TIP]
> If your repository has the "🚀 Start an Agentic Campaign" issue template, use it—the label is applied automatically.

### Step 2: Wait for automated generation

The campaign generator workflow kicks off automatically and completes two phases:

**Phase 1: Campaign Generation** (~30 seconds)

The workflow will:
1. ✅ Create a GitHub Project board with custom fields (Worker/Workflow, Priority, Status, Start Date, End Date, Effort)
2. ✅ Create three project views (Campaign Roadmap, Task Tracker, Progress Board)
3. ✅ Discover relevant workflows in your repository and the [agentics collection](https://github.com/githubnext/agentics)
4. ✅ Generate the complete campaign specification (`.github/workflows/<id>.campaign.md`)
5. ✅ Update your issue with campaign details and project board link

**Phase 2: Compilation** (~1-2 minutes)

The workflow will:
1. ✅ Assign a Copilot Coding Agent to compile the campaign
2. ✅ Run `gh aw compile <campaign-id>` to generate the orchestrator workflow
3. ✅ Create a pull request with:
   - `.github/workflows/<id>.campaign.md` (campaign specification)
   - `.github/workflows/<id>.campaign.lock.yml` (compiled orchestrator workflow)

> [!NOTE]
> **Total time:** 2-3 minutes from label application to PR creation. The campaign generator workflow shows real-time progress in the Actions tab.

### Step 3: Review the pull request

After 2-3 minutes, you'll have a pull request containing:

**Campaign specification** (`.campaign.md`)
- Campaign ID, name, and description
- Objectives and KPIs with targets
- List of workflows to coordinate
- Governance policies (rate limits, opt-out labels)
- Project board URL

**Compiled orchestrator** (`.campaign.lock.yml`)
- Auto-generated GitHub Actions workflow
- Discovery logic for finding work items
- Project update automation
- Status reporting configuration

**What to review:**

1. **Verify the goal is clear**: Does the objective match what you intended?
2. **Check the KPIs**: Are they measurable and relevant?
3. **Review workflows**: Do the discovered workflows make sense for your goal?
4. **Confirm governance limits**: Are the rate limits appropriate for your team?

> [!TIP]
> The specification file (`.campaign.md`) is human-readable YAML. You can edit it directly to adjust objectives, KPIs, or governance policies before merging.

### Step 4: Merge and launch

Once you're satisfied with the campaign configuration:

1. **Merge the pull request** to add the campaign to your repository
2. **Navigate to Actions** tab in GitHub
3. **Find your campaign orchestrator** workflow (named after your campaign)
4. **Click "Run workflow"** to start the first orchestration run

**What happens on first run:**

The orchestrator will:
- Create an Epic issue representing the overall campaign
- Add the Epic to your project board
- Discover existing work items (if any workflows have run)
- Generate the first progress status update
- Set up state tracking for future runs

> [!NOTE]
> **Default schedule:** Orchestrators run daily at 6 PM UTC. You can manually trigger runs anytime from the Actions tab.

## Understanding workflow discovery

The campaign generator automatically finds and suggests workflows for your campaign:

**Discovery sources:**

1. **Your repository** (`.github/workflows/*.md`)
   - All agentic workflow files in your workflows directory
   - Parsed frontmatter shows capabilities and triggers

2. **Regular GitHub Actions** (`.github/workflows/*.yml`)
   - Existing automation that could be enhanced with AI
   - Assessed for potential as campaign workers

3. **Agentics collection** ([githubnext/agentics](https://github.com/githubnext/agentics))
   - 17+ pre-built reusable workflows for common tasks
   - Categories: triage & analysis, research & planning, coding & development

**Why this matters:**

- **No manual catalog maintenance** – Discovery happens automatically every time
- **Always up-to-date** – New workflows are found immediately
- **Accurate suggestions** – Based on actual workflow definitions, not static metadata
- **Comprehensive coverage** – Finds all workflows without configuration

> [!TIP]
> Browse the [agentics collection](https://github.com/githubnext/agentics) before creating custom workflows. You might find what you need already built and tested.

## Adding work to your campaign

Once your campaign is running, you can add work items in two ways:

### Automatic discovery (recommended)

Workflows automatically tag their outputs with campaign labels:

```yaml
# Worker workflow creates issues with tracker label
safe-outputs:
  create-issue:
    labels:
      - "campaign:framework-upgrade"
```

The orchestrator discovers these items on its next run and adds them to the project board automatically.

### Manual addition

You can also manually add existing issues or PRs:

1. Open the issue or pull request
2. Apply the campaign tracker label (e.g., `campaign:framework-upgrade`)
3. Wait for the next orchestrator run (or trigger manually)

> [!IMPORTANT]
> **Campaign item protection:** Once an item has a `campaign:*` label, other automated workflows (like issue-monster) will skip it. This prevents conflicts and ensures only your campaign orchestrator manages these items.

## Next steps

Now that your campaign is running:

- **[Monitor progress](/gh-aw/guides/campaigns/project-management/)** on your GitHub Project board
- **[Understand the flow](/gh-aw/guides/campaigns/flow/)** to see how orchestrators work
- **[Configure governance](/gh-aw/guides/campaigns/specs/#governance-pacing--safety)** to adjust rate limits and policies
