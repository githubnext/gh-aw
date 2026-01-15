---
title: CLI commands
description: Command reference for managing agentic campaigns with gh aw
---

The GitHub Agentic Workflows CLI provides commands for inspecting, validating, and managing agentic campaigns.

> [!TIP]
> For the fastest campaign setup, use the [automated creation flow](/gh-aw/guides/campaigns/getting-started/#automated-campaign-creation) by applying the `create-agentic-campaign` label to an issue.

## Campaign commands

From the root of the repo:

```bash
gh aw campaign                         # List all agentic campaigns
gh aw campaign security                # Filter by ID or name substring
gh aw campaign --json                  # JSON output

gh aw campaign status                  # Live status for all agentic campaigns
gh aw campaign status incident         # Filter by ID or name substring
gh aw campaign status --json           # JSON status output

gh aw campaign new my-campaign-id      # Scaffold a new agentic campaign spec
gh aw campaign validate                # Validate agentic campaign specs (fails on problems)
gh aw campaign validate --no-strict    # Report problems without failing
```

## List campaigns

Display all agentic campaigns defined in `.github/workflows/*.campaign.md`:

```bash
gh aw campaign
```

Filter by campaign ID or name:

```bash
gh aw campaign security
```

Get machine-readable JSON output:

```bash
gh aw campaign --json
```

## Check campaign status

View live status of all agentic campaigns with their associated project boards:

```bash
gh aw campaign status
```

Filter status by campaign ID or name:

```bash
gh aw campaign status incident
```

Get status in JSON format:

```bash
gh aw campaign status --json
```

## Create new campaign

Scaffold a new agentic campaign spec file interactively:

```bash
gh aw campaign new my-campaign-id
```

This creates `.github/workflows/my-campaign-id.campaign.md` with a basic structure.

## Validate campaigns

Validate all agentic campaign specs:

```bash
gh aw campaign validate
```

By default, validation fails if problems are found. For non-failing validation (useful in CI while you iterate):

```bash
gh aw campaign validate --no-strict
```

## Compilation and orchestrators

**Agentic campaign specs and orchestrators:** When agentic campaign spec files exist under `.github/workflows/*.campaign.md`, `gh aw compile` validates those specs (including referenced `workflows`) and fails if problems are found. By default, `compile` also synthesizes an orchestrator workflow for each valid spec that has meaningful details and compiles it to a corresponding `.campaign.lock.yml` file. Orchestrators are only generated when the agentic campaign spec includes tracker labels, workflows, memory paths, or a metrics glob.

> [!NOTE]
> During compilation, a `.campaign.g.md` file is generated locally as a debug artifact to help developers review the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

See the [compile command documentation](/gh-aw/setup/cli/#compile) for details.

## Alternative: automated campaign creation

For a streamlined, fully-automated method, create an issue and apply the `create-agentic-campaign` label. This triggers an optimized two-phase campaign creation flow:

**Phase 1 - Campaign Generator** (~30 seconds): Automatically creates a GitHub Project board, discovers relevant workflows from the local repository and [agentics collection](https://github.com/githubnext/agentics), generates the complete campaign specification, and updates the issue with details.

**Phase 2 - Compilation** (~1-2 minutes): Automatically assigns a Copilot Coding Agent to compile the campaign using `gh aw compile` and create a pull request with all campaign files.

Total time: 2-3 minutes (60% faster than the previous flow).

See the [Getting started guide](/gh-aw/guides/campaigns/getting-started/#automated-campaign-creation) for complete details on the automated creation process.
