---
permissions:
  contents: read
  pull-requests: read
# Base configuration for AI-powered PR code review workflows
# Provides: GitHub PR tools and review comment safe-outputs

tools:
  github:
    toolsets: [pull_requests, repos]

safe-outputs:
  create-pull-request-review-comment:
    side: "RIGHT"
    max: 10
  submit-pull-request-review:
    max: 1
  create-check-run:
    max: 1
---

## PR Code Review Configuration

This shared component provides the standard tooling for AI pull request code review agents.

### Available Tools

- **GitHub PR tools** — Access PR diffs, file changes, review threads, and check runs

### Review Guidelines

1. **Use pre-fetched diff** — Read `/tmp/gh-aw/agent/pr-diff.patch` instead of calling `get_diff`; it is already capped to prevent token-heavy context payloads.
2. **Use pre-fetched review comments** — Read `/tmp/gh-aw/agent/pr-review-comments.json` instead of calling `get_review_comments`; check this file before posting new comments to avoid duplication.
3. **Submit as a unified review** — Batch comments and call `submit-pull-request-review` once with an overall assessment.
