---
title: ChatOps
description: A guide to building interactive automation using command triggers and safe outputs for ChatOps-style workflows.
---

ChatOps brings automation directly into development conversations. GitHub Agentic Workflows makes ChatOps natural and secure through command triggers that respond to slash commands, and safe outputs that handle results securely without granting write permissions to AI agents.

## Overview

ChatOps provides an easy and secure way to bring automation directly into development conversations. Team members can trigger automation by typing simple slash commands like `/review` or `/deploy` directly in GitHub issues and pull requests. These are called command triggers.

Command triggers make any GitHub repository responsive to automation commands. When you configure a command trigger, your workflow automatically listens for specific slash commands in issues, pull requests, and comments.

```aw wrap
---
on:
  command:
    name: review
    events: [pull_request_comment]  # Only respond to /review in PR comments
roles: [admin, maintainer]  # Default security restriction
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

This workflow creates an AI code reviewer that activates when someone types `/review` in a pull request comment. The AI agent runs with minimal read permissions, while safe outputs handle comment creation through separate secured jobs.

## Filtering Command Events

By default, command triggers respond to mentions in all comment-related contexts: issue bodies, issue comments, pull request bodies, PR comments, and PR review comments. You can restrict where commands are active using the `events:` field:

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

By default, ChatOps workflows restrict execution to users with `admin` or `maintainer` repository permissions. This prevents unauthorized users from triggering automation. Permission checks happen at runtime, automatically canceling workflows for unauthorized users.

You can customize access using the `roles:` configuration, but using `roles: all` creates security risks, especially in public repositories where any authenticated user could trigger workflows.

## Accessing Context Information

ChatOps workflows have access to sanitized context from the triggering event through `needs.activation.outputs.text`. This provides safer access to issue content, pull request details, or comment text with reduced security risks.

```aw wrap
# In your workflow content, reference the sanitized text:
Analyze this content: "${{ needs.activation.outputs.text }}"
```

The sanitized context automatically filters out potential security issues like unauthorized mentions, malicious links, and excessive content while preserving the essential information your automation needs.

**Security Caution**: While sanitization reduces many risks, the content may still contain prompt injection attempts. Always treat user-provided content as potentially untrusted and design your automation to be resilient against malicious instructions embedded in issue descriptions, comments, or pull request content.

Start with simple, high-value commands and expand based on team needs. Each command should solve real problems and integrate naturally into existing development conversations.