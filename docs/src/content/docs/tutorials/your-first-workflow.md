---
title: Your First Workflow
description: Create your first agentic workflow in 5 minutes. Build a simple issue greeter that welcomes new contributors with a friendly AI-generated message.
sidebar:
  order: 1
---

This tutorial will guide you through creating your first agentic workflow in 5 minutes. You'll build a simple bot that welcomes new contributors when they open their first issue.

## What You'll Build

An issue greeter workflow that automatically welcomes new contributors when they open their first issue in your repository.

## Prerequisites

A GitHub repository with Actions enabled where you have maintainer access.

## Steps

### Step 1: Create the Workflow File

Create a new file in your repository at `.github/workflows/issue-greeter.md`:

```aw wrap title=".github/workflows/issue-greeter.md"
---
on:
  issues:
    types: [opened]
safe-outputs:
  add-comment:
---

# Issue Greeter

Welcome new contributors with a friendly message.

When a new issue is opened:
1. Check if the issue author is a new contributor to the repository
2. If they are a new contributor, post a warm greeting that:
   - Thanks them for their first contribution
   - Lets them know the team will review it soon
   - Invites them to add any additional details if needed

Keep the tone friendly and professional.
```

### Step 2: Compile the Workflow

Compile the workflow to generate the GitHub Actions YAML file:

```bash
gh aw compile issue-greeter
```

This creates `.github/workflows/issue-greeter.lock.yml` which GitHub Actions will execute.

### Step 3: Commit and Push

Commit both files to your repository:

```bash
git add .github/workflows/issue-greeter.md .github/workflows/issue-greeter.lock.yml
git commit -m "Add issue greeter workflow"
git push
```

### Step 4: Add AI Secret

The workflow needs a GitHub Copilot CLI token to run. [Create a Fine-Grained Personal Access Token](https://github.com/settings/personal-access-tokens/new) with Copilot Requests permission (Read-only access), then add it as a repository secret:

```bash
gh secret set COPILOT_CLI_TOKEN -a actions --body "<your-token>"
```

### Step 5: Test Your Workflow

Create a new issue in your repository. Within a minute, the workflow will run and add a friendly welcome comment to your issue.

## What You Learned

- How to create an agentic workflow using markdown
- How to trigger workflows on GitHub events (issue creation)
- How to use safe-outputs for controlled GitHub interactions
- How to compile workflows into GitHub Actions YAML

## Next Steps

Now that you have your first workflow running:

- **Customize the greeting**: Edit the markdown content to change the message
- **Add more triggers**: Explore other events in the [frontmatter reference](/gh-aw/reference/frontmatter/)
- **Try different tools**: Learn about available tools in the [tools reference](/gh-aw/reference/tools/)
- **Explore samples**: Browse [IssueOps guide](/gh-aw/guides/issueops/) for more workflow ideas
