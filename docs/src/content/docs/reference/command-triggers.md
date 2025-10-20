---
title: Command Triggers
description: Learn about command triggers and context text functionality for agentic workflows, including special @mention triggers for interactive automation.
sidebar:
  order: 500
---

GitHub Agentic Workflows add the convenience `command:` trigger to create workflows that respond to `/my-bots` in issues and comments.

```yaml wrap
on:
  command:
    name: my-bot  # Optional: defaults to filename without .md extension
```

You can also use the shorthand string format:

```yaml wrap
on:
  command: "my-bot"  # Shorthand: string directly specifies command name
```

This automatically creates issue/PR triggers (`opened`, `edited`, `reopened`), comment triggers (`created`, `edited`), and conditional execution matching `/command-name` mentions.

You can combine `command:` with other events like `workflow_dispatch` or `schedule`:

```yaml wrap
on:
  command:
    name: my-bot
  workflow_dispatch:
  schedule:
    - cron: "0 9 * * 1"
```

**Note**: You cannot combine `command` with `issues`, `issue_comment`, or `pull_request` as they would conflict.

## Filtering Command Events

By default, command triggers respond to `/command-name` mentions in all comment-related contexts. Use the `events:` field to restrict where commands are active:

```yaml wrap
on:
  command:
    name: my-bot
    events: [issues, issue_comment]  # Only in issue bodies and issue comments
```

**Supported events:** `issues` (issue bodies), `issue_comment` (issue comments only), `pull_request_comment` (PR comments only), `pull_request` (PR bodies), `pull_request_review_comment` (PR review comments), or `*` (all comment events, default).

### Example command workflow

Using object format:

```aw wrap
---
on:
  command:
    name: summarize-issue
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
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

```yaml
on:
  command:
    name: my-bot
  reaction: "rocket"  # Override default "eyes"
```

See [Reactions](/gh-aw/reference/frontmatter/) for available reactions and detailed behavior.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
