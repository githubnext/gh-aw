---
title: CLI commands
description: Command reference for managing agentic campaigns with gh aw
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

The GitHub Agentic Workflows CLI provides commands for inspecting, validating, and managing agentic campaigns.

> [!IMPORTANT]
> **Use the automated creation flow to create campaigns.** The CLI commands below are for inspecting, validating, and managing existing campaigns. See the [Getting started guide](/gh-aw/guides/campaigns/getting-started/) for campaign creation.

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

## Most common tasks

- See what campaigns exist: `gh aw campaign`
- Check which ones look unhealthy: `gh aw campaign status`
- Validate specs (locally or in CI): `gh aw campaign validate`

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

## Create new campaign (advanced)

> [!WARNING]
> This command is for advanced use cases only. **Use the [automated creation flow](/gh-aw/guides/campaigns/getting-started/) instead.**

Scaffold a new agentic campaign spec file interactively:

```bash
gh aw campaign new my-campaign-id
```

This creates `.github/workflows/my-campaign-id.campaign.md` with a basic structure, but you'll still need to manually configure all fields and compile the campaign.

## Validate campaigns

Validate all agentic campaign specs:

```bash
gh aw campaign validate
```

By default, validation fails if problems are found. For non-failing validation (useful in CI while you iterate):

```bash
gh aw campaign validate --no-strict
```

<details>
<summary>Compilation details (advanced)</summary>

The automated campaign creation flow handles compilation for you.

If you’re working on a campaign spec manually, see the [compile command documentation](/gh-aw/setup/cli/#compile).
</details>
