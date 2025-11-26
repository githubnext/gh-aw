---
title: Quick Start
description: Get your first agentic workflow running in minutes. Install the extension, add a sample workflow, set up secrets, and run your first AI-powered automation.
sidebar:
  order: 1
---

> [!WARNING]
> **GitHub Agentic Workflows** is a *research demonstrator* in early development and may change significantly.
> Using [agentic workflows](/gh-aw/reference/glossary/#agentic-workflow) (AI-powered automation that can make decisions) requires careful attention to security considerations and human supervision.
> Review all outputs carefully and use time-limited trials to evaluate effectiveness for your team.

## Prerequisites

This guide walks you through setup step-by-step, so you don't need everything upfront. Here's what you need at each stage:

:::note[Must Have (before you start)]
- **GitHub account** with access to a repository
- **GitHub CLI (gh)** installed and authenticated ([installation guide](https://cli.github.com))
  - Verify installation: Run `gh --version` (requires v2.0.0 or higher)
  - Verify authentication: Run `gh auth status`
- **Operating System:** Linux, macOS, or Windows with WSL
:::

:::tip[Must Configure (in your repository)]
- **Admin or write access** to your target repository
- **GitHub Actions** enabled
- **Discussions** enabled (required for the sample workflow in this guide)
:::

:::caution[Will Need Later (you'll set this up in Step 3)]
- **Personal Access Token (PAT)** with Copilot Requests permission—the guide walks you through creating this
:::

### Step 1 — Install the extension

Install the [GitHub CLI](https://cli.github.com/), then install the GitHub Agentic Workflows extension:

```bash wrap
gh extension install githubnext/gh-aw
```

:::caution[Installation fails in Codespaces?]
If you're working in a GitHub Codespace (especially outside the githubnext organization), the extension installation may fail due to authentication or permission issues. Use the standalone installer instead:

```bash wrap
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

After installation, run the binary directly with `./gh-aw` instead of `gh aw`.
:::

### Understanding Compilation

Before adding a workflow, it helps to understand how agentic workflows work.

**You write** `.md` → **Compile** `gh aw compile` → **GitHub Actions runs** `.lock.yml`

The `.md` file is human-friendly (natural language + simple config). GitHub Actions requires `.yml` format. The compile step translates your markdown into the YAML workflow file that GitHub Actions can execute.

:::caution
**Never edit `.lock.yml` files directly.** These are auto-generated files. Always edit the `.md` file and recompile with `gh aw compile` whenever you make changes.
:::

### Step 2 — Add a sample workflow

Add a sample from the [agentics](https://github.com/githubnext/agentics) collection. From your repository root run:

```bash wrap
gh aw add githubnext/agentics/daily-team-status --create-pull-request
```

This creates a pull request that adds `.github/workflows/daily-team-status.md` and the [compiled](/gh-aw/reference/glossary/#compilation) `.lock.yml` (the generated GitHub Actions workflow file). Review and merge the PR into your repo.

### Step 3 — Add an AI secret

Agentic workflows use a [coding agent](/gh-aw/reference/glossary/#agent) (the AI that executes your workflow instructions): GitHub Copilot CLI (default).

**For GitHub Copilot CLI**, create a fine-grained [Personal Access Token (PAT)](/gh-aw/reference/glossary/#personal-access-token-pat) with the "Copilot Requests" permission enabled. This permission allows your workflow to communicate with GitHub Copilot's coding agent to execute AI-powered tasks.

:::tip[Can't find Copilot Requests permission?]
The "Copilot Requests" permission is only available if:
- You have an active GitHub Copilot subscription (individual or through your organization)
- You're creating a **fine-grained token** (not a classic token)
- You select your **personal user account** as the Resource owner (not an organization)
- You select **"Public repositories"** under Repository access

If you still don't see this option:
- Check your [Copilot subscription status](https://github.com/settings/copilot)
- Contact your GitHub administrator if Copilot is managed by your organization
:::

1. Visit https://github.com/settings/personal-access-tokens/new
2. Under "Resource owner", select your user account (not an organization).
3. Under "Repository access," select **"Public repositories"** (required for the Copilot Requests permission to appear).
4. Under "Permissions," click "Add permissions" and select **"Copilot Requests"**.
5. Generate your token.
6. Add the token to your repository secrets as `COPILOT_GITHUB_TOKEN`:

```bash wrap
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "<your-personal-access-token>"
```

For more information, see the [GitHub Copilot CLI documentation](https://github.com/github/copilot-cli?tab=readme-ov-file#authenticate-with-a-personal-access-token-pat).

### Step 4 — Trigger a workflow run

Trigger the workflow immediately in GitHub Actions:

```bash wrap
gh aw run daily-team-status
```

After a few moments, check the status:

```bash wrap
gh aw status
```

Once complete, a new discussion post will be created in your repository with a daily news! The report will be automatically generated by the AI based on recent activity in your repository.

## Understanding Your First Workflow

Let's look at what you just added. The daily team status workflow automatically creates a status report every weekday and posts it as a discussion in your repository, and looks like this:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1-5"
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
network: defaults
tools:
  github:
safe-outputs:
  create-discussion:
---

# Daily Team Status

Create an upbeat daily status report for the team as a GitHub discussion.
- Recent repository activity (issues, PRs, discussions, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

...

1. Gather recent activity from the repository
2. Create a new GitHub discussion with your findings and insights
```

This workflow triggers every weekday at 9 AM via cron schedule, has [permissions](/gh-aw/reference/glossary/#permissions) to read repository content and create GitHub discussions, and runs AI instructions in natural language to generate status reports.

The section between the `---` markers (called [frontmatter](/gh-aw/reference/glossary/#frontmatter), the configuration section at the top of a workflow file) contains the [YAML](/gh-aw/reference/glossary/#yaml) configuration that defines when the workflow runs, what permissions it has, and what tools it can use. The section below the frontmatter contains the natural language instructions that tell the AI agent what to do. The [`safe-outputs`](/gh-aw/reference/glossary/#safe-outputs) section specifies that this workflow can safely create GitHub discussions without needing write permissions during the AI execution phase.

## Customize Your Workflow

Customize your workflow by editing the `.md` file and recompiling with `gh aw compile`. 
You can leverage the help of an agent to customize your workflow without having to learn about the YAML syntax. Run the following command to start an interactive session with GitHub Copilot CLI.

```bash wrap
# install copilot cli
npm install -g @github/copilot-cli
# install the custom agent files
gh aw init
```

Then, run the following to create and edit your workflow:

```bash wrap
# start an interactive session to customize the workflow
copilot
> /agent create-agentic-workflow
> edit @.github/workflows/daily-team-status.md
```

## What's next?

Use [Authoring Agentic Workflows](/gh-aw/setup/agentic-authoring/) to create workflows with AI assistance, explore more samples in the [agentics](https://github.com/githubnext/agentics) repository, and learn about workflow management in [Packaging & Distribution](/gh-aw/guides/packaging-imports/). To understand how agentic workflows work, read [How It Works](/gh-aw/introduction/how-it-works/).
