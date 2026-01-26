---
title: Command Triggers
description: Learn about slash command triggers and context text functionality for agentic workflows, including special @mention triggers for interactive automation.
sidebar:
  order: 500
---

GitHub Agentic Workflows add the convenience `slash_command:` trigger to create workflows that respond to `/my-bots` in issues and comments.

```yaml wrap
on:
  slash_command:
    name: my-bot  # Optional: defaults to filename without .md extension
```

You can also use shorthand formats:

```yaml wrap
on:
  slash_command: "my-bot"  # Shorthand: string directly specifies command name
```

```yaml wrap
on: /my-bot  # Ultra-short: slash prefix automatically expands to slash_command + workflow_dispatch
```

## Multiple Command Identifiers

A single workflow can respond to multiple slash command names by providing an array of command identifiers:

```yaml wrap
on:
  slash_command:
    name: ["cmd.add", "cmd.remove", "cmd.list"]
```

When triggered, the matched command is available as `needs.activation.outputs.slash_command`, allowing your workflow to determine which command was used:

```aw wrap
---
on:
  slash_command:
    name: ["summarize", "summary", "tldr"]
permissions:
  issues: write
---

# Multi-Command Handler

You invoked the workflow using: `/${{ needs.activation.outputs.slash_command }}`

Now analyzing the content...
```

This feature enables command aliases and grouped command handlers without workflow duplication.

> [!NOTE]
> **Deprecated Syntax**
> 
> The `command:` trigger field is deprecated. Use `slash_command:` instead. The old syntax still works but will show a deprecation warning.

This automatically creates issue/PR triggers (`opened`, `edited`, `reopened`), comment triggers (`created`, `edited`), and conditional execution matching `/command-name` mentions.

> [!CAUTION]
> **Command Position Requirement**
> 
> The command must be the **first word** of the comment or body text to trigger the workflow. This prevents accidental triggers when the command is mentioned elsewhere in the content.

You can combine `slash_command:` with other events like `workflow_dispatch` or `schedule`:

```yaml wrap
on:
  slash_command:
    name: my-bot
  workflow_dispatch:
  schedule: weekly on monday
```

**Note**: You cannot combine `slash_command` with `issues`, `issue_comment`, or `pull_request` as they would conflict.

**Exception for Label-Only Events**: You CAN combine `slash_command` with `issues` or `pull_request` if those events are configured for label-only triggers (`labeled` or `unlabeled` types only). This allows workflows to respond to slash commands while also reacting to label changes.

```yaml wrap
on:
  slash_command: deploy
  issues:
    types: [labeled, unlabeled]  # Valid: label-only triggers don't conflict
```

This pattern is useful when you want a workflow that can be triggered both manually via commands and automatically when labels change.

## Filtering Command Events

By default, command triggers respond to `/command-name` mentions in all comment-related contexts. Use the `events:` field to restrict where commands are active:

```yaml wrap
on:
  slash_command:
    name: my-bot
    events: [issues, issue_comment]  # Only in issue bodies and issue comments
```

**Supported events:** `issues` (issue bodies), `issue_comment` (issue comments only), `pull_request_comment` (PR comments only), `pull_request` (PR bodies), `pull_request_review_comment` (PR review comments), `discussion` (discussion bodies), `discussion_comment` (discussion comments), or `*` (all comment events, default).

### Example command workflow

Using object format:

```aw wrap
---
on:
  slash_command:
    name: summarize-issue
permissions:
  issues: write
tools:
  github:
    toolsets: [issues]
---

# Issue Summarizer

When someone mentions /summarize-issue in an issue or comment, 
analyze and provide a helpful summary.

The current context text is: "${{ needs.activation.outputs.text }}"
```

## Context Text

All workflows access `needs.activation.outputs.text`, which provides **sanitized** context: for issues and PRs, it's `title + "\n\n" + body`; for comments and reviews, it's the body content.

```aw wrap
# Analyze this content: "${{ needs.activation.outputs.text }}"
```

**Why sanitized context?** The sanitized text neutralizes @mentions and bot triggers (like `fixes #123`), protects against XML injection, filters URIs to trusted HTTPS domains, limits content size (0.5MB max, 65k lines), and strips ANSI escape sequences.

**Comparison:**
```aw wrap
# RECOMMENDED: Secure sanitized context
Analyze this issue: "${{ needs.activation.outputs.text }}"

# DISCOURAGED: Raw context values (security risks)
Title: "${{ github.event.issue.title }}"
Body: "${{ github.event.issue.body }}"
```

## Reactions

Command workflows automatically add the "eyes" (👀) emoji reaction to triggering comments and edit them with workflow run links, providing immediate feedback. Customize the reaction:

```yaml wrap
on:
  slash_command:
    name: my-bot
  reaction: "rocket"  # Override default "eyes"
```

See [Reactions](/gh-aw/reference/frontmatter/) for available reactions and detailed behavior.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
