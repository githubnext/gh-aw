---
title: Upgrading Agentic Workflows
description: Step-by-step guide to upgrade your repository to the latest version of agentic workflows, including updating extensions, applying codemods, compiling workflows, and validating changes.
sidebar:
  order: 100
---

This guide walks you through upgrading agentic workflows. `gh aw upgrade` handles the full process: updating the dispatcher agent file, migrating deprecated workflow syntax, and recompiling all workflows.

> [!TIP]
> Quick Upgrade
>
> For most users, upgrading is a single command:
>
> ```bash wrap
> gh aw upgrade
> ```
>
> This updates agent files, applies codemods, and compiles all workflows.

## Prerequisites

Before upgrading, ensure you have GitHub CLI (`gh`) v2.0.0+, the latest gh-aw extension, and a clean working directory in your Git repository. Verify with `gh --version`, `gh extension list | grep gh-aw`, and `git status`.

Create a backup branch before upgrading so you can recover if something goes wrong:

```bash wrap
git checkout -b backup-before-upgrade
git checkout -  # return to your previous branch
```

## Step 1: Upgrade the Extension

Upgrade the `gh aw` extension to get the latest features and codemods:

```bash wrap
gh extension upgrade gh-aw
```

Check your version with `gh aw version` and compare against the [latest release](https://github.com/github/gh-aw/releases). If you encounter issues, try a clean reinstall with `gh extension remove gh-aw` followed by `gh extension install github/gh-aw`.

## Step 2: Run the Upgrade Command

Run the upgrade command from your repository root:

```bash wrap
gh aw upgrade
```

This command performs three main operations:

### 2.1 Updates Dispatcher Agent File

Updates `.github/agents/agentic-workflows.agent.md` to the latest template. Workflow prompt files (`.github/aw/*.md`) are resolved directly from GitHub by the agent — they're no longer managed by the CLI.

### 2.2 Applies Codemods to All Workflows

The upgrade automatically applies codemods to fix deprecated fields in all workflow files (`.github/workflows/*.md`).

### 2.3 Compiles All Workflows

The upgrade automatically compiles all workflows to generate or update `.lock.yml` files, ensuring they're ready to run in GitHub Actions.

### Command Options

```bash wrap
gh aw upgrade                       # updates agent files + codemods + compiles
gh aw upgrade -v                    # verbose output
gh aw upgrade --no-fix              # skip codemods and compilation
gh aw upgrade --dir custom/workflows
```

## Step 3: Review the Changes

Run `git diff .github/workflows/` to verify the changes. Typical migrations include `sandbox: false` → `sandbox.agent: false`, `daily at` → `daily around`, and removal of deprecated `network.firewall` and `mcp-scripts.mode` fields.

## Step 4: Commit and Push

Stage and commit your changes:

```bash wrap
git add .github/workflows/ .github/agents/
git commit -m "Upgrade agentic workflows to latest version"
git push origin main
```

Always commit both `.md` and `.lock.yml` files together.

## Troubleshooting

**Extension upgrade fails:** Try a clean reinstall with `gh extension remove gh-aw && gh extension install github/gh-aw`.

**Codemods not applied:** Manually apply with `gh aw fix --write -v`.

**Compilation errors:** Review errors with `gh aw compile my-workflow --validate` and fix YAML syntax in source files.

**Workflows not running:** Verify `.lock.yml` files are committed, check status with `gh aw status`, and confirm secrets are valid with `gh aw secrets bootstrap`.

**Breaking changes:** Revert with `git checkout backup-before-upgrade` and review [release notes](https://github.com/github/gh-aw/releases).

## Advanced Topics

**Upgrading across versions:** Review the [changelog](https://github.com/github/gh-aw/blob/main/CHANGELOG.md) for cumulative changes when upgrading across multiple releases.

See the [troubleshooting guide](/gh-aw/troubleshooting/common-issues/) if you run into issues.
