---
title: Integrity Filtering
description: How integrity filtering restricts agent access to GitHub content based on author trust and merge status, and how filtered events appear in logs.
sidebar:
  order: 680
---

Integrity filtering controls which GitHub content an agent can access during a workflow run. Rather than filtering by permissions, it filters by **trust**: the author association of an issue, pull request, or comment, and whether that content has been merged into the main branch.

## How It Works

The MCP gateway intercepts tool calls to GitHub and applies integrity checks to each piece of content returned. If an item's integrity level is below the configured minimum, the gateway discards it before the AI engine sees it. This happens transparently — the agent never receives filtered content, and filtered items are logged as `DIFC_FILTERED` events for later inspection.

## Configuration

Set `min-integrity` under `tools.github` in your workflow frontmatter:

```aw wrap
tools:
  github:
    min-integrity: approved
```

`min-integrity` can be specified alone. When `repos` is omitted, it defaults to `"all"`. If `repos` is also specified, both fields must be present.

```aw wrap
tools:
  github:
    repos: "myorg/*"
    min-integrity: approved
```

## Integrity Levels

| Level | What passes through |
|-------|---------------------|
| `merged` | Objects reachable from the main branch (any author) |
| `approved` | Objects authored by `OWNER`, `MEMBER`, or `COLLABORATOR` |
| `unapproved` | Objects authored by `CONTRIBUTOR` or `FIRST_TIME_CONTRIBUTOR` |
| `none` | All objects, including `FIRST_TIMER` and users with no association (`NONE`) |

Levels are ordered from most restrictive to least. Setting `min-integrity: approved` means only `approved` and `merged` content reaches the agent. `unapproved` and `none` content is filtered out.

**`merged`** is the strictest level. An item qualifies as `merged` when it is reachable from the default branch, regardless of who authored it. This is useful for workflows that should only act on production content.

**`approved`** corresponds to users who have a formal trust relationship with the repository: owners, members, and collaborators. This is the most common choice for public repository workflows.

**`unapproved`** includes contributors who have had code merged before, as well as first-time contributors. Appropriate when community participation is welcome and the workflow's outputs are reviewed before being applied.

**`none`** allows all content through. Use this deliberately, with appropriate safeguards, for workflows designed to process untrusted input — such as triage bots or spam detection.

## Default Behavior

For **public repositories**, if no `min-integrity` or `lockdown` is configured, the runtime automatically applies `min-integrity: approved`. This protects public workflows even when additional authentication has not been set up. See [Automatic Minimum-Integrity Protection](/gh-aw/reference/lockdown-mode/#automatic-minimum-integrity-protection) for details.

For **private and internal repositories**, no guard policy is applied automatically. Content from all users is accessible by default.

## Choosing a Level

The right level depends on who you want the agent to see content from:

- **Workflows that automate code review or apply changes**: `merged` or `approved` — only act on trusted content.
- **Workflows that respond to maintainers and trusted contributors**: `approved` — a common, safe default for most workflows.
- **Community triage or planning workflows**: `unapproved` — allow contributor input while excluding anonymous or first-time interactions.
- **Public-data workflows or spam detection**: `none` — see all activity, but ensure the workflow's outputs are not directly applied without review.

> [!WARNING]
> Setting `min-integrity: none` on a public repository disables the automatic protection. Only use it when the workflow is explicitly designed to handle untrusted input.

## Examples

**Allow only merged content:**

```aw wrap
tools:
  github:
    repos: "all"
    min-integrity: merged
```

**Trusted contributors only (typical for a public repository workflow):**

```aw wrap
tools:
  github:
    min-integrity: approved
```

**Allow all community contributions (for a triage workflow):**

```aw wrap
tools:
  github:
    min-integrity: unapproved
```

**Explicitly disable filtering on a public repository:**

```aw wrap
tools:
  github:
    min-integrity: none
```

**Scope to specific organizations with integrity filtering:**

```aw wrap
tools:
  github:
    mode: remote
    toolsets: [repos, issues, pull_requests]
    repos:
      - "myorg/*"
      - "partner/shared-repo"
    min-integrity: approved
```

## In Logs and Reports

When an item is filtered by the integrity check, the MCP gateway records a `DIFC_FILTERED` event in the run's `gateway.jsonl` log. Each event includes:

- **Server**: the MCP server that returned the filtered content
- **Tool**: the tool call that produced it (e.g., `list_issues`, `get_pull_request`)
- **User**: the login of the content's author
- **Reason**: a description such as `"Resource has lower integrity than agent requires."`
- **Integrity tags**: the tags assigned to the item that caused it to be filtered
- **Author association**: the GitHub author association (`CONTRIBUTOR`, `FIRST_TIMER`, etc.)

When gateway metrics are displayed, filtered events appear in a **DIFC Filtered Events** table alongside the standard server usage table:

```text
┌─────────────────────────────────────────────────────────────────────────────────────┐
│ DIFC Filtered Events                                                                │
├────────────────┬───────────────┬───────────────┬─────────────────────────────────-─┤
│ Server         │ Tool          │ User          │ Reason                             │
├────────────────┼───────────────┼───────────────┼────────────────────────────────────┤
│ github         │ list_issues   │ new-user      │ Resource has lower integrity than  │
│                │               │               │ agent requires.                    │
└────────────────┴───────────────┴───────────────┴────────────────────────────────────┘
```

The `Total DIFC Filtered` count in the summary line shows how many items were suppressed during the run.

### Filtering Logs by Integrity Events

To download only runs that had integrity-filtered content, use the `--filtered-integrity` flag with the `logs` command:

```bash
gh aw logs --filtered-integrity
```

This is useful when investigating whether your `min-integrity` configuration is filtering expected content or when tuning the level after observing real traffic patterns.

## Relationship to Lockdown Mode

[Lockdown Mode](/gh-aw/reference/lockdown-mode/) is a separate but related feature of the GitHub MCP server. It filters content to users with push access. The two features can coexist:

- `lockdown: true` filters using the GitHub MCP server's own content filter (based on push access).
- `min-integrity` filters using the MCP gateway's DIFC mechanism (based on author association and merge status).

For most public repository workflows, `min-integrity: approved` provides equivalent protection to lockdown mode without requiring additional authentication.

## Related Documentation

- [GitHub Tools Reference](/gh-aw/reference/github-tools/) — Full `tools.github` configuration including `repos` and guard policies
- [Lockdown Mode](/gh-aw/reference/lockdown-mode/) — GitHub MCP server content filtering for public repositories
- [MCP Gateway](/gh-aw/reference/mcp-gateway/) — Gateway architecture and log format
- [CLI Reference: logs](/gh-aw/setup/cli/#logs) — Downloading and analyzing workflow run logs
