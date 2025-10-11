---
title: Command Triggers
description: Learn about command triggers and context text functionality for agentic workflows, including special @mention triggers for interactive automation.
sidebar:
  order: 600
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

This automatically creates:
- Issue and PR triggers (`opened`, `edited`, `reopened`)
- Comment triggers (`created`, `edited`)
- Conditional execution matching `/command-name` mentions

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

**Supported event identifiers:**
- `issues` - Issue bodies (opened, edited, reopened)
- `issue_comment` - Comments on issues only (excludes PR comments)
- `pull_request_comment` - Comments on pull requests only (excludes issue comments)
- `pull_request` - Pull request bodies (opened, edited, reopened)
- `pull_request_review_comment` - Pull request review comments
- `*` - All comment-related events (default when omitted)

**Examples:**

Only respond in issue contexts:
```yaml wrap
on:
  command:
    name: triage
    events: [issues, issue_comment]
```

Only respond in pull request contexts:
```yaml wrap
on:
  command:
    name: review
    events: [pull_request, pull_request_comment, pull_request_review_comment]
```

Only respond in comments (not bodies):
```yaml wrap
on:
  command:
    name: help
    events: [issue_comment, pull_request_comment]
```

**Implementation Details:**

Both `issue_comment` and `pull_request_comment` map to GitHub Actions' `issue_comment` event. The compiler automatically generates appropriate filters:
- `issue_comment`: Adds condition `github.event.issue.pull_request == null` (comments on issues)
- `pull_request_comment`: Adds condition `github.event.issue.pull_request != null` (comments on PRs)

This provides precise control over where your commands are active without needing manual condition writing.

**Note**: Using this feature results in the addition of `.github/actions/check-team-member/action.yml` file to the repository when the workflow is compiled. This file is used to check if the user triggering the workflow has appropriate permissions to operate in the repository.

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

Or using the shorthand string format (same behavior, more concise):

```aw wrap
---
on:
  command: "summarize-issue"  # Shorthand: string directly specifies command name
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
---

# Issue Summarizer

Same workflow as above, just using the shorthand string syntax.
```

## Context Text (`needs.activation.outputs.text`)

All workflows have access to a special computed `needs.activation.outputs.text` value that provides **sanitized** context based on the triggering event:

```aw wrap
# Analyze this content: "${{ needs.activation.outputs.text }}"
```

**How `text` is computed:**
- **Issues**: `title + "\n\n" + body`
- **Pull Requests**: `title + "\n\n" + body`  
- **Issue Comments**: `comment.body`
- **PR Review Comments**: `comment.body`
- **PR Reviews**: `review.body`
- **Other events**: Empty string

**Why use sanitized context text instead of raw `github.event` values?**

The `needs.activation.outputs.text` provides critical security protections that raw context values lack:

- **@mention neutralization**: Prevents unintended notifications by converting `@user` to `` `@user` ``
- **Bot trigger safety**: Protects against accidental bot commands by converting `fixes #123` to `` `fixes #123` ``
- **XML injection protection**: Converts XML tags to safe parentheses format
- **URI security**: Only allows HTTPS URIs from trusted domains; others become "(redacted)"
- **Content safety**: Limits size (0.5MB max, 65k lines) and removes control characters
- **ANSI sanitization**: Strips escape sequences that could manipulate terminal output

**Comparison:**
```aw wrap
# RECOMMENDED: Secure sanitized context
Analyze this issue: "${{ needs.activation.outputs.text }}"

# DISCOURAGED: Raw context values (security risks)
Title: "${{ github.event.issue.title }}"
Body: "${{ github.event.issue.body }}"
```

**Note**: Using this feature results in the addition of `.github/actions/compute-text/action.yml` file to the repository when the workflow is compiled.

## Reactions

Command workflows **automatically** provide immediate visual feedback by adding the "eyes" (👀) emoji reaction to triggering comments and automatically editing them with workflow run links.

This default behavior can be customized by explicitly specifying a different reaction:

```yaml
on:
  command:
    name: my-bot
  reaction: "rocket"  # Override default "eyes" with custom reaction
```

When someone mentions `/my-bot` in a comment, the workflow will:
1. Add the emoji reaction (👀 by default, or your custom choice) to the comment
2. Automatically edit the comment to include a link to the workflow run

:::note
For non-command workflows triggered by `issues` or `pull_request` events with reactions enabled, the behavior is slightly different:
- A reaction is added to the issue/PR
- A new comment is created with the workflow run link (instead of editing an existing comment)
- The comment ID and URL are exposed as job outputs (`comment_id` and `comment_url`)
:::

This provides users with immediate feedback that their request was received and gives them easy access to monitor the workflow execution.

See [Reactions](/gh-aw/reference/frontmatter/) for the complete list of available reactions and detailed behavior.

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
