---
title: Quick Start
description: Get your first agentic workflow running in minutes. Install the extension, add a sample workflow, set up secrets, and run your first AI-powered automation.
sidebar:
  order: 1
---

> [!WARNING]
> **GitHub Agentic Workflows** is a *research demonstrator* in early development and may change significantly.
> Using [agentic workflows](/gh-aw/reference/glossary/#agentic-workflow) (AI-powered workflows that can make autonomous decisions) means giving AI [agents](/gh-aw/reference/glossary/#agent) (autonomous AI systems) the ability to make decisions and take actions in your repository. This requires careful attention to security considerations and human supervision.
> Review all outputs carefully and use time-limited trials to evaluate effectiveness for your team.

## Adding a Daily Status Workflow to Your Repo

This is a happy path guide to get you started with automated daily team status reports in an existing GitHub repository you have admin or write access to. If you stumble on one of these steps, go to the [Prerequisites](#prerequisites) section for setup instructions.

### Step 1 — Install the extension

Install the [GitHub CLI](https://cli.github.com/), then install the GitHub Agentic Workflows extension:

```bash wrap
gh extension install githubnext/gh-aw
```

:::caution[Working in GitHub Codespaces?]

If you're working in a GitHub Codespace, the extension installation *may* fail due to restricted permissions that prevent global npm installs. Use the standalone installer instead:

```bash wrap
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

:::

### Step 2 — Add a sample workflow

Add a sample from the [agentics](https://github.com/githubnext/agentics) collection. From your repository root run:

```bash wrap
gh aw add githubnext/agentics/daily-team-status --create-pull-request
```

This creates a pull request that adds `.github/workflows/daily-team-status.md` and the [compiled](/gh-aw/reference/glossary/#compilation) (translated from markdown to GitHub Actions YAML) `.lock.yml` (the generated GitHub Actions workflow file). 
   - Review and merge the PR into your repo.
   - Pull the changes into your (local) repo.

### Step 3 — Add an AI secret

[Agentic workflows](/gh-aw/reference/glossary/#agentic-workflow) (AI-powered workflows) need to authenticate with an AI service to execute your natural language instructions. By default, they use **GitHub Copilot** as the [coding agent](/gh-aw/reference/glossary/#agent) (the AI system that executes your instructions).

To allow your workflows to use Copilot, you'll create a token and add it as a repository secret.

#### Create a Personal Access Token (PAT)

Create a [Personal Access Token](/gh-aw/reference/glossary/#personal-access-token-pat) to authenticate your workflows with GitHub Copilot:

1. Visit <https://github.com/settings/personal-access-tokens/new>
2. Configure the token:
   - **Token name**: "Agentic Workflows Copilot"
   - **Expiration**: 90 days (recommended for testing)
   - **Resource owner**: Your personal account (required for Copilot Requests permission)
   - **Repository access**: "Public repositories" (required for Copilot Requests permission to appear)
3. Add permissions:
   - In **"Account permissions"** (not Repository permissions), find **"Copilot Requests"**
   - Set to **"Access: Read"**
4. Click **"Generate token"** and copy it immediately (you won't see it again)

:::tip[Can't find Copilot Requests permission?]

Requires an active [GitHub Copilot subscription](https://github.com/settings/copilot), a fine-grained token (not classic), personal account as Resource owner, and "Public repositories" or "All repositories" selected. Contact your GitHub administrator if Copilot is managed by your organization.

:::

#### Add the token to your repository

Store the token as a repository secret:

1. Go to **your repository** → **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**
3. Set **Name** to `COPILOT_GITHUB_TOKEN` and paste the token in **Secret**
4. Click **Add secret**

Repository secrets are encrypted and only accessible to workflows in your repository. See [GitHub Copilot CLI documentation](https://github.com/github/copilot-cli?tab=readme-ov-file#authenticate-with-a-personal-access-token-pat) for more details.

#### Verify your setup

Before running workflows, verify everything is configured correctly:

```bash wrap
gh aw status
```

**Expected output:**

```text
┌─────────────────┬───────┬────────┬──────┬──────────────┬──────┬──────────┬──────────────┐
│Workflow         │Engine │Compiled│Status│Time Remaining│Labels│Run Status│Run Conclusion│
├─────────────────┼───────┼────────┼──────┼──────────────┼──────┼──────────┼──────────────┤
│daily-team-status│copilot│No      │active│30d 22h       │-     │-         │-             │
└─────────────────┴───────┴────────┴──────┴──────────────┴──────┴──────────┴──────────────┘
```

This confirms the workflow is compiled, enabled, and scheduled correctly.

:::tip[Troubleshooting]

If the workflow isn't listed, run `gh aw compile` and verify `.github/workflows/daily-team-status.md` exists. If errors occur when running, verify the `COPILOT_GITHUB_TOKEN` secret is set with "Copilot Requests" permission and hasn't expired. Run `gh aw secrets bootstrap --engine copilot` to check configuration.

:::

### Step 4 — Trigger a workflow run

Trigger the workflow immediately in GitHub Actions (this may fail in a codespace):

```bash wrap
gh aw run daily-team-status
```

After a few moments, check the status:

```bash wrap
gh aw status
```

Once complete, a new discussion will be created in your repository with a daily team status report! The report will be automatically generated by the AI based on recent activity in your repository, including issues, PRs, discussions, releases, and code changes.

### Agentic Setup

If you want to use Copilot to configure GitHub Agentic Workflows, run:

```bash wrap
npx --yes @github/copilot -i "activate https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install.md"
```

## Prerequisites

Before installing, ensure you have:

- ✅ **GitHub CLI** (`gh`) - A command-line tool for GitHub operations. [Install here](https://cli.github.com) v2.0.0+ and authenticate with `gh auth login`
- ✅ **GitHub account** with admin or write access to a repository
- ✅ **[GitHub Actions](https://docs.github.com/en/actions)** (GitHub's automation platform) enabled in your repository
- ✅ **Git** installed on your machine
- ✅ **Operating System:** Linux, macOS, or Windows with WSL

**Verify your setup:**

```bash
gh --version      # Should show version 2.0.0 or higher
gh auth status    # Should show "Logged in to github.com"
git --version     # Should show git version 2.x or higher
```

## How Agentic Workflows Work

Before installing anything, it helps to understand the workflow lifecycle:

```text
1. You write       2. Compile           3. GitHub Actions runs
   .md file    →    gh aw compile   →    .lock.yml file
   (natural         (translates to        (GitHub Actions
   language)        GitHub Actions)       executes)
```

**Why two files?**

- **`.md` file**: Human-friendly markdown with natural language instructions and simple YAML [frontmatter](/gh-aw/reference/glossary/#frontmatter) (configuration at the top between `---` markers). This is what you write and edit.
- **[`.lock.yml` file](/gh-aw/reference/glossary/#workflow-lock-file-lockyml)**: Machine-ready GitHub Actions YAML with security hardening applied. This is what GitHub Actions runs.
- **[Compilation](/gh-aw/reference/glossary/#compilation)**: The `gh aw compile` command translates your markdown into validated, secure GitHub Actions YAML.

Think of it like writing code in a high-level language (Python, JavaScript) that gets compiled to machine code. You write natural language, GitHub runs the compiled workflow.

:::caution[Important]
**Never edit [`.lock.yml` files](/gh-aw/reference/glossary/#workflow-lock-file-lockyml) directly.** These are auto-generated. Always edit the `.md` file and recompile with `gh aw compile`.
:::

## Understanding Your First Workflow

The daily team status workflow creates a status report daily and posts it as a discussion. The workflow file has two parts:

- **[Frontmatter](/gh-aw/reference/glossary/#frontmatter)** (YAML configuration section between `---` markers) — Configures when the workflow runs and what it can do
- **Markdown instructions** — Natural language task descriptions for the AI

```aw wrap
---
on:
  schedule: daily
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
    title-prefix: "[team-status] "
    category: "announcements"
---

# Daily Team Status

Create an upbeat daily status report for the team as a GitHub discussion.

## What to include

- Recent repository activity (issues, PRs, discussions, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

## Style

- Be positive, encouraging, and helpful 🌟
- Use emojis moderately for engagement
- Keep it concise - adjust length based on actual activity

## Process

1. Gather recent activity from the repository
2. Create a new GitHub discussion with your findings and insights
```

**Key configuration elements:**

- **[`schedule: daily`](/gh-aw/reference/schedule-syntax/)** — Runs once per day at a randomized time to distribute load
- **[`tools:`](/gh-aw/reference/tools/)** — Capabilities the AI can use (like GitHub API access)
- **[`safe-outputs:`](/gh-aw/reference/safe-outputs/)** (pre-approved GitHub operations) — Allows creating discussions without giving the AI write permissions

## Customize Your Workflow

Edit the `.md` file and recompile with `gh aw compile`. For AI-assisted customization, you can use GitHub Copilot Chat in VS Code or the interactive Copilot CLI session.

To use the interactive Copilot CLI:

```bash wrap
npm install -g @github/copilot-cli
gh aw init
copilot
```

Then in the interactive Copilot session, use the `/agent` command:
```
/agent
> select create-agentic-workflow
> edit @.github/workflows/daily-team-status.md
```

## What's next?

Use [Authoring Agentic Workflows](/gh-aw/setup/agentic-authoring/) to create workflows with AI assistance, explore more samples in the [agentics](https://github.com/githubnext/agentics) repository, and learn about workflow management in [Packaging & Distribution](/gh-aw/guides/packaging-imports/). To understand how agentic workflows work, read [How It Works](/gh-aw/introduction/how-it-works/).
