---
title: Pull Request Reviewer Trigger
description: Configure the synthetic on.pull_request_reviewer trigger for centralized pull request review workflows.
sidebar:
  order: 510
---

`on.pull_request_reviewer` is an experimental synthetic trigger (a convenience trigger that expands to multiple native GitHub Actions events) for pull request reviewer workflows. It compiles to centralized routing in the generated `agentic_commands.yml` workflow and combines reviewer lifecycle events with slash-command dispatch.

## Configure `on.pull_request_reviewer`

Use an empty value to default the slash command name to the workflow ID:

```yaml wrap
on:
  pull_request_reviewer:
```

Use a string to set a custom slash command name:

```yaml wrap
on:
  pull_request_reviewer: reviewer
```

`pull_request_reviewer` and `slash_command` are mutually exclusive in the same workflow. Defining both causes a compilation error.

Compiling a workflow with this trigger emits an experimental warning:

```text
Using experimental feature: pull_request_reviewer
```

## Trigger coverage

This trigger wires the workflow into three centralized paths.

First, reviewer lifecycle events are subscribed automatically:

- `pull_request` with `ready_for_review` and `review_requested`
- `pull_request_review` with `submitted`

Second, built-in slash-command routing is enabled for pull request contexts:

- `pull_request_comment`
- `pull_request_review_comment`

Slash commands still follow the same first-token rule described in [Command Triggers](/gh-aw/reference/command-triggers/): the command must be the first token in the comment body.

Third, dispatches are queued with PR-scoped concurrency. When no explicit `concurrency:` is set, the compiler applies:

```yaml wrap
concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number || github.event.issue.number || github.run_id }}-all-reviewers"
  queue: max
```

## How centralized routing works

When at least one reviewer workflow exists, the generated `agentic_commands.yml` workflow includes reviewer routes in `GH_AW_REVIEWER_ROUTING` and listens for reviewer lifecycle events alongside centralized slash-command events (including issue comments and pull-request review comments used by non-reviewer slash-command workflows).

For reviewer lifecycle events, the router dispatches the reviewer workflow through `workflow_dispatch` and includes `aw_context.reviewer_lifecycle_event`:

- `pull_request` for `ready_for_review` and `review_requested`
- `pull_request_review` for `submitted`

For `pull_request_review` submissions, the router resolves the target workflow from the workflow marker embedded in the review body (`<!-- gh-aw-workflow-id: ... -->`). If no marker is found, dispatch is skipped.

If the pull request is already closed when routing starts, centralized routing exits early and no reviewer workflow is dispatched.

## Constraints

- Reviewer slash command names must be unique across all `pull_request_reviewer` workflows (case-insensitive).
- `pull_request_reviewer` and `slash_command` cannot be declared together in one workflow.
- `pull_request_review` actions `edited` and `dismissed` are ignored by reviewer lifecycle routing.

## Related documentation

- [Triggers](/gh-aw/reference/triggers/) - Full trigger reference
- [Command Triggers](/gh-aw/reference/command-triggers/) - Slash-command behavior and centralized strategy
- [Safe Outputs Pull Requests](/gh-aw/reference/safe-outputs-pull-requests/) - Safe output patterns for PR workflows
