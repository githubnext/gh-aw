# Optimizing Agentic Workflow Token Consumption

This prompt guides you, a coding agent, to reduce the AI token usage and cost of a **GitHub Agentic Workflow (gh-aw)**.

## How to Use This Prompt

There are two ways to invoke this optimization workflow:

### Option A: Through Copilot

If your repository is configured with the `agentic-workflows` skill, use Copilot Chat:

```text
agentic-workflows optimize https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

### Option B: Self-Contained (with URL)

Share this file's URL with any AI assistant or coding agent:

```text
Optimize the agentic workflow token consumption using https://raw.githubusercontent.com/github/gh-aw/main/optimize.md

The workflow run is at https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

The agent will follow the steps below to install `gh aw`, analyze the run, and apply cost-reducing changes.

---

## Step 1: Install GitHub Agentic Workflows CLI Extension

Check if `gh aw` is installed by running

```bash
gh aw version
```

If it is installed, run:

```bash
gh extension upgrade aw
```

to upgrade to latest. If it is not installed, run the installation script from the main branch of the gh-aw repository:

```bash
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash
```

**What this does**: Downloads and installs the gh-aw binary to `~/.local/share/gh/extensions/gh-aw/`

**Verify installation**:

```bash
gh aw version
```

You should see version information displayed. If you encounter an error, check that:

- GitHub CLI (`gh`) is installed and authenticated
- The installation script completed without errors
- `~/.local/share/gh/extensions` is in your PATH

## Step 2: Optimize Token Consumption

Follow carefully the instructions in the appropriate prompt file. Read ALL the instructions in the prompt file before taking any action.

Below, ROOT is the location where you found this file. For example,

- if this file is at `https://raw.githubusercontent.com/github/gh-aw/main/optimize.md` then the ROOT is `https://raw.githubusercontent.com/github/gh-aw/main`
- if this file is at `https://raw.githubusercontent.com/github/gh-aw/v0.35.1/optimize.md` then the ROOT is `https://raw.githubusercontent.com/github/gh-aw/v0.35.1`

**Prompt file**: `ROOT/.github/aw/optimize-agentic-workflow.md`

**Use cases**:

- "Why is this workflow consuming so many tokens?"
- "Reduce the AI credit usage for workflow X"
- "Optimize the run that hit the max-ai-credits guardrail"
- "This workflow exceeded max_turns — how do I make it more efficient?"

## Step 3: Apply Optimizations

After identifying the root cause:

1. Edit the workflow markdown file (`.github/workflows/<workflow-name>.md`)
2. Recompile the workflow:

```bash
gh aw compile <workflow-name>
```

3. Check for syntax errors or validation warnings.

## Step 4: Commit and Push Changes

Commit the changes, e.g.

```bash
git add .github/workflows/<workflow-name>.md .github/workflows/<workflow-name>.lock.yml
git commit -m "Optimize agentic workflow: <describe optimization>"
git push
```

If there is branch protection on the default branch, create a pull request instead and report the link to the pull request.

## Instructions

When a user interacts with you:

1. **Extract the run URL or workflow name** from the user's request
2. **Fetch and read the optimization prompt** from `ROOT/.github/aw/optimize-agentic-workflow.md`
3. **Follow the loaded prompt's instructions** exactly
4. **If uncertain**, ask clarifying questions

## Quick Reference

```bash
# Audit a specific workflow run (includes token usage)
gh aw audit <run-id> --json

# Diff two or more workflow runs to measure improvement
gh aw audit <base-run-id> <optimized-run-id>

# Download and analyze workflow logs
gh aw logs <workflow-name>

# Compile workflows after optimizing
gh aw compile <workflow-name>

# Show status of all workflows
gh aw status
```
