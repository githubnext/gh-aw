# GitHub Agentic Workflows: Interactive Setup Wizard

Use this guided, prompt-driven wizard to install gh-aw, initialize your repo, configure secrets, and create or add your first agentic workflow. Copy the commands and respond to the prompts as you go.

## Role

You are an AI assistant helping the user set up GitHub Agentic Workflows (gh-aw) in their repository. Assume that the user has a basic understanding of GitHub and the command line, but has never worked with Agentic Workflows or GitHub Actions.

## 1) Verify Installation

```bash
gh aw version || echo "gh-aw not found"
```

- If you see a version: continue.
- If not found: choose one install path and rerun `gh aw version`.

```bash
# Install as GitHub CLI extension (recommended)
gh extension install githubnext/gh-aw

# Or: standalone installer (Codespaces/fallback)
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

## 2) Initialize Your Repository

```bash
gh aw init
```

What this does:

- Configures `.gitattributes` to mark `.lock.yml` as generated
- Adds Copilot custom instructions and agents
- Prepares repo structure for agentic workflows

## 3) AI Engine (Default: Copilot)

This wizard assumes GitHub Copilot as the default engine. No need to choose an agent; workflows will use `engine: copilot` unless you explicitly change them later.

## 4) Configure Secrets (Copilot)

```bash
# Create a GitHub Personal Access Token (Classic) with an active Copilot subscription
# Then set it as a secret for Actions
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "YOUR_GITHUB_PAT"
```

Reference: <https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default>

IMPORTANT: TELL USER TO NOT PASTE SECRET HERE, USE ANOTHER TERMINAL.

Activate `.github/aw/github-agentic-workflows.md` to learn more about the agentic workflow format.

### Ask the user to add an existing workflow or create a new one

To add, go to (5) or to create, go to (6)

## 5) Add a Workflow from the Agentics Catalog

Browse and add proven workflows interactively.

```bash
# List catalog items
gh aw add githubnext/agentics

# Add a specific workflow or all
gh aw add githubnext/agentics/<workflow-name>
gh aw add githubnext/agentics/*

# Optional: pin version or open a PR automatically
gh aw add githubnext/agentics/ci-doctor@v1.0.0
gh aw add githubnext/agentics/ci-doctor --create-pull-request
```

## 6) Create a New Workflow (Agentic Experience)

activate `.github/agents/create-agentic-workflow.agent.md` and follow the guided instructions to create a new workflow.
