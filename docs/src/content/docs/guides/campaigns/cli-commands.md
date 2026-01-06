---
title: "CLI Commands"
description: "Command reference for managing agentic campaigns with gh aw"
---

The GitHub Agentic Workflows CLI provides commands for inspecting, validating, and managing agentic campaigns.

## Campaign Commands

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

gh aw campaign improvements            # Analyze improvement implementation status
gh aw campaign improvements -v         # Show evidence for each improvement
gh aw campaign improvements --json     # JSON improvements analysis
```

## List Campaigns

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

## Check Campaign Status

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

## Create New Campaign

Scaffold a new agentic campaign spec file interactively:

```bash
gh aw campaign new my-campaign-id
```

This creates `.github/workflows/my-campaign-id.campaign.md` with a basic structure.

## Validate Campaigns

Validate all agentic campaign specs:

```bash
gh aw campaign validate
```

By default, validation fails if problems are found. For non-failing validation (useful in CI while you iterate):

```bash
gh aw campaign validate --no-strict
```

## Analyze Campaign Improvements

Analyze which campaign improvements from the [improvements guide](/gh-aw/guides/campaigns/improvements/) have been implemented:

```bash
gh aw campaign improvements
```

This command checks for implementation of five key improvements:

1. **Summarized Campaign Reports** (High Priority) - Generate human-readable progress summaries with aggregated metrics and Epic issue updates
2. **Campaign Learning System** (Medium Priority) - Capture and share learnings across runs and between campaigns
3. **Enhanced Metrics Integration** (High Priority) - Enable orchestrators to read and act on historical metrics for decision-making
4. **Campaign Retrospectives** (Medium Priority) - Add campaign completion workflow with retrospective reports
5. **Cross-Campaign Analytics** (Low Priority) - Aggregate metrics across campaigns for portfolio-level visibility

Each improvement is classified as:
- **Implemented** - Feature is fully implemented
- **Partial** - Some components exist but feature is incomplete
- **Not Implemented** - Feature has not been started

Show detailed evidence for each improvement status:

```bash
gh aw campaign improvements -v
```

Get machine-readable JSON output:

```bash
gh aw campaign improvements --json
```

The analysis examines the codebase for:
- Presence of specific files (e.g., `pkg/campaign/report.go`, `pkg/campaign/learning.go`)
- Code patterns and functionality (e.g., metrics fetching, adaptive logic)
- Campaign configuration (e.g., governance policies, multiple campaigns)
- Repository artifacts (e.g., learnings in repo-memory)

Use this command to:
- Track progress on implementing campaign improvements
- Identify which features are partially implemented and need completion
- Prioritize future development based on current implementation status
- Generate reports on campaign system maturity

## Compilation and Orchestrators

**Agentic campaign specs and orchestrators:** When agentic campaign spec files exist under `.github/workflows/*.campaign.md`, `gh aw compile` validates those specs (including referenced `workflows`) and fails if problems are found. By default, `compile` also synthesizes an orchestrator workflow for each valid spec that has meaningful details and compiles it to a corresponding `.campaign.lock.yml` file. Orchestrators are only generated when the agentic campaign spec includes tracker labels, workflows, memory paths, or a metrics glob.

During compilation, a `.campaign.g.md` file is generated locally as a debug artifact to help developers review the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

See the [compile command documentation](/gh-aw/setup/cli/#compile) for details.

## Alternative: GitHub Issue Forms

For a low-code/no-code method, use the "🚀 Start an Agentic Campaign" issue form in the GitHub UI. The form captures campaign intent with structured fields and can trigger an agent to scaffold the spec file automatically. 

See the [Getting Started guide](/gh-aw/guides/campaigns/getting-started/#start-an-agentic-campaign-with-github-issue-forms) for details.
