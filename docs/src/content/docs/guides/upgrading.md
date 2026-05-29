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
gh aw upgrade --pre-releases    # Also consider newer pre-releases
```

Check your version with `gh aw version` and compare against the [latest release](https://github.com/github/gh-aw/releases). By default, `gh aw upgrade` follows stable releases; use `--pre-releases` to opt into pre-release builds, which are installed by exact tag. If you encounter issues, try a clean reinstall with `gh extension remove gh-aw` followed by `gh extension install github/gh-aw`.

## Step 2: Run the Upgrade Command

Run the upgrade command from your repository root:

```bash wrap
gh aw upgrade
```

This updates `.github/skills/agentic-workflows/SKILL.md` to the latest template, applies codemods to fix deprecated fields in all workflow files (`.github/workflows/*.md`), and recompiles all workflows to regenerate `.lock.yml` files. Use `--no-fix` to skip codemods and compilation, `-v` for verbose output, or `--dir` to target a custom workflows directory.

## Step 3: Review the Changes

Run `git diff .github/workflows/` to verify the changes. Typical migrations include `sandbox: false` → `sandbox.agent: false`, `app:` → `github-app:`, `safe-inputs:` → `mcp-scripts:`, `daily at` → `daily around`, and removal of deprecated `network.firewall` and `mcp-scripts.mode` fields.

## Step 4: Commit and Push

Stage and commit your changes:

```bash wrap
git add .github/workflows/ .github/skills/
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
