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

Before you begin, make sure you have:

- [ ] **GitHub account** with access to a repository where you can:
  - Push changes and add Actions secrets
  - Enable GitHub Actions, Issues, and Discussions
- [ ] **GitHub CLI (gh)** installed ([installation guide](https://cli.github.com))
  - Verify: Run `gh --version` (requires v2.0.0 or higher)
- [ ] **Repository permissions:** Admin or write access to your target repository
- [ ] **Repository features enabled:** 
  - GitHub Actions
  - Issues or Discussions (depending on your workflow needs)
- [ ] **Operating System:** Linux, macOS, or Windows with WSL
- [ ] **Personal Access Token (PAT)** for GitHub Copilot CLI (a secure key for API access—you'll create this in Step 3 below)

### Step 1 — Install the extension

Install the [GitHub CLI](https://cli.github.com/), then install the GitHub Agentic Workflows extension:

```bash wrap
gh extension install githubnext/gh-aw
```

If this step fails, you may need to use a personal access token or run the [install-gh-aw.sh script](https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh).

### Step 2 — Add a sample workflow

Add a sample from the [agentics](https://github.com/githubnext/agentics) collection. From your repository root run:

```bash wrap
gh aw add githubnext/agentics/daily-team-status --pr
```

This creates a pull request that adds `.github/workflows/daily-team-status.md` and the [compiled](/gh-aw/reference/glossary/#compilation) `.lock.yml` (the generated GitHub Actions workflow file). The [compilation](/gh-aw/reference/glossary/#compilation) process (converting markdown to GitHub Actions YAML) translates your human-friendly markdown into the YAML format that GitHub Actions can execute. Review and merge the PR into your repo.

#### Why Compile?

The `.md` file is human-friendly (natural language + simple config). GitHub Actions requires `.yml` format. The compile step translates your markdown into the YAML workflow file that GitHub Actions can execute.

Think of it like: **Markdown** (what you write) → **YAML** (what Actions runs)

:::note
The compiled `.lock.yml` file is auto-generated—you edit the `.md` file and recompile whenever you make changes.
:::

### Step 3 — Add an AI secret

Agentic workflows use a [coding agent](/gh-aw/reference/glossary/#agent) (the AI that executes your workflow instructions): GitHub Copilot CLI (default). To authenticate with the Copilot API, you need to create a Personal Access Token (PAT) with the "Copilot Requests" permission—this permission allows your workflow to communicate with GitHub Copilot's coding agent.

**For GitHub Copilot CLI**, create a fine-grained [Personal Access Token (PAT)](/gh-aw/reference/glossary/#personal-access-token-pat) with the "Copilot Requests" permission enabled:

1. Visit https://github.com/settings/personal-access-tokens/new
2. Under "Resource owner", select **your user account** (not an organization—organization-owned tokens cannot access Copilot)
3. Under "Repository access", select **"Public repositories"** (required for the Copilot Requests permission to appear)
4. Under "Permissions", expand "Account permissions" and enable **"Copilot Requests"** (read access)
5. Generate your token
6. Add the token to your repository secrets as `COPILOT_GITHUB_TOKEN`:

```bash wrap
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "<your-personal-access-token>"
```

:::tip[Can't find the Copilot Requests permission?]
The "Copilot Requests" permission only appears if:
- You have an **active GitHub Copilot subscription** (Individual, Business, or Enterprise)
- You selected **"Public repositories"** under Repository access (step 3)
- You selected **your user account** as Resource owner (step 2), not an organization
- You're creating a **fine-grained token** (not a classic token)

If you don't see this option after checking the above, verify your [Copilot subscription status](https://github.com/settings/copilot) or contact your GitHub administrator.
:::

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
