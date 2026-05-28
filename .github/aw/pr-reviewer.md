---
description: Guidance for implementing PR reviewer agentic workflows with ready_for_review triggers, centralized slash commands, and safe review actions.
---

## PR Reviewer Workflow Pattern

Use this pattern for reviewer workflows that should run automatically when a PR is ready, and manually via slash command in PR discussions.

## Trigger Model

Use `ready_for_review` for automatic review starts:

```yaml
on:
  pull_request:
    types: [ready_for_review]
  slash_command:
    strategy: centralized
    name: review
    events: [pull_request_comment, pull_request_review_comment]
```

- `pull_request.types: [ready_for_review]` starts review when draft PRs become reviewable.
- Centralized slash-command routing enables one command entrypoint for both PR comments and review comments.

## Safe Output Behavior

Reviewer workflows should focus on these outputs:

- `create-pull-request-review-comment` for line-level feedback.
- `resolve-pull-request-review-thread` for resolved/handled threads.
- `submit-pull-request-review` to create the final review state.
- `update-pull-request-review` when amending an existing review is preferable to creating a new one.

Keep caps conservative (`max`) to avoid noisy or runaway reviews.

## Integrity and GitHub Tool Access

Use constrained GitHub access with explicit integrity and toolsets:

```yaml
tools:
  github:
    min-integrity: approved
    toolsets: [pull_requests, issues, repos]
```

Guidance:
- Prefer `pull_requests` for reviewer operations.
- Include `issues` only when the workflow intentionally interacts with issue-style comment surfaces or cross-links.
- Choose the lowest `min-integrity` that still supports the required reviewer actions.

## Existing `ready_for_review` Reviewer Examples

Use these workflows as references:

- `.github/workflows/pr-code-quality-reviewer.md`
- `.github/workflows/mattpocock-skills-reviewer.md`
- `.github/workflows/test-quality-sentinel.md`
