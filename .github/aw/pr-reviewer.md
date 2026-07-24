---
description: Guidance for implementing PR reviewer agentic workflows with ready_for_review triggers, centralized slash commands, and safe review actions.
---

## PR Reviewer Workflow Pattern

For reviewer workflows that run automatically when a PR is ready and manually via slash command.

## Trigger Model

```yaml
on:
  pull_request:
    types: [ready_for_review]
  slash_command:
    strategy: centralized
    name: review
    events: [pull_request_comment, pull_request_review_comment]
```

`ready_for_review` starts review when drafts become reviewable. Centralized routing handles both PR comments and review comments via one entrypoint.

## Safe Outputs

- `create-pull-request-review-comment` — line-level feedback
- `resolve-pull-request-review-thread` — resolved threads
- `submit-pull-request-review` — final review state
- `update-pull-request-review` — amend an existing review

Keep `max` caps conservative to avoid runaway reviews.

## Default Review Events: No APPROVE

**The GitHub Actions actor (`GITHUB_TOKEN`) cannot `APPROVE` a pull request. It can post `COMMENT` and `REQUEST_CHANGES` reviews.**

By default, configure `submit-pull-request-review` with `allowed-events: [COMMENT, REQUEST_CHANGES]` to enforce this constraint:

```yaml
safe-outputs:
  submit-pull-request-review:
    max: 1
    allowed-events: [COMMENT, REQUEST_CHANGES]
```

Do not instruct the agent to approve a PR unless the workflow uses a PAT or app token with explicit pull-request approval permissions. Using `APPROVE` with the default `GITHUB_TOKEN` will fail at runtime.

## Integrity and GitHub Tool Access

```yaml
tools:
  github:
    min-integrity: approved
    toolsets: [pull_requests, issues, repos]
```

- Prefer `pull_requests` for reviewer operations.
- Add `issues` only when interacting with issue-style comment surfaces or cross-links.
- Use the lowest `min-integrity` that supports the required actions.

## Examples

- `.github/workflows/pr-code-quality-reviewer.md`
- `.github/workflows/mattpocock-skills-reviewer.md`
- `.github/workflows/test-quality-sentinel.md`
