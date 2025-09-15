---
title: Alias Triggers  
description: Learn about special @mention triggers and context text functionality for agentic workflows, enabling interactive automation through command-style triggers.
---

Alias triggers provide a way to create interactive agentic workflows that respond to special @mention-style commands in GitHub issues and comments.

## What are Alias Triggers?

Alias triggers allow you to create workflows that activate when someone mentions a specific command in an issue or comment using the `/command-name` format. This enables interactive, on-demand automation that team members can invoke as needed.

## Command Syntax

Alias triggers use the `command:` event type in your workflow frontmatter:

```yaml
on:
  command:
    name: my-command  # Optional: defaults to filename without .md extension
```

When configured, users can trigger the workflow by mentioning `/my-command` in:
- Issue descriptions and comments
- Pull request descriptions and comments
- Review comments

## Basic Example

```markdown
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

The current context text is: "${{ needs.task.outputs.text }}"
```

## Context Text Access

All alias-triggered workflows have access to contextual information through `${{ needs.task.outputs.text }}`:

- **Issues**: `title + "\n\n" + body`
- **Pull Requests**: `title + "\n\n" + body`  
- **Issue Comments**: `comment.body`
- **PR Review Comments**: `comment.body`
- **PR Reviews**: `review.body`

This allows your agentic workflow to understand what it's responding to.

## Visual Feedback

You can provide immediate visual feedback when commands are triggered:

```yaml
on:
  command:
    name: my-bot
reaction: "eyes"
```

This will:
1. Add the specified emoji reaction (👀) to the triggering comment
2. Automatically edit the comment to include a link to the workflow run

## Security Considerations

- Only team members with appropriate repository permissions can trigger command workflows
- The workflow compiler automatically adds permission checks
- Commands respect the same security model as other GitHub Actions

## Common Use Cases

- **Code Review Assistance**: `/review-pr` to get AI-powered code analysis
- **Issue Triage**: `/triage` to automatically categorize and label issues  
- **Documentation Updates**: `/update-docs` to refresh documentation
- **Release Preparation**: `/prepare-release` to automate release checklists
- **Bug Analysis**: `/analyze-bug` to investigate reported issues

## Combining with Other Triggers

You can combine command triggers with other events:

```yaml
on:
  command:
    name: my-bot
  schedule:
    - cron: "0 9 * * 1"  # Also run weekly
  workflow_dispatch:      # Allow manual triggering
```

**Note**: You cannot combine `command` with `issues`, `issue_comment`, or `pull_request` as they would conflict.

## Related Documentation

- **[Command Triggers](../reference/command-triggers/)** - Detailed technical implementation
- **[Frontmatter Options](../reference/frontmatter/)** - Complete configuration reference
- **[Visual Feedback](../reference/frontmatter/#visual-feedback-reaction)** - Available reaction options
- **[Security Notes](../guides/security-notes/)** - Security best practices