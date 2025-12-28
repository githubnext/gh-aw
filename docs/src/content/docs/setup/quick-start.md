---
title: Quick Start
description: Get your first agentic workflow running in minutes. Install the extension, add a sample workflow, set up secrets, and run your first AI-powered automation.
sidebar:
  order: 1
---

> [!WARNING]
> **GitHub Agentic Workflows** is a *research demonstrator* in early development and may change significantly.
> Using [agentic workflows](/gh-aw/reference/glossary/#agentic-workflow) means giving AI [agents](/gh-aw/reference/glossary/#agent) (autonomous AI systems) the ability to make decisions and take actions in your repository. This requires careful attention to security considerations and human supervision.
> Review all outputs carefully and use time-limited trials to evaluate effectiveness for your team.

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



### Agentic Setup

If you want to use Copilot to configure GitHub Agentic Workflows, run:

```bash wrap
npx --yes @github/copilot -i "activate https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install.md"
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
- **`.md` file**: Human-friendly markdown with natural language instructions and simple YAML configuration. This is what you write and edit.
- **[`.lock.yml` file](/gh-aw/reference/glossary/#workflow-lock-file-lockyml)**: Machine-ready GitHub Actions YAML with security hardening applied. This is what GitHub Actions runs.
- **Compilation**: The `gh aw compile` command translates your markdown into validated, secure GitHub Actions YAML.

Think of it like writing code in a high-level language (Python, JavaScript) that gets compiled to machine code. You write natural language, GitHub runs the compiled workflow.

:::caution[Important]
**Never edit [`.lock.yml` files](/gh-aw/reference/glossary/#workflow-lock-file-lockyml) directly.** These are auto-generated. Always edit the `.md` file and recompile with `gh aw compile`.
:::

### Step 1 — Install the extension

:::caution[Working in GitHub Codespaces?]
If you're working in a GitHub Codespace (especially outside the githubnext organization), the extension installation may fail due to restricted permissions that prevent global npm installs. Use the standalone installer instead:

```bash wrap
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

After installation, the binary is installed to `~/.local/share/gh/extensions/gh-aw/gh-aw` and can be used with `gh aw` commands just like the extension installation.
:::

Install the [GitHub CLI](https://cli.github.com/), then install the GitHub Agentic Workflows extension:

```bash wrap
gh extension install githubnext/gh-aw
```

### Step 2 — Add a sample workflow

Add a sample from the [agentics](https://github.com/githubnext/agentics) collection. From your repository root run:

```bash wrap
gh aw add githubnext/agentics/daily-team-status --create-pull-request
```

This creates a pull request that adds `.github/workflows/daily-team-status.md` and the [compiled](/gh-aw/reference/glossary/#compilation) `.lock.yml` (the generated GitHub Actions workflow file). Review and merge the PR into your repo.

### Step 3 — Add an AI secret

Agentic workflows need to authenticate with an AI service to execute your natural language instructions. By default, they use **GitHub Copilot** as the [coding agent](/gh-aw/reference/glossary/#agent) (the AI system that executes your instructions).

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
Workflow                 Engine    State     Enabled  Schedule
──────────────────────────────────────────────────────────────
daily-team-status        copilot   ✓         Yes      0 9 * * 1-5
```

This confirms the workflow is compiled, enabled, and scheduled correctly.

:::tip[Troubleshooting]
If the workflow isn't listed, run `gh aw compile` and verify `.github/workflows/daily-team-status.md` exists. If errors occur when running, verify the `COPILOT_GITHUB_TOKEN` secret is set with "Copilot Requests" permission and hasn't expired. Run `gh aw secrets bootstrap --engine copilot` to check configuration.
:::

### Step 4 — Trigger a workflow run

Trigger the workflow immediately in GitHub Actions:

```bash wrap
gh aw run daily-team-status
```

After a few moments, check the status:

```bash wrap
gh aw status
```

Once complete, a new issue will be created in your repository with daily news! The report will be automatically generated by the AI based on recent activity in your repository.

## Understanding Your First Workflow

The daily team status workflow creates a status report every weekday and posts it as an issue. The workflow file has two parts:
- **[Frontmatter](/gh-aw/reference/glossary/#frontmatter)** (YAML configuration section) — The section between `---` markers that configures when the workflow runs and what it can do
- **Markdown instructions** — Natural language task descriptions for the AI

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
  create-issue:
---

# Daily Team Status

Create an upbeat daily status report for the team as a GitHub issue.
- Recent repository activity (issues, PRs, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

...

1. Gather recent activity from the repository
2. Create a new GitHub issue with your findings and insights
```

**Key configuration elements:**
- **[`tools:`](/gh-aw/reference/tools/)** — Capabilities the AI can use (GitHub API access)
- **[`safe-outputs:`](/gh-aw/reference/safe-outputs/)** (validated GitHub API operations) — Allows creating issues without giving the AI write permissions

## What Happens Next

After triggering your workflow, understanding the execution lifecycle helps you monitor, debug, and iterate on your agentic workflows.

### How workflows are triggered

Your workflow runs based on the events configured in the `on:` section of your markdown file:

- **Scheduled workflows** run automatically at specified times (e.g., `cron: "0 9 * * 1-5"` runs weekdays at 9 AM)
- **Event-triggered workflows** run when specific GitHub events occur (e.g., when an issue is opened, a pull request is created)
- **Manual workflows** run on-demand using `gh aw run` or the GitHub UI when `workflow_dispatch:` is configured

The `daily-team-status` workflow you just ran includes both scheduled execution and manual triggering via `workflow_dispatch:`.

Learn more about workflow triggers in the [Triggers reference](/gh-aw/reference/triggers/).

### Viewing workflow runs

To see your workflow execution in GitHub:

1. Navigate to your repository on GitHub.com
2. Click the **Actions** tab at the top of the repository
3. You'll see a list of recent workflow runs with their status:
   - 🟢 Green checkmark: Successful completion
   - 🔴 Red X: Failed execution
   - 🟡 Yellow circle: Currently running
4. Click on any workflow run to see detailed execution steps

You can also check workflow status from the command line:

```bash wrap
gh aw status
```

This shows all your workflows, their current state, and scheduling information.

### Accessing and reading logs

Logs provide visibility into what the AI agent did during workflow execution:

**From the GitHub UI:**
1. Go to the **Actions** tab in your repository
2. Click on a workflow run
3. Click on a job name to expand it
4. View logs for each step, including AI decisions and actions

**From the command line:**
```bash wrap
gh aw logs daily-team-status
```

This downloads and displays recent execution logs, showing AI decisions, tool usage, and any errors.

For detailed analysis of a specific run:
```bash wrap
gh aw audit 12345678
```

Replace `12345678` with the run ID from the workflow run URL.

### Troubleshooting common issues

If your workflow doesn't run as expected:

- **Check secrets**: Verify `COPILOT_GITHUB_TOKEN` is set and hasn't expired using `gh aw secrets bootstrap --engine copilot`
- **Review logs**: Use `gh aw logs` to see detailed execution information and error messages
- **Verify compilation**: Run `gh aw compile` to catch syntax errors before running
- **Check permissions**: Ensure your token has the required permissions (Copilot Requests for Copilot engine)

For comprehensive troubleshooting guidance, see the [Common Issues](/gh-aw/troubleshooting/common-issues/) documentation.

## Customize Your Workflow

Edit the `.md` file and recompile with `gh aw compile`. For AI-assisted customization using GitHub Copilot CLI:

```bash wrap
npm install -g @github/copilot-cli
gh aw init
copilot
> /agent
> select create-agentic-workflow
> edit @.github/workflows/daily-team-status.md
```

## What's next?

Use [Authoring Agentic Workflows](/gh-aw/setup/agentic-authoring/) to create workflows with AI assistance, explore more samples in the [agentics](https://github.com/githubnext/agentics) repository, and learn about workflow management in [Packaging & Distribution](/gh-aw/guides/packaging-imports/). To understand how agentic workflows work, read [How It Works](/gh-aw/introduction/how-it-works/).
