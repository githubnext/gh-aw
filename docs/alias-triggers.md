# 🏷️ Alias Triggers

This guide covers alias triggers and context text functionality for agentic workflows.

## Special `alias:` Trigger

GitHub Agentic Workflows add the convenience `alias:` trigger to create workflows that respond to `@mentions` in issues and comments.

```yaml
on:
  alias:
    name: my-bot  # Optional: defaults to filename without .md extension
```

This automatically creates:
- Issue and PR triggers (`opened`, `edited`, `reopened`)
- Comment triggers (`created`, `edited`)
- Conditional execution matching `@alias-name` mentions

You can combine `alias:` with other events like `workflow_dispatch` or `schedule`:

```yaml
on:
  alias:
    name: my-bot
  workflow_dispatch:
  schedule:
    - cron: "0 9 * * 1"
```

**Note**: You cannot combine `alias` with `issues`, `issue_comment`, or `pull_request` as they would conflict.

**Note**: Using this feature results in the addition of `.github/actions/check-team-member/action.yml` file to the repository when the workflow is compiled. This file is used to check if the user triggering the workflow has appropriate permissions to operate in the repository.

### Example alias workflow

```markdown
---
on:
  alias:
    name: summarize-issue
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
---

# Issue Summarizer

When someone mentions @summarize-issue in an issue or comment, 
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

**Note**: Using this feature results in the addition of ".github/actions/compute-text/action.yml" file to the repository when the workflow is compiled.

## Related Documentation

- [Frontmatter Options](frontmatter.md) - All configuration options for workflows
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Commands](commands.md) - CLI commands for workflow management
