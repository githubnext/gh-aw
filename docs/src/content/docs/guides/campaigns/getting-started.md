---
title: "Getting Started"
description: "Quick start guide for creating and launching agentic campaigns"
---

This guide walks through the fastest way to create and launch an agentic campaign.

## Quick start

1. Create an agentic campaign spec: `.github/workflows/<id>.campaign.md`.
2. Reference one or more workflows in `workflows:`.
3. Set `project-url` to the org Project v2 URL you use as the agentic campaign dashboard.
4. Add a `tracker-label` so issues/PRs can be queried consistently.
5. Run `gh aw compile` to validate campaign specs and compile workflows.

## Lowest-friction walkthrough (recommended)

The simplest, least-permissions way to run an agentic campaign is:

### 1. Create the agentic campaign spec (in a PR)

Choose one of these approaches:

**Option A (No-code)**: Use the "🚀 start an agentic campaign" issue form in the GitHub UI to capture intent with structured fields. The form creates an agentic campaign issue, and an agent can scaffold the spec file for you.

**Option B (CLI)**: Use `gh aw campaign new <id>` to generate an agentic campaign spec file locally.

**Option C (Manual)**: Author `.github/workflows/<id>.campaign.md` manually.

### 2. Create the org Project board once (manual)

Create an org Project v2 in the GitHub UI and copy its URL into `project-url`.

This avoids requiring a PAT or GitHub App setup just to provision boards.

Minimum clicks (one-time setup):
- In GitHub: your org → **Projects** → **New project**.
- Give it a name (for example: `Code Health: <Campaign Name>`).
- Choose any starting layout (Table/Board). You can change views later.
- Copy the Project URL and set it as `project-url` in the agentic campaign spec.

Optional but recommended for "kanban lanes":
- Create a **Board** view and set **Group by** to a single-select field (commonly `Status`).
- Note: workflows can create/update fields and single-select options, but they do not currently create or configure Project views.

### 3. Have workflows keep the board in sync using `GITHUB_TOKEN`

Enable the `update-project` safe output in the launcher/monitor workflows.

Default behavior is **update-only**: if the board does not exist, the project job fails with instructions.

### 4. Opt in to auto-creating the board only when you intend to

If you want workflows to create missing boards, explicitly set `create_if_missing: true` in the `update_project` output.

For many orgs, you may also need a token override (`safe-outputs.update-project.github-token`) with sufficient org Project permissions.

## Start an Agentic Campaign with GitHub Issue Forms

For a low-code/no-code approach, you can create an agentic campaign using the GitHub UI with the "🚀 Start an Agentic Campaign" issue form:

1. **Go to the repository's Issues tab** and click "New issue"
2. **Select "🚀 Start an Agentic Campaign"** from the available templates
3. **Fill in the structured form fields**:
   - **Campaign Name** (required): Human-readable name (e.g., "Framework Upgrade Q1 2025")
   - **Campaign Identifier** (required): Unique ID using lowercase letters, digits, and hyphens (e.g., "framework-upgrade-q1-2025")
   - **Campaign Version** (required): Version string (defaults to "v1")
   - **Project Board URL** (required): URL of the GitHub Project you created to serve as the agentic campaign dashboard
   - **Campaign Type** (optional): Select from Migration, Upgrade/Modernization, Security Remediation, etc.
   - **Scope** (optional): Define what repositories, components, or areas will be affected
   - **Constraints** (optional): List any constraints or requirements (deadlines, approvals, etc.)
   - **Prior Learnings** (optional): Share relevant learnings from past similar campaigns
4. **Submit the form** to create the agentic campaign issue

### What happens after submission

When you submit the issue form:

1. **an agentic campaign issue is created** - This becomes your campaign's central hub with the `campaign` and `campaign-tracker` labels
2. **An agent validates your project board** - Ensures the URL is accessible and properly configured
3. **an agentic campaign spec is generated** - Creates `.github/workflows/<id>.campaign.md` with your inputs as a PR
4. **The spec is linked to the issue** - So you can track the technical implementation
5. **Your project board is configured** - The agent sets up tracking labels and fields

You manage the agentic campaign from the issue. The generated workflow files are implementation details and should not be edited directly.

### Benefits of the issue form approach

- **Captures intent, not YAML**: Focus on what you want to accomplish, not technical syntax
- **Structured validation**: Form fields ensure required information is provided
- **Lower barrier to entry**: No need to understand campaign spec file format
- **Traceable**: Issue serves as the agentic campaign's command center with full history
- **Agent-assisted scaffolding**: Automated generation of spec files and workflows
