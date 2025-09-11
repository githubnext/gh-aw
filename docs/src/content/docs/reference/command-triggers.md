# 🏷️ Command Triggers

This guide covers command triggers and context text functionality for agentic workflows.

## Special `command:` Trigger

GitHub Agentic Workflows add the convenience `command:` trigger to create workflows that respond to `/mentions` in issues and comments.

```yaml
on:
  command:
    name: my-bot  # Optional: defaults to filename without .md extension
```

This automatically creates:
- Issue and PR triggers (`opened`, `edited`, `reopened`)
- Comment triggers (`created`, `edited`)
- Conditional execution matching `/command-name` mentions

You can combine `command:` with other events like `workflow_dispatch` or `schedule`:

```yaml
on:
  command:
    name: my-bot
  workflow_dispatch:
  schedule:
    - cron: "0 9 * * 1"
```

**Note**: You cannot combine `command` with `issues`, `issue_comment`, or `pull_request` as they would conflict.

**Note**: Using this feature results in the addition of `.github/actions/check-team-member/action.yml` file to the repository when the workflow is compiled. This file is used to check if the user triggering the workflow has appropriate permissions to operate in the repository.

### Example command workflow

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

## Context Text (`needs.task.outputs.text`)

All workflows have access to a special computed `needs.task.outputs.text` value that provides context based on the triggering event:

```markdown
# Analyze this content: "${{ needs.task.outputs.text }}"
```

**How `text` is computed:**
- **Issues**: `title + "\n\n" + body`
- **Pull Requests**: `title + "\n\n" + body`  
- **Issue Comments**: `comment.body`
- **PR Review Comments**: `comment.body`
- **PR Reviews**: `review.body`
- **Other events**: Empty string

**Note**: Using this feature results in the addition of `.github/actions/compute-text/action.yml` file to the repository when the workflow is compiled.

## Visual Feedback with Reactions

Command workflows can provide immediate visual feedback by adding reactions to triggering comments and automatically editing them with workflow run links:

```yaml
on:
  command:
    name: my-bot
  reaction: "eyes"
```

When someone mentions `/my-bot` in a comment, the workflow will:
1. Add the specified emoji reaction (👀) to the comment
2. Automatically edit the comment to include a link to the workflow run

This provides users with immediate feedback that their request was received and gives them easy access to monitor the workflow execution.

See [Visual Feedback (`reaction:`)](frontmatter.md#visual-feedback-reaction) for the complete list of available reactions.

## Related Documentation

- [Frontmatter Options](frontmatter.md) - All configuration options for workflows
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Commands](commands.md) - CLI commands for workflow management
