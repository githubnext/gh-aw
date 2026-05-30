---
title: Fork Support
description: How GitHub Agentic Workflows behaves in forked repositories and how to allow PRs from trusted forks.
sidebar:
  order: 7
---

GitHub Agentic Workflows has two distinct fork scenarios with different behaviors: **inbound pull requests from forks** and **running workflows inside a forked repository**.

## Running workflows in a fork

Agentic workflows do **not** run in forked repositories. When a workflow runs in a fork, all jobs skip automatically using the `if: ${{ !github.event.repository.fork }}` condition injected at compile time.

This means:

- Agent jobs are skipped — no AI execution occurs
- Maintenance and self-update jobs do not run
- No secrets from the upstream repository are available

This is intentional. Forks lack the secrets and context required for agentic workflows to function correctly, and there is no safe way to run agents with partial configuration.

> [!NOTE]
> To run agentic workflows in your own repository, fork the upstream repo and configure your own secrets — the workflows will then run in your copy of the repository, which is not a fork from GitHub Actions' perspective.

## Inbound pull requests from forks

When a pull request is opened from a fork to your repository, the default behavior is to **block the workflow from running** — the `pull_request` trigger includes a repository ID check that verifies the PR head branch comes from the same repository.

To allow workflows to run for PRs from trusted fork repositories, use the `forks:` field:

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
    forks: ["trusted-org/*"]
---
```

### Fork patterns

The `forks:` field accepts a string or a list of repository patterns:

| Pattern | Matches |
|---|---|
| `"*"` | All forks (use with caution) |
| `"owner/*"` | All forks from a specific user or organization |
| `"owner/repo"` | A specific fork repository |

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
    forks:
      - "trusted-org/*"
      - "partner/specific-fork"
---
```

> [!WARNING]
> Allowing all forks (`"*"`) means any user who forks your repository can trigger agent execution. Workflows triggered from fork PRs run with the permissions configured in the workflow — review those permissions carefully before allowing untrusted forks.
