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

This guide walks you through setup step-by-step, so you don't need everything upfront. Here's what you need at each stage and **why**:

:::note[Must Have (before you start)]

- **GitHub account** with access to a repository
  - *Why:* Agentic workflows run as GitHub Actions in your repositories
- **GitHub CLI (gh)** installed and authenticated ([installation guide](https://cli.github.com))
  - *Why:* The CLI is required to compile workflows and deploy them to your repository
  - Verify installation: Run `gh --version` (requires v2.0.0 or higher)
  - Verify authentication: Run `gh auth status`
- **Operating System:** Linux, macOS, or Windows with WSL
  - *Why:* The CLI tools and workflow compilation require a Unix-like environment

:::

:::tip[Must Configure (in your repository)]

- **Admin or write access** to your target repository
  - *Why:* You need permission to add workflows, enable Actions, and configure secrets
- **GitHub Actions** enabled
  - *Why:* Agentic workflows compile to GitHub Actions YAML files that run on GitHub's infrastructure
- **Discussions** enabled
  - *Why:* The Hello World example creates a discussion post to show a visible result
  - *How to enable:* Go to your repository Settings → General → Features → Check "Discussions"

:::

:::caution[Will Need Later (you'll set this up in Step 3)]

- **Personal Access Token (PAT)** with Copilot Requests permission ([PAT documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens))
  - *Why:* This token authenticates your workflows to use GitHub Copilot as the AI agent that executes your natural language instructions
  - *Note:* "Copilot Requests" is a new permission type added in 2024 that allows workflows to communicate with GitHub Copilot's AI services
  - *Important:* If you don't see this permission, ensure your GitHub account has Copilot access
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

### Step 2 — Create your first workflow

Let's create a simple "Hello World" workflow that introduces you to agentic workflows without overwhelming complexity.

Create a new file `.github/workflows/hello-world.md` in your repository with this content:

```aw wrap
---
# Trigger: Run manually from GitHub Actions UI
on:
  workflow_dispatch:

# Permissions: Read repository information
permissions:
  contents: read

# AI Engine: Use GitHub Copilot
engine: copilot

# Safe Output: Allow creating discussions
safe-outputs:
  create-discussion: {}
---

# My First Agentic Workflow

Create a discussion in the "General" category that says "Hello from my first agentic workflow!"

Include today's date and a fun fact about GitHub Copilot.
```

Now compile the workflow to generate the GitHub Actions YAML file:

```bash wrap
gh aw compile .github/workflows/hello-world.md
```

This creates `.github/workflows/hello-world.lock.yml` (the [compiled](/gh-aw/reference/glossary/#compilation) GitHub Actions workflow file).

Commit both files to your repository:

```bash wrap
git add .github/workflows/hello-world.md .github/workflows/hello-world.lock.yml
git commit -m "Add my first agentic workflow"
git push
```

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

### Step 4 — Trigger your workflow

Trigger your workflow manually in GitHub Actions:

```bash wrap
gh aw run hello-world
```

After a few moments, check the status:

```bash wrap
gh aw status
```

Once complete, a new discussion post will be created in your repository with a friendly "Hello World" message and today's date! The AI will also include an interesting fact about GitHub Copilot.

## Understanding Your First Workflow

Let's look at what you just created. The workflow has two parts: **configuration** (frontmatter) and **instructions** (markdown body).

### Configuration (Frontmatter)

The section between `---` markers configures when and how the workflow runs:

```aw wrap
---
# Trigger: Run manually from GitHub Actions UI
on:
  workflow_dispatch:

# Permissions: Read repository information
permissions:
  contents: read

# AI Engine: Use GitHub Copilot
engine: copilot

# Safe Output: Allow creating discussions
safe-outputs:
  create-discussion: {}
---
```

**Breaking it down:**

- **`on: workflow_dispatch`** - This workflow runs only when you manually trigger it from the GitHub Actions UI or CLI. No automatic scheduling.

- **`permissions: contents: read`** - The AI can read your repository content but cannot make any changes directly. This is a key security feature.

- **`engine: copilot`** - Uses GitHub Copilot as the AI [agent](/gh-aw/reference/glossary/#agent) that executes your instructions.

- **`safe-outputs: create-discussion: {}`** - Pre-approves the AI to request creating a discussion. The actual creation happens in a separate, permission-controlled job for security. The `{}` means use default settings.

:::tip[What is frontmatter?]
[Frontmatter](/gh-aw/reference/glossary/#frontmatter) is the configuration section at the top of markdown files, written in [YAML](/gh-aw/reference/glossary/#yaml) format. It's enclosed by `---` markers and controls how the workflow behaves.

Think of it like the settings panel for your workflow - you're not writing instructions yet, just configuring the environment.
:::

### Instructions (Markdown Body)

The section below the frontmatter contains your natural language instructions:

```markdown
# My First Agentic Workflow

Create a discussion in the "General" category that says "Hello from my first agentic workflow!"

Include today's date and a fun fact about GitHub Copilot.
```

This is plain English telling the AI what to do. The AI reads these instructions, executes them within the configured environment, and uses the `create-discussion` safe-output to post the result.

**Why this works:**

1. The AI reads your repository and the instructions
2. It generates content based on your requirements (greeting, date, fun fact)
3. It requests to create a discussion using the safe-output
4. A separate GitHub Actions job creates the discussion with proper permissions

### Want to Try Something More Complex?

Once you're comfortable with this basic workflow, explore more advanced examples:

- **[Daily Team Status](/gh-aw/examples/scheduled/daily-team-status/)** - Automated status reports with cron scheduling, GitHub API integration, and richer safe-output configuration
- **[Manual Research Workflows](/gh-aw/examples/manual/)** - On-demand workflows with custom inputs
- **[Comment-Triggered Workflows](/gh-aw/examples/comment-triggered/)** - Workflows that respond to issue and PR comments

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
> edit @.github/workflows/hello-world.md
```

## What's next?

Use [Authoring Agentic Workflows](/gh-aw/setup/agentic-authoring/) to create workflows with AI assistance, explore more samples in the [agentics](https://github.com/githubnext/agentics) repository, and learn about workflow management in [Packaging & Distribution](/gh-aw/guides/packaging-imports/). To understand how agentic workflows work, read [How It Works](/gh-aw/introduction/how-it-works/).
