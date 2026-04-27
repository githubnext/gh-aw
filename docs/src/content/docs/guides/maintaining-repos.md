---
title: Maintaining Repos with Agentic Workflows
description: How to use repo-assist, safe-outputs, and integrity filtering to manage an open-source repository at scale — controlling what agents can do, filtering untrusted input, and debugging failures.
sidebar:
  order: 20
---

Open-source maintainers face a unique challenge when running agentic workflows: anyone can open an issue or PR, triggering agent runs that consume compute and tokens — but not every contributor is equally trustworthy. gh-aw addresses this with two complementary safety mechanisms:

- **Safe-outputs** — The primary mechanism for controlling *what an agent can do*. Every GitHub mutation (opening issues, commenting, creating PRs) must be explicitly declared; anything not listed is blocked.
- **Integrity filtering** — The primary mechanism for controlling *what content the agent sees*. Content from untrusted authors is filtered from the agent's context before the run starts.

Together they form a defense-in-depth model: integrity filtering keeps untrusted content out of the agent's context, and safe-outputs ensure the agent can only produce authorized side-effects. This guide shows how to use [🌈 Repo Assist](https://github.com/githubnext/agentics/blob/main/docs/repo-assist.md) as the primary entry point for managing incoming work, and how to configure both mechanisms so your repository scales safely.

## Repo Assist as Your Triage Layer

[🌈 Repo Assist](https://github.com/githubnext/agentics/blob/main/docs/repo-assist.md) is a workflow that runs on every new issue or PR, classifies the content, and routes work to the right place. It is the recommended starting point for any public repository because it:

- Sees all incoming content (including from untrusted users), so nothing is silently ignored.
- Applies lightweight, low-cost classification (labels, comments) rather than heavy agent actions.
- Acts as a gate that downstream code-modifying agents depend on before they run.

## Controlling Workflow Outputs with Safe-Outputs

Safe-outputs is the primary mechanism for controlling what a workflow can do. Every action that produces a side-effect on GitHub — labeling an issue, posting a comment, opening a pull request, merging — must be explicitly declared in the `safe-outputs:` block. If an action isn't listed, the runtime blocks it before it reaches the API.

This is what makes it safe to run repo-assist with `min-integrity: unapproved`: even if the agent were to generate an instruction to open a PR or close an issue, the runtime would reject it because those outputs weren't declared.

The available safe-outputs map directly to GitHub actions:

| Safe-output | What it allows |
| ------------ | --------------- |
| `label-issue` | Apply or remove labels on an issue |
| `comment-issue` | Post a comment on an issue |
| `comment-pull-request` | Post a comment on a pull request |
| `create-pull-request` | Open a new pull request |
| `merge-pull-request` | Merge a pull request (experimental) |
| `close-issue` | Close an issue |
| `create-issue` | Open a new issue |
| `assign-issue` | Assign an issue to a user or team |

## Controlling Workflow Inputs with Integrity Filtering

Integrity filtering is the primary mechanism for controlling what content the agent sees. It evaluates the author of each issue, PR, or comment and removes items that don't meet the configured trust threshold — before the agent's context is assembled. Every public repository automatically applies `min-integrity: approved` as a baseline — repo-assist overrides this to `unapproved` so it can see issues from contributors and first-time contributors, not just trusted members.

The four configurable levels, from most to least restrictive:

| Level | Who qualifies |
| ------- | -------------- |
| `merged` | PRs merged into the default branch; commits reachable from main |
| `approved` | Owners, members, collaborators; non-fork PRs on public repos; recognized bots (`dependabot`, `github-actions`) |
| `unapproved` | Contributors who have had a PR merged before; first-time contributors |
| `none` | All content including users with no prior relationship |

Choose based on what the workflow does:

- **Repo-assist / triage workflows**: `unapproved` — classify content from contributors and first-time contributors without acting on it.
- **Code-modifying workflows** (open PRs, apply patches, close issues): `approved` or `merged` — only act on trusted input.
- **Spam detection or analytics**: `none` — see everything, but produce no direct GitHub mutations.

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

The compiler handles the rest — when `integrity-reactions: true` is set, it automatically:

- Enables the CLI proxy (`cli-proxy: true`), which is required for reaction-based integrity decisions
- Injects default endorsement reactions: `THUMBS_UP`, `HEART`
- Injects default disapproval reactions: `THUMBS_DOWN`, `CONFUSED`
- Uses `endorser-min-integrity: approved` (only reactions from owners, members, and collaborators count)
- Uses `disapproval-integrity: none` (a disapproval reaction demotes content to `none`)

These defaults mean that when a trusted member (owner, member, or collaborator) adds a 👍 or ❤️ reaction to an issue or comment, the item's integrity is promoted to `approved` — making it visible to agents using `min-integrity: approved`. Conversely, a 👎 or 😕 reaction from a trusted member demotes the item to `none`.

See the [Integrity Filtering Reference](/gh-aw/reference/integrity/) for complete configuration details.

## Scaling Strategies

### Token Budget Awareness

Integrity filtering directly reduces token consumption: items filtered by the gateway never appear in the agent's context window. On a busy public repository, `min-integrity: approved` on downstream agents can reduce context size dramatically compared to seeing all activity.

Use `gh aw logs --format markdown --count 20` to track token trends over time. The cross-run report surfaces cost spikes, anomalous token usage, and per-run breakdowns so you can detect regressions before they accumulate.

### Rate Limiting

The `rate-limit` frontmatter key caps how many times a workflow can run in a sliding window, preventing a flood of incoming issues from exhausting compute or inference budget:

```aw wrap
rate-limit:
  max-runs: 5
  window: 1h
```

See [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) for full options.

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
/agent agentic-workflows

Debug this run: https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

The agent loads the `debug-agentic-workflow` prompt, audits the run, and explains what went wrong. Follow up with specific questions about blocked domains, missing tools, or safe-output failures.

On GitHub.com with [agentic authoring configured](/gh-aw/guides/agentic-authoring/):

```text
/agent agentic-workflows debug https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

### Manual Debugging with CLI Commands

**Audit a specific run:**

```bash
gh aw audit RUN_ID
gh aw audit RUN_ID --json    # machine-readable output
gh aw audit RUN_ID --parse   # writes log.md and firewall.md
```

The audit report covers: failure summary, tool usage, MCP server health, firewall analysis, token metrics, and missing tools.

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
| **Token budget exhaustion** | Run hit token limit before completing. | Raise `min-integrity` to reduce context; add `cache-memory:`; simplify prompt; tighten `rate-limit`. |
| **Network blocks** | Required domain blocked by firewall. | Check firewall section of audit; add domain to `network.allowed`; see [Network Configuration Guide](/gh-aw/guides/network-configuration/). |

### Iterative Debug Workflow

1. Check the workflow run summary in the GitHub Actions UI.
2. Run `gh aw audit RUN_ID` for a structured breakdown.
3. For complex issues, use `/agent agentic-workflows` in Copilot Chat.
4. Edit the `.md` file → run `gh aw compile` to validate → trigger a new run.
5. Compare the new run against the baseline with `gh aw audit BASELINE_ID NEW_ID`.

## Related Documentation

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) — Complete output type documentation and format requirements
- [Integrity Filtering Reference](/gh-aw/reference/integrity/) — Complete `min-integrity` and policy configuration
- [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) — Preventing runaway workflows
- [Cost Management](/gh-aw/reference/cost-management/) — Token budget tracking and optimization
- [Audit Commands](/gh-aw/reference/audit/) — `gh aw audit` and `gh aw logs` reference
- [Debugging Workflows](/gh-aw/troubleshooting/debugging/) — Detailed debugging procedures
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) — Firewall and domain setup
- [GitHub Tools Reference](/gh-aw/reference/github-tools/) — Full `tools.github` options
