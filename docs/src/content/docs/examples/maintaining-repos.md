---
title: 'Example: Automated Repository Maintenance'
description: How to use repo-assist, safe-outputs, and integrity filtering to manage open-source repositories at scale — controlling what agents can do, filtering untrusted input, and debugging failures.
sidebar:
  order: 20
---

Open-source maintainers using GitHub Agentic Workflows (`gh-aw`) need to manage both cost and trust: anyone can open an issue or PR, but not every contributor should influence an AI agent or trigger GitHub side effects. Two complementary controls address this risk: **safe outputs**, which limit configured agent writes, and **integrity filtering**, which limits what content an agent can see.

Together they provide defense in depth: integrity filtering keeps untrusted content out of the agent context, while safe-outputs ensure the workflow can produce only authorized side-effects. This guide shows how to use [🌈 Repo Assist](https://github.com/githubnext/agentics/blob/main/docs/repo-assist.md) as the entry point for incoming work and how to configure both controls for safe scale.

:::note[Real-world impact]
A [study of 15 open-source repositories](https://github.com/githubnext/repo-assist-impact/blob/main/report.md) found this approach achieved a **9× median increase** in issue closure and PR merge velocity, reducing open issue counts in every repository. Projects that were largely dormant became actively maintained, with several reaching near-complete backlog clearance. Results hold across languages and project types.
:::

## Repo Assist as Your Triage Layer

[Repo Assist](https://github.com/githubnext/agentics/blob/main/docs/repo-assist.md) is a recurring agentic workflow that selects maintenance tasks based on the repository's current backlog. It can label and investigate issues, implement focused fixes, improve tests and documentation, maintain pull requests, and summarize its activity for maintainers. You can also invoke it on demand with `/repo-assist <instructions>`.

Add Repo Assist to a repository with the interactive setup:

```bash
gh aw add-wizard githubnext/agentics/repo-assist
```

Use Repo Assist as a starting point for AI-assisted project maintenance, then review its proposed changes and adjust its schedule and permissions to match the project's maintainer capacity.

## Controlling Workflow Outputs with Safe-Outputs

Safe outputs are the primary mechanism for controlling writes requested by the agent job. Each permitted GitHub side effect — labeling an issue, posting a comment, opening a pull request, or merging — must be explicitly declared in the `safe-outputs:` block. If an agent requests an undeclared safe output, the runtime blocks it before it reaches the API.

Custom jobs and explicitly granted write permissions are separate trust boundaries and are not constrained by the safe-output list. Prefer safe outputs for agent-requested writes, and review any custom write path independently.

The available safe-outputs map directly to GitHub actions:

| Safe-output | What it allows |
| ------------ | --------------- |
| `add-labels` | Apply configured labels to an issue or pull request |
| `add-comment` | Post a comment on an issue, pull request, or discussion |
| `create-pull-request` | Open a new pull request |
| `merge-pull-request` | Merge a pull request (experimental) |
| `close-issue` | Close an issue |
| `create-issue` | Open a new issue |
| `assign-to-user` | Assign an issue to a user |

## Controlling Workflow Inputs with Integrity Filtering

Integrity filtering controls what content the agent sees. It evaluates the author of each issue, PR, or comment and removes items that do not meet the configured trust threshold before the agent context is assembled. Public repositories default to `min-integrity: approved`; repo-assist overrides that to `unapproved` so it can still see issues from contributors and first-time contributors.

The four configurable levels, from most to least restrictive, are:

| Level | Who qualifies |
| ------- | -------------- |
| `merged` | PRs merged into the default branch; commits reachable from main |
| `approved` | Owners, members, collaborators; non-fork PRs on public repos; recognized bots (`dependabot`, `github-actions`) |
| `unapproved` | Contributors who have had a PR merged before; first-time contributors |
| `none` | All content including users with no prior relationship |

Choose the level based on the workflow's role: use `unapproved` for repo-assist and other triage workflows that classify contributor input without acting on it, `approved` or `merged` for code-modifying workflows that open PRs or apply patches, and `none` for spam detection or analytics workflows that need full visibility but produce no direct GitHub mutations.

### Reactions as Trust Signals

Maintainers can use GitHub reactions (👍, ❤️) to promote content past the integrity filter without modifying labels. This is useful in repo-assist workflows where a maintainer wants to fast-track an external contribution.

To enable reactions, add the `integrity-reactions` feature flag:

```aw wrap
features:
  integrity-reactions: true
tools:
  github:
    min-integrity: approved
```

The compiler handles the rest. With `integrity-reactions: true`, it selects CLI GitHub access through the host policy proxy, sets `THUMBS_UP` and `HEART` as the default endorsement reactions, sets `THUMBS_DOWN` and `CONFUSED` as the default disapproval reactions, uses `endorser-min-integrity: approved`, and uses `disapproval-integrity: none`.

In practice, a 👍 or ❤️ from a trusted member promotes the item's integrity to `approved`, making it visible to agents that require that level. A 👎 or 😕 from a trusted member demotes the item to `none`.

See the [Integrity Filtering Reference](/gh-aw/reference/integrity/) for complete configuration details.

## Scaling Strategies

### Token Budget Awareness

Integrity filtering directly reduces token consumption because items filtered by the gateway never appear in the agent context. On a busy public repository, setting downstream agents to `min-integrity: approved` can cut context size substantially.

Use `gh aw logs --format markdown --count 20` to track token trends over time. The cross-run report highlights cost spikes, anomalous token usage, and per-run breakdowns so you can catch regressions early.

### Rate Limiting

The `user-rate-limit` frontmatter key caps how many times a workflow can run in a sliding window, preventing a flood of incoming issues from exhausting compute or inference budget:

```aw wrap
user-rate-limit:
  max-runs-per-window: 5
  window: 60
```

Match the run rate to your available review bandwidth. If the default cadence is too high, reduce frequency instead of disabling automation entirely. See [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) for full options.

### Pre-Activation Association Skips

For maintainer-operated moderation and triage workflows, you can skip runs early for specific event/author-association combinations using `on.skip-author-associations`:

```aw wrap
on:
  issue_comment:
    types: [created]
  skip-author-associations:
    issue_comment: [owner, member, collaborator]
```

This compiles into a pre-activation job-level `if` guard using event-specific payload fields such as `github.event.comment.author_association`, `github.event.issue.author_association`, and `github.event.pull_request.author_association`, so matching runs are skipped before agent execution starts.

### Concurrency Controls

Workflows automatically use dual concurrency control (per-workflow and per-engine). For repo-assist, you may want higher concurrency so multiple issues are triaged in parallel rather than queued:

```aw wrap
concurrency:
  max-parallel: 3
```

### Scoping Repository Access

`allowed-repos` prevents cross-repository reads that aren't necessary for the workflow's task:

```aw wrap
tools:
  github:
    allowed-repos: "myorg/*"
    min-integrity: approved
```

This is useful in monorepo or multi-repo setups where the agent should only read from the organization's own repos.

## Debugging Failed Workflows

### Quick Start: AI-Assisted Debugging

The fastest path to a root cause is to hand the failing run URL to the Copilot CLI:

```bash
copilot
```

Inside the CLI:

```text
agentic-workflows

Debug this run: https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

The agent loads the `debug-agentic-workflow` prompt, audits the run, and explains what went wrong. Follow up with specific questions about blocked domains, missing tools, or safe-output failures.

On GitHub.com with [agentic authoring configured](/gh-aw/guides/agentic-authoring/):

```text
agentic-workflows debug https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

### Manual Debugging with CLI Commands

**Audit a specific run:**

```bash
gh aw audit RUN_ID
gh aw audit RUN_ID --json    # machine-readable output
gh aw audit RUN_ID --parse   # writes log.md and firewall.md
```

The audit report covers the failure summary, tool usage, MCP server health, firewall analysis, token metrics, and missing tools.

**Analyze logs across multiple runs:**

```bash
gh aw logs my-workflow
gh aw logs my-workflow --format markdown --count 10
gh aw logs --filtered-integrity    # only runs with DIFC-filtered events
```

**Compare two runs for regressions:**

```bash
gh aw audit BASELINE_ID CURRENT_ID
```

### Common Failure Patterns

| Failure | Symptom / Cause | Fixes |
| --------- | ----------------- | ------- |
| **Missing tool calls** | Tool not configured or wrong name. Check `missing_tools` in audit. | Add to `tools:` in frontmatter; fix any `safeoutputs-` prefix; check MCP connectivity. |
| **Authentication failures** | Token permissions too narrow or API key missing. | Review `permissions:` block; ensure secrets are set; see [Auth Reference](/gh-aw/reference/auth/). |
| **Integrity filtering blocking content** | Author's association below `min-integrity`. `DIFC_FILTERED` events in audit show details. | Adjust `min-integrity`; add author to `trusted-users`; use `approval-labels`; check `gh aw logs --filtered-integrity`. |
| **Safe-output validation failures** | Agent attempted undeclared GitHub action. Safe-outputs blocks anything not listed. | Review `safe-outputs:`; check `safe_outputs.jsonl` in audit artifacts; see [Safe Outputs Reference](/gh-aw/reference/safe-outputs/). |
| **Token budget exhaustion** | Run hit token limit before completing. | Raise `min-integrity` to reduce context; add `cache-memory:`; simplify prompt; tighten `user-rate-limit`. |
| **Network blocks** | Required domain blocked by firewall. | Check firewall section of audit; add domain to `network.allowed`; see [Network Configuration Guide](/gh-aw/guides/network-configuration/). |

### Iterative Debug Workflow

1. Check the workflow run summary in the GitHub Actions UI.
2. Run `gh aw audit RUN_ID` for a structured breakdown.
3. For complex issues, use the `agentic-workflows` skill in Copilot Chat.
4. Edit the `.md` file → run `gh aw compile` to validate → trigger a new run.
5. Compare the new run against the baseline with `gh aw audit BASELINE_ID NEW_ID`.

## Related Documentation

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for output types and format requirements, [Integrity Filtering Reference](/gh-aw/reference/integrity/) for `min-integrity` and policy configuration, [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) for runaway-workflow prevention, [Cost Management](/gh-aw/reference/cost-management/) for token optimization, [Audit Commands](/gh-aw/reference/audit/) for `gh aw audit` and `gh aw logs`, [Debugging Workflows](/gh-aw/troubleshooting/debugging/) for detailed troubleshooting, [Network Configuration Guide](/gh-aw/guides/network-configuration/) for firewall setup, and [GitHub Tools Reference](/gh-aw/reference/github-tools/) for `tools.github` options.
