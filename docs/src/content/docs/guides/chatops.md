---
title: Implementing ChatOps with Agentic Workflows
description: A guide to building interactive automation using command triggers and safe outputs for ChatOps-style workflows.
---

ChatOps provides an easy and secure way to bring automation directly into development conversations. Team members can trigger automation by typing simple slash commands like `/review` or `/deploy` directly in GitHub issues and pull requests.

GitHub Agentic Workflows makes ChatOps natural and secure through command triggers that respond to slash commands, and safe outputs that handle results securely without granting write permissions to AI agents.

## How ChatOps Works

Command triggers make any GitHub repository responsive to automation commands. When you configure a command trigger, your workflow automatically listens for specific slash commands in issues, pull requests, and comments.

```yaml
---
on:
  command:
    name: review
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

When someone types /review in a pull request, perform a thorough analysis of the changes.

Examine the diff for potential bugs, security vulnerabilities, performance implications, code style issues, and missing tests or documentation.

Create specific review comments on relevant lines of code and add a summary comment with overall observations and recommendations.
```

This workflow creates an AI code reviewer that activates when someone types `/review` in a pull request. The AI agent runs with minimal read permissions, while safe outputs handle comment creation through separate secured jobs.

## Security and Access Control

By default, ChatOps workflows restrict execution to users with `admin` or `maintainer` repository permissions. This prevents unauthorized users from triggering automation. Permission checks happen at runtime, automatically canceling workflows for unauthorized users.

You can customize access using the `roles:` configuration, but using `roles: all` creates security risks, especially in public repositories where any authenticated user could trigger workflows.

## Key Benefits

**Security**: AI agents run with minimal permissions while safe outputs handle write operations through separate secured jobs.

**Context**: Commands create permanent records within the conversation context, making automation decisions visible and auditable.

**Accessibility**: Complex automation becomes available through simple slash commands that any team member can use.

**Integration**: Automation feels like natural conversation rather than external tooling.

Start with simple, high-value commands and expand based on team needs. Each command should solve real problems and integrate naturally into existing development conversations.