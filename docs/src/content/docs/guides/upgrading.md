---
title: Upgrading Agentic Workflows
description: Step-by-step guide to upgrade your repository to the latest version of agentic workflows, including updating extensions, applying codemods, and validating changes.
sidebar:
  order: 100
---

This guide walks you through upgrading your agentic workflows to the latest version, ensuring you have access to the newest features, improvements, and security fixes.

## Overview

The upgrade process updates two key areas:

1. **Agent and prompt files** — GitHub Copilot instructions, dispatcher agent, and workflow creation prompts
2. **Workflow syntax** — Automatically migrates deprecated fields and applies the latest configuration patterns

> [!TIP]
> Quick Upgrade
>
> For most users, upgrading is a single command:
> ```bash wrap
> gh aw upgrade
> ```
> This updates agent files and applies codemods to all workflows.

## Prerequisites

Before upgrading, ensure you have:

- ✅ **GitHub CLI** (`gh`) v2.0.0 or higher — Check with `gh --version`
- ✅ **Latest gh-aw extension** — Upgrade with `gh extension upgrade gh-aw`
- ✅ **Git repository** with agentic workflows initialized (`.github/workflows/*.md` files exist)
- ✅ **Write access** to the repository to commit changes
- ✅ **Clean working directory** — Commit or stash any uncommitted changes

**Verify your setup:**

```bash wrap
gh --version                      # Should show 2.0.0+
gh extension list | grep gh-aw    # Should show gh-aw extension
git status                        # Should show "working tree clean"
```

## Step 1: Upgrade the Extension

Before upgrading workflows, ensure you have the latest version of the `gh-aw` extension:

```bash wrap
gh extension upgrade gh-aw
```

This downloads and installs the latest release with new features, bug fixes, and updated codemods.

> [!NOTE]
> Check Current Version
>
> Run `gh aw version` to see your current version.
> Compare against the [latest release](https://github.com/githubnext/gh-aw/releases) to confirm you're up to date.

### Alternative: Clean Reinstall

If you encounter issues with the upgrade, try a clean reinstall:

```bash wrap
gh extension remove gh-aw
gh extension install githubnext/gh-aw
gh aw version    # Verify installation
```

## Step 2: Backup Your Workflows

Before making changes, create a backup of your workflows to easily revert if needed:

```bash wrap
# Create a backup branch
git checkout -b backup-before-upgrade

# Or create a backup directory
cp -r .github/workflows .github/workflows.backup
```

Alternatively, ensure your latest changes are committed and pushed to a remote branch:

```bash wrap
git add .
git commit -m "Pre-upgrade snapshot"
git push origin main
```

> [!TIP]
> Git History
>
> Since workflows are tracked in Git, you can always view or revert changes using:
> ```bash wrap
> git diff HEAD~1 .github/workflows/    # See what changed
> git checkout HEAD~1 -- .github/workflows/my-workflow.md  # Revert single file
> ```

## Step 3: Run the Upgrade Command

Run the upgrade command from your repository root:

```bash wrap
gh aw upgrade
```

This command performs two main operations:

### 3.1 Updates Agent and Prompt Files

The upgrade updates these files to the latest templates (similar to running `gh aw init`):

- `.github/aw/github-agentic-workflows.md` — GitHub Copilot custom instructions
- `.github/agents/agentic-workflows.agent.md` — Dispatcher agent for routing tasks
- `.github/aw/create-agentic-workflow.md` — Prompt for creating new workflows
- `.github/aw/update-agentic-workflow.md` — Prompt for updating existing workflows
- `.github/aw/create-shared-agentic-workflow.md` — Prompt for shared workflows
- `.github/aw/debug-agentic-workflow.md` — Prompt for debugging workflows
- `.github/aw/upgrade-agentic-workflows.md` — Prompt for upgrade guidance

### 3.2 Applies Codemods to All Workflows

The upgrade automatically applies codemods to fix deprecated fields in all workflow files (`.github/workflows/*.md`):

| Codemod | What It Fixes | Example |
|---------|---------------|---------|
| **timeout-minutes-migration** | Replaces `timeout_minutes` with `timeout-minutes` | `timeout_minutes: 30` → `timeout-minutes: 30` |
| **network-firewall-migration** | Removes deprecated `network.firewall` field | Deletes `firewall: mandatory` |
| **sandbox-agent-false-removal** | Removes `sandbox.agent: false` (firewall now mandatory) | Deletes `agent: false` |
| **safe-inputs-mode-removal** | Removes deprecated `safe-inputs.mode` field | Deletes `mode: auto` |
| **schedule-at-to-around-migration** | Converts `daily at TIME` to `daily around TIME` | `daily at 10:00` → `daily around 10:00` |
| **delete-schema-file** | Deletes deprecated schema file | Removes `.github/aw/schemas/agentic-workflow.json` |
| **delete-old-agents** | Deletes old `.agent.md` files moved to `.github/aw/` | Removes outdated agent files |

**Example output:**

```text
Updating agent and prompt files...
✓ Updated agent and prompt files
Applying codemods to all workflows...
Processing workflow: daily-team-status
  ✓ Applied schedule-at-to-around-migration
  ✓ Applied timeout-minutes-migration
Processing workflow: issue-triage
  ✓ Applied safe-inputs-mode-removal
All workflows processed.

✓ Upgrade complete
```

### Command Options

```bash wrap
# Standard upgrade (updates agent files + applies codemods)
gh aw upgrade

# Verbose output (shows detailed progress)
gh aw upgrade -v

# Update agent files only (skip codemods)
gh aw upgrade --no-fix

# Upgrade workflows in custom directory
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

After upgrading, carefully review all changes before committing:

### Check Modified Files

```bash wrap
# View all modified files
git status

# See what changed in each file
git diff .github/workflows/
git diff .github/aw/
git diff .github/agents/
```

### Review Codemod Changes

Focus on workflow files (`.md`) to verify codemods applied correctly:

```bash wrap
# Review specific workflow changes
git diff .github/workflows/my-workflow.md

# Check all workflow changes
git diff .github/workflows/*.md
```

**What to verify:**

- ✅ Deprecated fields are removed or updated
- ✅ Formatting and comments are preserved
- ✅ No unintended changes to workflow logic
- ✅ YAML frontmatter syntax remains valid

### Common Changes to Expect

**Before upgrade:**
```yaml wrap
---
on:
  schedule: daily at 09:00
timeout_minutes: 45
network:
  firewall: mandatory
safe-inputs:
  mode: auto
---
```

**After upgrade:**
```yaml wrap
---
on:
  schedule: daily around 09:00
timeout-minutes: 45
network: defaults
safe-inputs:
  allowed-commands: ["gh", "git"]
---
```

> [!TIP]
> Detailed Comparison
>
> Use `git diff --word-diff` to see changes highlighted at the word level:
> ```bash wrap
> git diff --word-diff .github/workflows/my-workflow.md
> ```

## Step 5: Compile and Validate

Compile your workflows to ensure they're valid and generate updated `.lock.yml` files:

```bash wrap
# Compile all workflows
gh aw compile

# Compile with validation
gh aw compile --validate

# Compile specific workflow
gh aw compile my-workflow
```

**Expected output:**

```text
✓ Compiled daily-team-status
✓ Compiled issue-triage
✓ Compiled security-scanner
All workflows compiled successfully.
```

### Troubleshooting Compilation Errors

If compilation fails, the error message will indicate the issue:

```bash wrap
# See detailed validation errors
gh aw compile my-workflow --validate
```

**Common issues and fixes:**

| Error | Cause | Fix |
|-------|-------|-----|
| `Invalid frontmatter YAML` | Syntax error in YAML | Check YAML indentation and structure |
| `Unknown field: X` | Deprecated field not migrated | Run `gh aw fix my-workflow --write` |
| `Missing required field: Y` | Required configuration missing | Add missing field to frontmatter |
| `Invalid schedule syntax` | Incorrect schedule format | Use [schedule syntax](/gh-aw/reference/schedule-syntax/) |

> [!NOTE]
> Automatic Fixes
>
> If you still have deprecated fields after upgrading, manually apply codemods:
> ```bash wrap
> gh aw fix my-workflow --write    # Fix specific workflow
> gh aw fix --write                # Fix all workflows
> ```

## Step 6: Test Your Workflows

Before pushing changes, test your workflows to ensure they work correctly:

### Local Validation

```bash wrap
# Check workflow status
gh aw status

# Validate specific workflow
gh aw compile my-workflow --validate
```

### Test Workflow Execution

Trigger a workflow manually to verify it runs successfully:

```bash wrap
# Trigger workflow (requires workflow_dispatch)
gh aw run my-workflow

# Trigger and push changes in one command
gh aw run my-workflow --push
```

**Wait for completion:**

```bash wrap
# Monitor workflow status
gh aw status

# View execution logs
gh aw logs my-workflow
```

### Verify MCP Configuration (if using MCP servers)

If your workflows use MCP servers, verify the configuration:

```bash wrap
# List workflows with MCP servers
gh aw mcp list

# Inspect specific workflow's MCP configuration
gh aw mcp inspect my-workflow
```

> [!TIP]
> Test in Draft PR
>
> Create a draft pull request to test workflows in CI before merging:
> ```bash wrap
> git checkout -b upgrade-workflows
> git add .
> git commit -m "Upgrade workflows to latest version"
> git push origin upgrade-workflows
> gh pr create --draft --title "Upgrade workflows" --body "Testing upgraded workflows"
> ```

## Step 7: Commit and Push Changes

Once you've reviewed and tested the changes, commit them to your repository:

```bash wrap
# Stage all changes
git add .github/workflows/ .github/aw/ .github/agents/

# Create a descriptive commit message
git commit -m "Upgrade agentic workflows to latest version

- Updated agent and prompt files
- Applied codemods to migrate deprecated fields
- Recompiled all workflows"

# Push to remote repository
git push origin main
```

### Recommended Commit Structure

For better traceability, consider separate commits for different types of changes:

```bash wrap
# Commit 1: Agent file updates
git add .github/aw/ .github/agents/
git commit -m "Update agent and prompt files to latest templates"

# Commit 2: Workflow migrations
git add .github/workflows/*.md
git commit -m "Apply codemods to migrate deprecated workflow fields"

# Commit 3: Recompiled lock files
git add .github/workflows/*.lock.yml
git commit -m "Recompile workflows after upgrade"

# Push all commits
git push origin main
```

> [!CAUTION]
> Lock Files Must Be Committed
>
> Always commit both `.md` and `.lock.yml` files together. The `.lock.yml` files are the actual workflows GitHub Actions runs.
> Never add `.lock.yml` to `.gitignore`.

## Troubleshooting Common Issues

### Extension Upgrade Fails

**Symptom:** `gh extension upgrade gh-aw` fails with permission or network errors.

**Solutions:**

```bash wrap
# Try clean reinstall
gh extension remove gh-aw
gh extension install githubnext/gh-aw

# Or use standalone installer
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

### Codemods Not Applied

**Symptom:** Deprecated fields still present after running `gh aw upgrade`.

**Solution:**

```bash wrap
# List available codemods
gh aw fix --list-codemods

# Manually apply codemods with verbose output
gh aw fix --write -v

# Check specific workflow
gh aw fix my-workflow --write -v
```

### Compilation Errors After Upgrade

**Symptom:** `gh aw compile` fails with validation errors.

**Solution:**

```bash wrap
# See detailed error messages
gh aw compile my-workflow --validate

# Check for syntax errors
cat .github/workflows/my-workflow.md

# Verify YAML structure
head -20 .github/workflows/my-workflow.md
```

### Workflows Not Running After Upgrade

**Symptom:** Workflows don't trigger or execute after upgrading.

**Solutions:**

1. **Verify compilation:** Ensure `.lock.yml` files are up-to-date and committed
   ```bash wrap
   gh aw compile
   git status  # Check if .lock.yml files are modified
   ```

2. **Check workflow status:** Ensure workflows are enabled
   ```bash wrap
   gh aw status
   ```

3. **Verify secrets:** Confirm AI engine tokens are still valid
   ```bash wrap
   gh aw secrets bootstrap --engine copilot
   ```

4. **Review workflow logs:** Check for execution errors
   ```bash wrap
   gh aw logs my-workflow
   ```

### Breaking Changes or Unexpected Behavior

**Symptom:** Workflows behave differently after upgrade.

**Solution:**

```bash wrap
# Revert to backup
git checkout backup-before-upgrade

# Or revert specific files
git checkout HEAD~1 -- .github/workflows/my-workflow.md

# Recompile after reverting
gh aw compile
```

Then review [release notes](https://github.com/githubnext/gh-aw/releases) for breaking changes and migration guidance.

## Advanced Topics

### Upgrading Across Multiple Versions

If you're upgrading from an older version (e.g., v0.0.x to v0.2.x), review the [changelog](https://github.com/githubnext/gh-aw/blob/main/CHANGELOG.md) for all intermediate versions to understand cumulative changes.

### Custom Workflow Directory

For repositories using custom workflow directories:

```bash wrap
# Upgrade custom directory
gh aw upgrade --dir custom/workflows

# Compile custom directory
gh aw compile --dir custom/workflows
```

### Selective Codemod Application

To apply specific codemods only:

```bash wrap
# Check what would be changed (dry-run)
gh aw fix my-workflow

# Apply specific workflow
gh aw fix my-workflow --write

# Skip codemods during upgrade
gh aw upgrade --no-fix

# Then manually fix specific workflows
gh aw fix workflow-1 workflow-2 --write
```

### Upgrading in CI/CD Pipelines

Automate upgrades in your CI/CD pipeline:

```yaml wrap
name: Upgrade Agentic Workflows
on:
  schedule:
    - cron: '0 0 * * MON'  # Weekly on Monday
  workflow_dispatch:

jobs:
  upgrade:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install gh-aw
        run: gh extension install githubnext/gh-aw
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Upgrade workflows
        run: gh aw upgrade
      
      - name: Compile workflows
        run: gh aw compile
      
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v6
        with:
          commit-message: 'Automated upgrade of agentic workflows'
          title: 'Upgrade agentic workflows'
          branch: automated-upgrade
          labels: automation, upgrade
```

> [!WARNING]
> Review Automated Upgrades
>
> Always review automated upgrade PRs before merging to catch any unexpected changes or breaking updates.

## Best Practices

- ✅ **Upgrade regularly** — Stay current with latest features and security fixes
- ✅ **Review changes** — Always inspect diffs before committing
- ✅ **Test workflows** — Trigger manual runs to verify functionality
- ✅ **Read release notes** — Understand what changed in each version
- ✅ **Keep backups** — Use Git branches or tags for easy rollback
- ✅ **Update extension** — Upgrade `gh aw` before upgrading workflows

## What's Next?

After upgrading your workflows:

- **Learn about new features** — Check the [changelog](https://github.com/githubnext/gh-aw/blob/main/CHANGELOG.md) for what's new
- **Explore advanced configuration** — See the [frontmatter reference](/gh-aw/reference/frontmatter-full/)
- **Optimize workflow performance** — Review [best practices](/gh-aw/guides/deterministic-agentic-patterns/)
- **Add new workflows** — Browse the [agentics collection](https://github.com/githubnext/agentics)

Need help? Check the [troubleshooting guide](/gh-aw/troubleshooting/common-issues/) or [open an issue](https://github.com/githubnext/gh-aw/issues/new).
