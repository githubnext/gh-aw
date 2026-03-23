---
# Base configuration for AI-powered PR code review workflows
# Provides: cache-memory, GitHub PR tools, and review comment safe-outputs

tools:
  cache-memory: true
  github:
    toolsets: [pull_requests, repos]

safe-outputs:
  create-pull-request-review-comment:
    side: "RIGHT"
  submit-pull-request-review:
    max: 1
---

## PR Code Review Configuration

This shared component provides the standard tooling for AI pull request code review agents.

### Available Tools

- **`cache-memory`** — Persist review history across runs at `/tmp/gh-aw/cache-memory/`
  - Store previous review notes: `/tmp/gh-aw/cache-memory/pr-{number}.json`
  - Avoid repeating comments seen in previous reviews
- **GitHub PR tools** — Access PR diffs, file changes, review threads, and check runs

### Review Guidelines

1. **Check cache first** — Read `/tmp/gh-aw/cache-memory/pr-${{ github.event.issue.number }}.json` to avoid re-stating previous comments
2. **Use `get_diff`** — Fetch the actual diff to review line-by-line changes
3. **Use `get_review_comments`** — Check existing review threads before adding new ones
4. **Submit as a unified review** — Batch comments and call `submit-pull-request-review` once with an overall assessment

### Safe Output Usage

- `create-pull-request-review-comment` — Post inline comments on specific lines
- `submit-pull-request-review` — Submit the overall review (APPROVE / REQUEST_CHANGES / COMMENT)
