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

1. **Use `get_diff`** — Fetch the actual diff to review line-by-line changes
2. **Use `get_review_comments`** — Check existing review threads before adding new ones
3. **Submit as a unified review** — Batch comments and call `submit-pull-request-review` once with an overall assessment

### Safe Output Usage

- `create-pull-request-review-comment` — Post inline comments on specific lines
- `submit-pull-request-review` — Submit the overall review (APPROVE / REQUEST_CHANGES / COMMENT)
- `create_check_run` — When the final verdict is `APPROVE`, create one check run with `conclusion: "success"` summarizing that no blocking issues were found