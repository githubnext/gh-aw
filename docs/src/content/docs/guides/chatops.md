---
title: ChatOps
description: A guide to building interactive automation using command triggers and safe outputs for ChatOps-style workflows.
---

ChatOps brings automation into GitHub conversations through command triggers that respond to slash commands in issues, pull requests, and comments. Team members can trigger workflows by typing commands like `/review` or `/deploy` directly in discussions.

```aw wrap
---
on:
  command:
    name: review
    events: [pull_request_comment]  # Only respond to /review in PR comments
permissions:
  contents: read
  pull-requests: read
safe-outputs:
  create-pull-request-review-comment:
    max: 5
  add-comment:
---

# Code Review Assistant

When someone types /review in a pull request comment, perform a thorough analysis of the changes.

Examine the diff for potential bugs, security vulnerabilities, performance implications, code style issues, and missing tests or documentation.

Create specific review comments on relevant lines of code and add a summary comment with overall observations and recommendations.
```

When someone types `/review`, the AI analyzes code changes and posts review comments. The agent runs with read-only permissions while safe outputs handle write operations securely.

## Filtering Command Events

Command triggers respond to all comment contexts by default. Use the `events:` field to restrict where commands activate:

```aw wrap
---
on:
  command:
    name: triage
    events: [issues, issue_comment]  # Only in issue bodies and issue comments
---

# Issue Triage Bot

This command only responds when mentioned in issues, not in pull requests.
```

**Supported event identifiers:**
- `issues` - Issue bodies (opened, edited, reopened)
- `issue_comment` - Comments on issues only (excludes PR comments)
- `pull_request_comment` - Comments on pull requests only (excludes issue comments)
- `pull_request` - Pull request bodies (opened, edited, reopened)
- `pull_request_review_comment` - Pull request review comments
- `*` - All comment-related events (default when `events:` is omitted)

**Note**: Both `issue_comment` and `pull_request_comment` map to GitHub Actions' `issue_comment` event but with automatic filtering to distinguish between issue comments and PR comments. This provides precise control over where your commands are active.

## Security and Access Control

ChatOps workflows restrict execution to users with admin, maintainer, or write permissions by default. Permission checks happen at runtime, canceling workflows for unauthorized users.

Customize access with the `roles:` configuration. Use `roles: [admin, maintainer]` for stricter control. Avoid `roles: all` in public repositories as any authenticated user could trigger workflows.

## Accessing Context Information

Access sanitized event context through `needs.activation.outputs.text`:

```aw wrap
# Reference the sanitized text in your workflow:
Analyze this content: "${{ needs.activation.outputs.text }}"
```

Sanitization filters unauthorized mentions, malicious links, and excessive content while preserving essential information.

**Security**: Treat user-provided content as untrusted. Design workflows to resist prompt injection attempts in issue descriptions, comments, or pull request content.