---
title: Getting started
description: Quick start guide for creating and launching agentic campaigns
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

This guide shows you how to create your first campaign using the automated creation flow.

> [!IMPORTANT]
> **Automated creation is the only supported way to create campaigns.** It creates the Project, spec, and orchestrator for you.

## Quick start

1. Create an issue describing the goal
2. Apply the `create-agentic-campaign` label
3. Review the generated pull request
4. Merge and run the orchestrator from the Actions tab

> [!IMPORTANT]
> Use the automated campaign creation flow—it's the only supported way to create campaigns.

## Create a campaign (supported flow)

1. Create an issue describing the goal and scope.
2. Apply the `create-agentic-campaign` label.
3. Wait for a pull request to appear (usually a couple of minutes).
4. Review and merge the PR.
5. Go to Actions and run the campaign orchestrator workflow.

## What you’ll see after merge

- A **Project board** for tracking progress
- A **campaign spec** file (`.github/workflows/<id>.campaign.md`)
- A compiled **orchestrator** workflow (`.github/workflows/<id>.campaign.lock.yml`)

## Run it day-to-day

The orchestrator runs on a schedule (daily by default) and will:

- (Optional) dispatch worker workflows via `workflow_dispatch`
- sync issues/PRs into the Project
- post a Project status update each run

For details, see [Campaign lifecycle](/gh-aw/guides/campaigns/lifecycle/).

## Keep it simple (best practices)

- Start with one goal and 1–3 workflows.
- Keep worker workflows dispatchable (`workflow_dispatch`) and remove other triggers if the campaign is responsible for running them.
- Use conservative governance limits at first (e.g., 10 updates per run).

<details>
<summary>What gets created for you?</summary>

- A Project with standard fields and views
- A campaign spec wired to that Project
- A compiled orchestrator workflow

You’ll review everything in the generated pull request before it runs.
</details>
