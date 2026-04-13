---
title: Maintaining Repos with Agentic Workflows
description: How to manage a repository with agentic workflows at scale — filtering untrusted input, prioritizing trusted work, and debugging failures.
sidebar:
  order: 20
---

Running agentic workflows on a public or active repository introduces three practical challenges: managing the volume of incoming issues and PRs from unknown users, ensuring agents prioritize trusted work, and diagnosing failures efficiently. This guide ties together integrity filtering, repo-assist, and the debugging toolchain from a maintainer's perspective.

## Filtering Input with Integrity Levels

Every public repository workflow automatically applies `min-integrity: approved` as a baseline unless you override it. This means the agent only sees content from repository owners, members, collaborators, and non-fork PRs — filtering out first-time contributors and anonymous users by default.

The four configurable integrity levels, from most to least restrictive:

| Level | Who qualifies |
|-------|--------------|
| `merged` | PRs merged into the default branch; commits reachable from main |
| `approved` | Owners, members, collaborators; non-fork PRs on public repos; all content in private repos; recognized bots (`dependabot`, `github-actions`) |
| `unapproved` | Contributors who have had a PR merged before; first-time contributors |
| `none` | All content including users with no prior relationship |

Choose based on what the workflow does:

- **Code-modifying workflows** (open PRs, apply patches, close issues): use `approved` or `merged`. These workflows act on your behalf, so restrict them to trusted input.
- **Triage and labeling workflows**: use `unapproved`. These workflows classify without acting, so seeing community contributions is valuable.
- **Spam detection or analytics**: use `none`. These workflows process all input but produce no direct GitHub mutations.

```aw wrap
# Code review workflow — only trusted contributors
tools:
  github:
    min-integrity: approved
```

```aw wrap
# Triage workflow — include community contributors
tools:
  github:
    min-integrity: unapproved
```

> [!NOTE]
> Setting `min-integrity: none` on a public repository disables the automatic protection. Only use it when the workflow is designed to handle untrusted input safely.

### Fine-Grained Trust Controls

Beyond the global level, three per-item overrides let you handle edge cases:

**`trusted-users`** — Elevate specific accounts (contractors, partners, bots) to `approved` regardless of their GitHub author association:

```aw wrap
tools:
  github:
    min-integrity: approved
    trusted-users:
      - "contractor-alice"
      - "partner-org-bot"
```

**`approval-labels`** — Let a human reviewer label content to pass it through a stricter filter:

```aw wrap
tools:
  github:
    min-integrity: approved
    approval-labels:
      - "agent-approved"
      - "human-reviewed"
```

**`blocked-users`** — Unconditionally block known-bad accounts regardless of `min-integrity`:

```aw wrap
tools:
  github:
    min-integrity: approved
    blocked-users:
      - "known-spam-bot"
```

To manage these lists across multiple workflows without duplicating them, store them in GitHub repository or organization variables:

| Workflow field | GitHub variable |
|---------------|----------------|
| `blocked-users` | `GH_AW_GITHUB_BLOCKED_USERS` |
| `trusted-users` | `GH_AW_GITHUB_TRUSTED_USERS` |
| `approval-labels` | `GH_AW_GITHUB_APPROVAL_LABELS` |

The runtime automatically merges per-workflow values with the variable. Set these under **Settings → Secrets and variables → Actions → Variables**.

### Reactions as Trust Signals

Starting from MCPG v0.2.18, maintainers can use GitHub reactions (👍, ❤️) to dynamically promote or demote content integrity without modifying labels. This is an opt-in feature:

```aw wrap
features:
  integrity-reactions: true
mcp-gateway:
  version: "v0.2.18"
tools:
  github:
    min-integrity: approved
    endorsement-reactions:
      - "THUMBS_UP"
      - "HEART"
    disapproval-reactions:
      - "THUMBS_DOWN"
      - "CONFUSED"
    endorser-min-integrity: approved
    disapproval-integrity: none
```

When a trusted member (at or above `endorser-min-integrity`) adds an endorsement reaction to an issue or comment, the item's integrity is promoted to `approved`. A disapproval reaction demotes it to the level set by `disapproval-integrity`.

> [!IMPORTANT]
> Reactions only work when running through the MCPG proxy mode. They are not available in gateway mode.

See the [Integrity Filtering Reference](/gh-aw/reference/integrity/) for complete configuration details.

## Triage Layer with Repo-Assist

For repositories receiving high volumes of issues and PRs, a triage workflow acts as a router: it classifies incoming content and decides which specialized agent (if any) should act on it. This pattern is sometimes called "repo-assist."

A minimal triage workflow that handles routing:

```aw wrap
---
description: Triage incoming issues and route to appropriate agents
on:
  issues:
    types: [opened]
engine: copilot
tools:
  github:
    toolsets: [issues, labels]
    min-integrity: unapproved
safe-outputs:
  label-issue:
  comment-issue:
permissions:
  issues: write
  contents: read
---

Review the newly opened issue. Based on the issue content:

1. Apply the most relevant label from the existing label set.
2. If the issue is a bug report with a clear reproduction, add the label `needs-investigation`.
3. If the issue is from an `approved`-integrity author (owner, member, or collaborator), add `trusted-contributor` and consider assigning the Copilot agent to investigate.
4. If the issue appears to be spam or off-topic, add `invalid` and post a brief explanation comment.
5. Otherwise, post a comment thanking the contributor and explaining what information is still needed.
```

### Combining Triage with Code-Modifying Agents

The triage workflow uses `min-integrity: unapproved` so it sees all community issues. Downstream specialized agents use stricter filtering. The triage workflow's output (labels, assignments) becomes the signal that downstream agents rely on:

```text
Issue opened (any author)
  → Triage agent (min-integrity: unapproved)
    Adds label: "trusted-contributor" if author is approved
    Assigns Copilot if label is set
  → Code review agent (min-integrity: approved)
    Only triggered when Copilot is assigned
    Safe from untrusted input
```

This separation means compute-intensive agents only run on content that has passed human-or-automation review.

## Scaling Strategies

### Token Budget Awareness

Integrity filtering directly reduces token consumption: items filtered by the gateway never appear in the agent's context window. On a busy public repository, `min-integrity: approved` can reduce context size by 60–90% compared to `min-integrity: none`.

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

Workflows automatically use dual concurrency control (per-workflow and per-engine). For triage workflows, you may want a higher concurrency limit so issues are processed in parallel rather than queued:

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
gh aw audit diff BASELINE_ID CURRENT_ID
```

### Common Failure Patterns

**Missing tool calls**

The agent attempted a tool that wasn't configured or used the wrong name. Check the `missing_tools` section of the audit output.

Fixes:
- Add the required tool to the `tools:` section in frontmatter.
- Verify safe-output names don't have an incorrect prefix (`safeoutputs-` is wrong; use the tool name directly).
- Check MCP server connectivity.

**Authentication failures**

Token permissions are too narrow or an API key is missing.

Fixes:
- Review the `permissions:` block in the workflow frontmatter.
- Ensure required secrets (`COPILOT_GITHUB_TOKEN`, `ANTHROPIC_API_KEY`, etc.) are set.
- Check [Authentication Reference](/gh-aw/reference/auth/) for token requirements.

**Integrity filtering blocking expected content**

The `DIFC_FILTERED` events in the audit's firewall section show exactly which items were removed and why.

Fixes:
- Verify the author's GitHub association matches your `min-integrity` setting.
- Add the author to `trusted-users` if they should be promoted.
- Add `approval-labels` to allow label-based promotion.
- Use `gh aw logs --filtered-integrity` to find all runs with filtering events.

**Safe output validation failures**

The agent produced output in the wrong format or called a tool that isn't enabled.

Fixes:
- Review `safe-outputs:` configuration in frontmatter.
- Check `safe_outputs.jsonl` in the audit artifacts for the exact call that failed.
- See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for format requirements.

**Token budget exhaustion**

The run hit the token limit before completing its task.

Fixes:
- Raise `min-integrity` to reduce the agent's context.
- Add `cache-memory:` to reuse context across runs.
- Simplify the prompt or break the workflow into smaller focused tasks.
- Set a tighter `rate-limit` to prevent concurrent runs from competing for the same token budget.

**Network blocks**

A domain the agent needs is blocked by the firewall.

Fixes:
- Review the firewall section of the audit output.
- Add the required ecosystem or domain to `network.allowed`.
- See [Network Configuration Guide](/gh-aw/guides/network-configuration/) for ecosystem identifiers.

### Iterative Debug Workflow

1. Check the workflow run summary in the GitHub Actions UI.
2. Run `gh aw audit RUN_ID` for a structured breakdown.
3. For complex issues, use `/agent agentic-workflows` in Copilot Chat.
4. Edit the `.md` file → run `gh aw compile` to validate → trigger a new run.
5. Compare the new run against the baseline with `gh aw audit diff`.

## Worked Examples

### Public Repository

A public repository receives issues from anonymous users, contributors, and maintainers. The goal is to triage all issues, but only have the code-modifying agent act on trusted input.

**Triage workflow** (`issue-triage.md`):

```aw wrap
---
on:
  issues:
    types: [opened]
engine: copilot
tools:
  github:
    toolsets: [issues, labels]
    min-integrity: unapproved
safe-outputs:
  label-issue:
  comment-issue:
permissions:
  issues: write
  contents: read
---

Classify the issue and apply one label from the existing label set.
If the issue is a quality bug report, also add the label `agent-ready`.
```

**Code fix workflow** (`auto-fix.md`):

```aw wrap
---
on:
  issues:
    types: [labeled]
engine: copilot
tools:
  github:
    toolsets: [issues, pull_requests]
    min-integrity: approved
    approval-labels:
      - "agent-ready"
safe-outputs:
  create-pull-request:
permissions:
  issues: write
  pull-requests: write
  contents: write
---

The issue labeled `agent-ready` needs a fix. Reproduce the bug,
implement a minimal fix, and open a pull request.
```

The triage workflow labels issues as `agent-ready` when they meet quality criteria. The code fix workflow only runs on labeled issues and uses `approval-labels` to ensure it processes even external issues that a maintainer has approved.

### Inner-Source Repository

An organization's internal repository should allow cross-team contributions. Members from partner teams don't have formal collaborator status but are trusted.

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
engine: copilot
tools:
  github:
    allowed-repos: "myorg/*"
    min-integrity: approved
    trusted-users: ${{ vars.TRUSTED_PARTNER_ACCOUNTS }}
safe-outputs:
  comment-pull-request:
permissions:
  pull-requests: write
  contents: read
---

Review the pull request for correctness, style, and test coverage.
Post a detailed review comment.
```

Partner team members are listed in the `TRUSTED_PARTNER_ACCOUNTS` organization variable. `allowed-repos: "myorg/*"` prevents the agent from reading data from external repos.

### High-Security Repository

A repository requiring auditability wants the agent to only act on code that is already in the default branch.

```aw wrap
---
on:
  schedule:
    - cron: "0 6 * * *"
engine: copilot
tools:
  github:
    allowed-repos: "myorg/secure-repo"
    min-integrity: merged
    blocked-users: ${{ vars.GH_AW_GITHUB_BLOCKED_USERS }}
safe-outputs:
  create-issue:
permissions:
  issues: write
  contents: read
---

Scan the merged commits from the last 24 hours for security anti-patterns.
Open an issue for each finding with severity, location, and remediation steps.
```

`min-integrity: merged` ensures the agent only analyzes code that has passed code review and been merged. Even if a malicious PR was opened, it would never appear in the agent's context.

## Related Documentation

- [Integrity Filtering Reference](/gh-aw/reference/integrity/) — Complete `min-integrity` and policy configuration
- [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) — Preventing runaway workflows
- [Cost Management](/gh-aw/reference/cost-management/) — Token budget tracking and optimization
- [Audit Commands](/gh-aw/reference/audit/) — `gh aw audit` and `gh aw logs` reference
- [Debugging Workflows](/gh-aw/troubleshooting/debugging/) — Detailed debugging procedures
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) — Structured output configuration
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) — Firewall and domain setup
- [GitHub Tools Reference](/gh-aw/reference/github-tools/) — Full `tools.github` options
