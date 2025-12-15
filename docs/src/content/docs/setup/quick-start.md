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

- ✅ **[GitHub CLI](https://cli.github.com/)** installed and authenticated
  - *Why:* Required to compile workflows and deploy them to your repository
  - *How to verify:* Run `gh --version` (requires v2.0.0 or higher) and `gh auth login` to authenticate
- ✅ **GitHub account** with access to a repository (personal or organization)
  - *Why:* Agentic workflows run as GitHub Actions in your repositories
- ✅ **Admin or write access** to your target repository
  - *Why:* You need permission to add workflows, enable Actions, and configure secrets
- ✅ **GitHub Actions** enabled in your repository
  - *Why:* Agentic workflows compile to GitHub Actions YAML files that run on GitHub's infrastructure
- ✅ **Git** installed on your machine
  - *Why:* Used to manage repository files and push workflow changes
- ✅ **Operating System:** Linux, macOS, or Windows with WSL
  - *Why:* The CLI tools and workflow compilation require a Unix-like environment

:::tip[Optional: Enable Discussions]
The example workflow in this guide creates a daily status report as a **GitHub Discussion** post. To use this example as-is, enable Discussions in your repository settings. Otherwise, you can customize the workflow to use different outputs (like issues or comments).
:::

**Verify your setup:**

```bash
gh --version      # Should show version 2.0.0 or higher
gh auth status    # Should show "Logged in to github.com"
git --version     # Should show git version 2.x or higher
```

:::caution[You'll need later (Step 3)]
**Personal Access Token (PAT)** with Copilot Requests permission ([create token](https://github.com/settings/personal-access-tokens/new))
- *Why:* This token authenticates your workflows to use [GitHub Copilot](https://github.com/features/copilot) as the AI agent that executes your natural language instructions
- *Note:* "Copilot Requests" is a new permission type that allows workflows to communicate with GitHub Copilot's AI services
- *Important:* Requires an active GitHub Copilot subscription
- The guide walks you through creating this token in Step 3
:::

### Agentic Setup

If you want to use the help of Copilot to configure GitHub Agentic Workflows,
launch this command:

```bash wrap
npx --yes @github/copilot -i "activate https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install.md"
```

## Understanding Compilation

:::tip[What is "compilation" in agentic workflows?]

You'll write workflows in simple **Markdown files** (`.md`). GitHub Agentic Workflows **compiles** them into **GitHub Actions YAML files** ([`.lock.yml`](/gh-aw/reference/glossary/#workflow-lock-file-lockyml)) that GitHub can execute. Think of it like Markdown → HTML: you write in an easy format, and it gets converted to what GitHub needs.

**Why compile?** It translates your natural language instructions into precise GitHub Actions configuration with security hardening applied.

**No compiler installation needed** — the `gh aw compile` command handles everything automatically.

👉 Learn more in the [Compilation Process](/gh-aw/reference/compilation-process/) documentation.

:::

## How Agentic Workflows Work

Before installing anything, it helps to understand the workflow lifecycle:

```
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

Agentic workflows need to authenticate with an AI service to execute your natural language instructions. By default, they use **GitHub Copilot** as the [coding agent](/gh-aw/reference/glossary/#agent) (the AI that executes your workflow instructions).

To allow your workflows to use Copilot, you'll create a token and add it as a repository secret.

#### Create a Personal Access Token (PAT)

A [Personal Access Token](/gh-aw/reference/glossary/#personal-access-token-pat) is like a password that gives your workflows permission to use GitHub Copilot on your behalf.

**Step-by-step instructions:**

1. **Open the token creation page**: Visit <https://github.com/settings/personal-access-tokens/new>

2. **Configure basic settings:**
   - **Token name**: Enter a descriptive name like "Agentic Workflows Copilot"
   - **Expiration**: Choose an expiration date (90 days recommended for testing)
   - **Description**: Add a note like "For agentic workflows to use Copilot"

3. **Select Resource owner**: Choose your **personal user account** (not an organization)
   - This is required for the Copilot Requests permission to be available

4. **Select Repository access**: Choose **"Public repositories"** 
   - The Copilot Requests permission only appears when public repositories are selected
   - You can also choose "All repositories" or "Only select repositories" if you prefer

5. **Add the Copilot permission:**
   - Scroll to **"Account permissions"** section (not Repository permissions)
   - Find **"Copilot Requests"** and set it to **"Access: Read and write"**
   - This permission allows your workflows to send requests to GitHub Copilot

6. **Generate the token**: Click **"Generate token"** at the bottom of the page

7. **Copy your token**: The token will be displayed once—copy it now! You won't be able to see it again.

:::tip[Can't find Copilot Requests permission?]
The "Copilot Requests" permission is only available if:

- ✅ You have an active [GitHub Copilot subscription](https://github.com/settings/copilot)
- ✅ You're creating a **fine-grained token** (not a classic token)
- ✅ You selected your **personal user account** as Resource owner
- ✅ You selected **"Public repositories"** or "All repositories" under Repository access

If you still don't see this option:
- Check if your Copilot subscription is active
- Ensure you're on the correct token type (fine-grained, not classic)
- Contact your GitHub administrator if Copilot is managed by your organization

:::

#### Add the token to your repository

Now store this token as a secret in your repository so workflows can use it:

1. Navigate to **your repository** on GitHub.com
2. Click **Settings** in the repository navigation menu
3. In the left sidebar, click **Secrets and variables** → **Actions**
4. Click **New repository secret**
5. Configure the secret:
   - **Name**: `COPILOT_GITHUB_TOKEN` (must be exact)
   - **Secret**: Paste the personal access token you just copied
6. Click **Add secret**

✅ Your repository is now configured to run agentic workflows with GitHub Copilot!

:::note[Security note]
Repository secrets are encrypted and never exposed in workflow logs. Only workflows you add to the repository can access them.
:::

For more information, see the [GitHub Copilot CLI documentation](https://github.com/github/copilot-cli?tab=readme-ov-file#authenticate-with-a-personal-access-token-pat).

#### Verify your setup

Before running workflows, verify everything is configured correctly:

```bash wrap
gh aw status
```

**Expected output:**
```
Workflow                 Engine    State     Enabled  Schedule
──────────────────────────────────────────────────────────────
daily-team-status        copilot   ✓         Yes      0 9 * * 1-5
```

This confirms:
- ✓ The workflow is compiled and ready
- ✓ It's enabled for execution
- ✓ The schedule is configured correctly

:::tip[Troubleshooting]
**If the workflow isn't listed:**
- Run `gh aw compile` to compile the workflow
- Check that `.github/workflows/daily-team-status.md` exists

**If you see errors when running the workflow:**
- Verify `COPILOT_GITHUB_TOKEN` secret is set correctly in repository settings
- Check that your token has "Copilot Requests" permission
- Ensure your token hasn't expired
- Run `gh aw secrets bootstrap --engine copilot` to check token configuration
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

**Understanding the structure:**

- **[Frontmatter](/gh-aw/reference/glossary/#frontmatter)** (between `---` markers): The configuration section with [YAML](/gh-aw/reference/glossary/#yaml) settings that control when the workflow runs (`on:`), what permissions it has (`permissions:`), what tools it can use (`tools:`), and what outputs it can create (`safe-outputs:`).

- **Markdown body** (below frontmatter): Natural language instructions that tell the AI [agent](/gh-aw/reference/glossary/#agent) what tasks to perform. Written in plain English, not code.

- **[Safe outputs](/gh-aw/reference/glossary/#safe-outputs)**: Pre-approved actions (like creating discussions, issues, or comments) that the AI can request without needing write permissions during execution. The workflow processes these requests in separate, permission-controlled jobs for security.

## Customize Your Workflow

Customize your workflow by editing the `.md` file and recompiling with `gh aw compile`.
You can leverage the help of an agent to customize your workflow without having to learn about the YAML syntax. Run the following command to start an interactive session with GitHub Copilot CLI.

```bash wrap
# install copilot cli
npm install -g @github/copilot-cli
# install the prompt files
gh aw init
```

Then, run the following to create and edit your workflow:

```bash wrap
# start an interactive session to customize the workflow
copilot
> /agent
> select create-agentic-workflow
> edit @.github/workflows/daily-team-status.md
```

## What's next?

Use [Authoring Agentic Workflows](/gh-aw/setup/agentic-authoring/) to create workflows with AI assistance, explore more samples in the [agentics](https://github.com/githubnext/agentics) repository, and learn about workflow management in [Packaging & Distribution](/gh-aw/guides/packaging-imports/). To understand how agentic workflows work, read [How It Works](/gh-aw/introduction/how-it-works/).
