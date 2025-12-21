# Agents Command - UI for Managing Agentic Workflows

The `gh aw agents` command provides an easy-to-use interface for managing agentic workflows in your repository, similar to a package manager for workflows.

## Overview

The agents command group offers a comprehensive set of tools for discovering, installing, updating, and removing agentic workflows from repositories like [githubnext/agentics](https://github.com/githubnext/agentics).

## Commands

### `gh aw agents list`

List all installed agentic workflows with their metadata.

```bash
# List all installed agents
gh aw agents list

# List with JSON output
gh aw agents list --json

# Filter by source repository
gh aw agents list --repo githubnext/agentics
```

**Output:**
```
┌─────────┬────────────────────────────────────────────────────────────┬────────┬──────────────────────────────────┬────────────┬───────┐
│Name     │Description                                                 │Category│Source                            │Status      │Trigger│
├─────────┼────────────────────────────────────────────────────────────┼────────┼──────────────────────────────────┼────────────┼───────┤
│ci-doctor│Monitor CI workflows and investigate failures automatically.│Analysis│githubnext/agentics/ci-doctor@main│not compiled│issues │
└─────────┴────────────────────────────────────────────────────────────┴────────┴──────────────────────────────────┴────────────┴───────┘
```

### `gh aw agents browse`

Launch an interactive browser to discover and install agents from a repository.

```bash
# Browse agentics repository (default)
gh aw agents browse

# Browse a specific repository
gh aw agents browse githubnext/agentics

# Browse with version
gh aw agents browse githubnext/agentics@v1.0.0
```

**Features:**
- Multi-select interface for installing multiple agents at once
- Categorized agent listings (Triage, Analysis, Research, Coding, etc.)
- Shows installation status for each agent
- Displays agent descriptions inline

### `gh aw agents install`

Install one or more agentic workflows.

```bash
# Interactive mode (launches agent browser)
gh aw agents install

# Install specific agents
gh aw agents install ci-doctor
gh aw agents install ci-doctor daily-plan

# Install from specific repository
gh aw agents install ci-doctor --repo githubnext/agentics

# Force overwrite existing files
gh aw agents install ci-doctor --force
```

### `gh aw agents uninstall`

Remove one or more installed workflows.

```bash
# Interactive mode (select from installed agents)
gh aw agents uninstall

# Uninstall specific agents
gh aw agents uninstall ci-doctor
gh aw agents uninstall ci-doctor daily-plan

# Keep orphaned include files
gh aw agents uninstall ci-doctor --keep-orphans
```

### `gh aw agents update`

Update installed agents to their latest versions.

```bash
# Interactive mode (shows available updates)
gh aw agents update

# Update all agents without prompting
gh aw agents update --all

# Update specific agents
gh aw agents update ci-doctor daily-plan

# Force update even if no changes detected
gh aw agents update ci-doctor --force
```

### `gh aw agents info`

Display detailed information about a specific agent.

```bash
# Show agent information
gh aw agents info ci-doctor

# Output as JSON
gh aw agents info ci-doctor --json
```

**Output:**
```
═══════════════════════════════════════════════════════
Agent: ci-doctor
═══════════════════════════════════════════════════════

Description: Monitor CI workflows and investigate failures automatically.

Category:    Analysis
Status:      enabled
Trigger:     issues
Source:      githubnext/agentics/ci-doctor@main
Safe Outputs: create-issue, add-comment
File Path:   .github/workflows/ci-doctor.md
Lock File:   .github/workflows/ci-doctor.lock.yml

═══════════════════════════════════════════════════════
```

## Agent Categories

Agents are automatically categorized based on their purpose:

- **Triage & Analysis**: Issue triage, CI monitoring, code scanning
- **Research & Planning**: Status reports, research summaries, planning workflows
- **Coding & Development**: Code fixes, dependency updates, documentation
- **Other**: Miscellaneous workflows

## Features

### Interactive Selection

When using `browse`, `install` (without arguments), or `uninstall` (without arguments), you'll see an interactive multi-select interface:

```
Select agents to install from githubnext/agentics
Use space to select, enter to confirm

[ ] [Triage & Analysis] ci-doctor - Monitor CI workflows and investigate failures
[ ] [Triage & Analysis] issue-triage - Triage issues and pull requests
[ ] [Research & Planning] weekly-research - Collect research updates and trends
[ ] [Research & Planning] daily-team-status - Assess repository activity
[x] [Coding & Development] daily-progress (installed) - Automated daily development
[ ] [Coding & Development] pr-fix - Analyze and fix failing CI checks
```

### Status Indicators

Agents can have the following statuses:

- **enabled**: Workflow is compiled and active
- **disabled**: Workflow exists but is disabled
- **not compiled**: Markdown file exists but not yet compiled to .lock.yml
- **unknown**: Unable to determine status

### Source Tracking

All installed agents track their source repository and version, making updates easy:

```
source: githubnext/agentics/ci-doctor@main
```

## Comparison with `add` Command

The `agents` command provides a higher-level interface compared to the lower-level `add` command:

| Feature | `agents` | `add` |
|---------|----------|-------|
| Interactive browsing | ✓ | ✗ |
| Category filtering | ✓ | ✗ |
| Installation status | ✓ | ✗ |
| Update management | ✓ | ✗ |
| Agent metadata display | ✓ | ✗ |
| Multi-select UI | ✓ | ✗ |
| Uninstall support | ✓ | via `remove` |
| Direct workflow specs | ✗ | ✓ |
| Wildcard support | ✗ | ✓ |
| Custom naming | ✗ | ✓ |

**When to use `agents`:**
- You want a user-friendly interface for managing workflows
- You're exploring available workflows from agentics
- You need to update or remove installed workflows
- You want to see what's installed at a glance

**When to use `add`:**
- You know the exact workflow specification
- You need advanced features (wildcards, custom names, subdirectories)
- You're scripting workflow installation
- You're working with workflows from custom repositories

## Examples

### Install the CI Doctor agent

```bash
# Interactive way
gh aw agents install
# Select "ci-doctor" from the list

# Direct way
gh aw agents install ci-doctor
```

### Update all installed agents

```bash
gh aw agents update --all
```

### List agents from agentics repository

```bash
gh aw agents list --repo githubnext/agentics
```

### View detailed information about an agent

```bash
gh aw agents info ci-doctor
```

### Remove agents you no longer need

```bash
# Interactive selection
gh aw agents uninstall

# Direct removal
gh aw agents uninstall daily-plan weekly-research
```

## Integration with Existing Workflows

The `agents` command integrates seamlessly with existing `gh aw` commands:

1. **Install agents**: `gh aw agents install ci-doctor`
2. **Compile to Actions**: `gh aw compile` (or done automatically)
3. **Run the workflow**: `gh aw run ci-doctor`
4. **View logs**: `gh aw logs ci-doctor`
5. **Enable/disable**: `gh aw enable ci-doctor` / `gh aw disable ci-doctor`
6. **Update agent**: `gh aw agents update ci-doctor`

## See Also

- [`gh aw add`](../README.md#gh-aw-add) - Low-level workflow installation
- [`gh aw status`](../README.md#gh-aw-status) - View all workflow statuses
- [githubnext/agentics](https://github.com/githubnext/agentics) - Sample workflow repository
