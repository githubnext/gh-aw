---
title: Upgrading Agentic Workflows
description: Step-by-step guide to upgrade your repository to the latest version of agentic workflows, including updating extensions, applying codemods, compiling workflows, and validating changes.
sidebar:
  order: 100
---

This guide walks you through upgrading agentic workflows. `gh aw upgrade` handles the full process: updating the dispatcher agent file, migrating deprecated workflow syntax, and recompiling all workflows.

> [!TIP]
> Agentic Upgrade
>
> You can start an agent session in your repository on GitHub.com and use the command `/agent agentic-workflows Upgrade` to automatically upgrade your workflows.

> [!TIP]
> Quick Upgrade
>
> For most users, upgrading is a single command:
> ```bash wrap
> gh aw upgrade
> ```
> This updates agent files, applies codemods, and compiles all workflows.

## Prerequisites

Before upgrading, ensure you have GitHub CLI (`gh`) v2.0.0+, the latest gh-aw extension, and a clean working directory in your Git repository. Verify with `gh --version`, `gh extension list | grep gh-aw`, and `git status`.

## Step 1: Upgrade the Extension

Upgrade the `gh aw` extension to get the latest features and codemods:

```bash wrap
gh extension upgrade gh-aw
```

> [!TIP]
> Working in GitHub Codespaces?
>
> If the extension upgrade fails due to restricted permissions that prevent global npm installs, use the standalone installer instead:
>
> ```bash wrap
> curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash
> ```
>

Check your version with `gh aw version` and compare against the [latest release](https://github.com/github/gh-aw/releases). If you encounter issues, try a clean reinstall with `gh extension remove gh-aw` followed by `gh extension install github/gh-aw`.

## Step 2: Backup Your Workflows

Create a backup branch (`git checkout -b backup-before-upgrade`) before upgrading. Workflows are tracked in Git, so you can always revert with `git checkout backup-before-upgrade`.

## Step 3: Run the Upgrade Command

Run the upgrade command from your repository root:

```bash wrap
gh aw upgrade
```

This command performs three main operations:

### 3.1 Updates Dispatcher Agent File

Updates `.github/agents/agentic-workflows.agent.md` to the latest template. Workflow prompt files (`.github/aw/*.md`) are resolved directly from GitHub by the agent — they're no longer managed by the CLI.

### 3.2 Applies Codemods to All Workflows

The upgrade automatically applies codemods to fix deprecated fields in all workflow files (`.github/workflows/*.md`):

| Codemod | What It Fixes | Example |
|---------|---------------|---------|
| **sandbox-false-to-agent-false** | Converts `sandbox: false` to `sandbox.agent: false` | `sandbox: false` → `sandbox: { agent: false }` |
| **network-firewall-migration** | Removes deprecated `network.firewall` field | Deletes `firewall: mandatory` |
| **mcp-scripts-mode-removal** | Removes deprecated `mcp-scripts.mode` field | Deletes `mode: auto` |
| **safe-inputs-to-mcp-scripts** | Renames `safe-inputs:` to `mcp-scripts:` | `safe-inputs:` → `mcp-scripts:` |
| **schedule-at-to-around-migration** | Converts `daily at TIME` to `daily around TIME` | `daily at 10:00` → `daily around 10:00` |
| **delete-schema-file** | Deletes deprecated schema file | Removes `.github/aw/schemas/agentic-workflow.json` |
| **delete-old-agents** | Deletes old `.agent.md` files moved to `.github/aw/` | Removes outdated agent files |

### 3.2.1 Manual Migrations (No Codemod)

Some breaking changes require manual edits because no automatic codemod was shipped:

| Change | What to do |
|--------|------------|
| `engine: id: opencode` removed (April 2026) | Change to `engine: id: crush`. The OpenCode engine was archived; Crush is its successor. Also update any `GH_AW_MODEL_AGENT_OPENCODE` org/repo variables to `GH_AW_MODEL_AGENT_CRUSH`. |

### 3.3 Compiles All Workflows

The upgrade automatically compiles all workflows to generate or update `.lock.yml` files, ensuring they're ready to run in GitHub Actions.

**Example output:**

```text
Updating agent file...
✓ Updated agent file
Applying codemods to all workflows...
Processing workflow: daily-team-status
  ✓ Applied schedule-at-to-around-migration
  ✓ Applied timeout-minutes-migration
Processing workflow: issue-triage
  ✓ Applied mcp-scripts-mode-removal
All workflows processed.
Compiling all workflows...
✓ Compiled 3 workflow(s)

✓ Upgrade complete
```

### Command Options

```bash wrap
gh aw upgrade                       # updates agent files + codemods + compiles
gh aw upgrade -v                    # verbose output
gh aw upgrade --no-fix              # skip codemods and compilation
gh aw upgrade --dir custom/workflows
```

> [!WARNING]
> Custom Workflow Directory
>
> If you're using a custom workflow directory (not `.github/workflows`), always specify it with `--dir`:
> ```bash wrap
> gh aw upgrade --dir path/to/workflows
> ```

## Step 4: Review the Changes

Run `git diff .github/workflows/` to verify the changes. Typical migrations include `sandbox: false` → `sandbox.agent: false`, `daily at` → `daily around`, and removal of deprecated `network.firewall` and `mcp-scripts.mode` fields.

## Step 5: Verify Compilation

The upgrade automatically compiles workflows. To validate specific workflows, run `gh aw compile my-workflow --validate`. Common issues include invalid YAML syntax, deprecated fields (fix with `gh aw fix --write`), or incorrect schedule format. See the [schedule syntax reference](/gh-aw/reference/schedule-syntax/) for details.

## Step 6: Review Lock Files

Verify that each `.md` workflow has a corresponding `.lock.yml` file with `git status | grep .lock.yml`. Never edit `.lock.yml` files directly-they're auto-generated. Always edit the `.md` source and recompile.

## Step 7: Test Your Workflows

Trigger a manual run with `gh aw run my-workflow` and monitor with `gh aw logs my-workflow`. Consider testing via a draft PR before merging to production.

## Step 8: Commit and Push

Stage and commit your changes:

```bash wrap
git add .github/workflows/ .github/agents/
git commit -m "Upgrade agentic workflows to latest version"
git push origin main
```

Always commit both `.md` and `.lock.yml` files together — never add `.lock.yml` to `.gitignore`.

## Troubleshooting

**Extension upgrade fails:** Try a clean reinstall with `gh extension remove gh-aw && gh extension install github/gh-aw`.

**Codemods not applied:** Manually apply with `gh aw fix --write -v`.

**Compilation errors:** Review errors with `gh aw compile my-workflow --validate` and fix YAML syntax in source files.

**Workflows not running:** Verify `.lock.yml` files are committed, check status with `gh aw status`, and confirm secrets are valid with `gh aw secrets bootstrap`.

**Breaking changes:** Revert with `git checkout backup-before-upgrade` and review [release notes](https://github.com/github/gh-aw/releases).

## Advanced Topics

**Upgrading across versions:** Review the [changelog](https://github.com/github/gh-aw/blob/main/CHANGELOG.md) for cumulative changes when upgrading across multiple releases.

**CI/CD automation:** Automate upgrades with a scheduled workflow that creates PRs. Always review automated upgrade PRs before merging.

See the [troubleshooting guide](/gh-aw/troubleshooting/common-issues/) if you run into issues.
