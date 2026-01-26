---
title: Quick Start
description: Get your first agentic workflow running in minutes. Install the extension, add a sample workflow, set up secrets, and run your first AI-powered automation.
sidebar:
  order: 1
---

## Adding a Daily Status Workflow to Your Repo

In this guide you will add the automated [**Daily Status Report**](https://github.com/githubnext/agentics/blob/main/workflows/daily-team-status.md?plain=1) to an existing GitHub repository where you are a maintainer.

Remember the aim here is _automated AI_: to install something that will run _automatically_ every day, in the context of your repository, and create a fresh status report issue in your repository without any further manual intervention.

There are hundreds of other ways to use GitHub Agentic Workflows too, which you can explore in [Peli's Agent Factory](https://githubnext.github.io/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/). This workflow is just the start of what's possible, to get you familiar with the installation and setup process.

## Prerequisites

Before installing, ensure you have:

- ✅ **AI Account:** A GitHub Copilot, Anthropic Claude or OpenAI Codex subscription
- ✅ **GitHub Repository** you are a maintainer on
- ✅ **[GitHub Actions](https://docs.github.com/actions)** enabled in your repository
- ✅ **GitHub CLI** (`gh`) - A command-line tool for GitHub operations. [Install here](https://cli.github.com) v2.0.0+ and authenticate with `gh auth login`
- ✅ **Git** installed on your machine
- ✅ **Operating System:** Linux, macOS, or Windows with WSL

**Verify your setup:**

```bash
gh --version      # Should show version 2.0.0 or higher
gh auth status    # Should show "Logged in to github.com"
git --version     # Should show git version 2.x or higher
```

### Step 1 — Install the extension

Install the [GitHub CLI](https://cli.github.com/), then install the GitHub Agentic Workflows extension:

```bash wrap
gh extension install githubnext/gh-aw
```

> [!TIP]
> Working in GitHub Codespaces?
>
> If you're working in a GitHub Codespace, use the standalone installer instead:
>
> ```bash wrap
> curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
> ```

### Step 2 — Initialize Agentic Workflows support in your repository

Initialize agentic workflows in your repository, to configure optional additional supporting files and settings:

```bash wrap
gh aw init --push
```

This command installs tools and automatically commits and pushes the changes to your repository.

> [!TIP]
>
> If you have branch protection rules enabled, replace `--push` with `--create-pull-request`, then review and merge the pull request.

### Step 3 — Add a sample workflow

Add a sample from the [agentics](https://github.com/githubnext/agentics) collection. From your repository root run:

```bash wrap
gh aw add githubnext/agentics/daily-team-status --push
```

This adds `.github/workflows/daily-team-status.md` and `.github/workflows/daily-team-status.lock.yml` to your repository.  The second file is the [compiled](/gh-aw/reference/glossary/#compilation) GitHub Actions workflow file corresponding to the agentic workflow.

> [!TIP]
>
> If you have branch protection rules enabled, replace `--push` with `--create-pull-request`, then review and merge the pull request.

### Step 4 — Add an AI secret (Copilot Users)

[Agentic workflows](/gh-aw/reference/glossary/#agentic-workflow) need to authenticate with an AI service to execute. By default, they use **GitHub Copilot** as the AI service, but you can also use **Anthropic Claude** or **OpenAI Codex**.

The instructions below assume you have an active [GitHub Copilot subscription](https://github.com/settings/copilot). Claude/Codex Users see [AI Engines](/gh-aw/reference/engines/).

#### Copilot Users: Create a Personal Access Token (PAT)

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

> [!TIP]
> Can't find Copilot Requests permission?
>
> This requires an active [GitHub Copilot subscription](https://github.com/settings/copilot), a fine-grained token (not classic), personal account as Resource owner, and "Public repositories" or "All repositories" selected. Contact your GitHub administrator if Copilot is managed by your organization.
>

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

> [!TIP]
> Troubleshooting
>
> If the workflow isn't listed, run `gh aw compile` and verify `.github/workflows/daily-team-status.md` exists, and add and push it to your repo. If errors occur when running, verify the `COPILOT_GITHUB_TOKEN` secret is set with "Copilot Requests" permission and hasn't expired. Run `gh aw secrets bootstrap --engine copilot` to check configuration.

### Step 5 — Trigger a workflow run

Trigger the workflow immediately in GitHub Actions (this may fail in a codespace):

```bash wrap
gh aw run daily-team-status
```

After a few moments, check the status:

```bash wrap
gh aw status
```

Once complete, a new issue will be created in your repository with a daily team status report! The report will be automatically generated by the AI based on recent activity in your repository, including issues, PRs, discussions, releases, and code changes.

You have successfully installed your first automated agentic workflow into your repository.

## What's next?

Next up is [Authoring Agentic Workflows](/gh-aw/setup/agentic-authoring/) where you will learn how to create automated workflows with AI assistance. You can also explore the samples in [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/). To understand how agentic workflows work, read [How They Work](/gh-aw/introduction/how-they-work/).
