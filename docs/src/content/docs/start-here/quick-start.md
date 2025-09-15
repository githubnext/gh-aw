---
title: Quick Start Guide
description: Get your first agentic workflow running in minutes. Install the extension, add a sample workflow, set up secrets, and run your first AI-powered automation.
sidebar:
  order: 2
---

This guide will get you from zero to a running agentic workflow in minutes. You'll install the extension, add a sample workflow, set up the required secrets, and run it.

## Prerequisites

- A repository you are a maintainer of, can push to (or a fork) and have permission to add Actions secrets.

- A Claude or OpenAI API key. 

## Step 1 — Install the extension

```bash
gh extension install githubnext/gh-aw
```

## Step 2 — Add a sample workflow, review and merge

The easiest way to get started is to add a sample from [The Agentics](https://github.com/githubnext/agentics) collection. From your repository root run:

```bash
gh aw add weekly-research -r githubnext/agentics --pr
```

This creates a pull request that adds `.github/workflows/weekly-research.md` and the compiled `.lock.yml`. Review and merge the PR into your repo.

## Step 3 — Add an AI secret

Agentic workflows use an AI engine. For Claude add this repository secret:

```bash
gh secret set ANTHROPIC_API_KEY -a actions --body "<your-anthropic-api-key>"
```

For Codex (experimental), add:

```bash
gh secret set OPENAI_API_KEY -a actions --body "<your-openai-api-key>"
```

These secrets are used by Actions at runtime.

## Step 4 — Trigger a run of the workflow in GitHub Actions

Trigger the workflow immediately in GitHub Actions:

```bash
gh aw run weekly-research
```

Download and inspect execution logs:

```bash
gh aw logs weekly-research
```

## 📝 Understanding Your First Workflow

Let's look at what you just added. The weekly research workflow automatically triages new issues:

```markdown
---
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM

safe-outputs:
  create-issue:
---

# Weekly Research Report

Create a weekly research report summarizing recent developments in our field:

1. Research recent developments and trends
2. Summarize key findings 
3. Create an issue with the research report
4. Tag relevant team members

Keep the report concise but informative.
```

This workflow:
- **Triggers** every Monday at 9 AM via cron schedule
- **Has permissions** to read repository content and write issues
- **Uses tools** to create GitHub issues
- **Runs AI instructions** in natural language to create research reports

## What's next?

Now that you have your first workflow running:

- **Customize the workflow** — Edit the `.md` file to fit your needs, then recompile with `gh aw compile`
- **Explore more samples** — Check out [The Agentics](https://github.com/githubnext/agentics) repository
- **Learn the concepts** — Read [Concepts](../start-here/concepts/) to understand how agentic workflows work
- **Read the docs** — See [Documentation](../)

You're ready to start automating with agentic workflows! ✨
