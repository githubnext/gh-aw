---
title: Triggers
description: Triggers in GitHub Agentic Workflows
sidebar:
  order: 400
---

The `on:` section uses standard GitHub Actions syntax to define workflow triggers. For example:

```yaml
on:
  issues:
    types: [opened]
```

## Trigger Types

GitHub Agentic Workflows supports all standard GitHub Actions triggers plus additional enhancements for reactions, cost control, and advanced filtering.

### Dispatch Triggers (`workflow_dispatch:`)

Run workflows manually from the GitHub UI, API, or via `gh aw run`/`gh aw trial`. [Full syntax reference](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#on).

```yaml
on:
    workflow_dispatch:
```

### Scheduled Triggers (`schedule:`)

Run workflows on a recurring schedule using [cron syntax](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule).

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM
  disable-workflow-after: "+7d"     # Disable after a week
```

### Issue Triggers (`issues:`)

Trigger on issue events. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#issues).

```yaml
on:
  issues:
    types: [opened, edited, labeled]
```

### Pull Request Triggers (`pull_request:`)

Trigger on pull request events. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request).

```yaml
on:
  pull_request:
    types: [opened, synchronize, labeled]
    names: [ready-for-review, needs-review]
  reaction: "rocket"
```

### Comment Triggers
```yaml
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
  discussion_comment:
    types: [created]
  reaction: "eyes"
```

### Command Triggers (`command:`)

The `command:` trigger creates workflows that respond to `/command-name` mentions in issues, pull requests, and comments. See [Command Triggers](/gh-aw/reference/command-triggers/) for complete documentation.

**Basic Configuration:**
```yaml
on:
  command:
    name: my-bot
```

**Shorthand Format:**
```yaml
on:
  command: "my-bot"
```

**With Event Filtering:**
```yaml
on:
  command:
    name: summarize
    events: [issues, issue_comment]  # Only in issue bodies and comments
```

**Complete Workflow Example:**
```aw wrap
---
on:
  command:
    name: code-review
    events: [pull_request, pull_request_comment]
permissions:
  contents: read
  pull-requests: write
engine: claude
tools:
  github:
    allowed: [add_pull_request_review_comment]
safe-outputs:
  add-comment:
    max: 5
timeout_minutes: 10
---

# Code Review Assistant

When someone mentions /code-review in a pull request or PR comment,
analyze the code changes and provide detailed feedback.

The current context is: "${{ needs.activation.outputs.text }}"

Review the pull request changes and add helpful review comments on specific
lines of code where improvements can be made.
```

The command must appear as the **first word** in the comment or body text. Command workflows automatically add the "eyes" (👀) reaction and edit comments with workflow run links.

### Label Filtering (`names:`)

An additional kind of issue and pull request trigger is available in GitHub Agentic Workflows to specific label names using the `names:` field:

```yaml
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, critical, security]
```

This filtering is especially useful for [LabelOps workflows](/gh-aw/guides/labelops/) where specific labels trigger different automation behaviors.

### Reactions (`reaction:`)

An additional option  `reaction:` is available within the `on:` section to enable emoji reactions on the triggering GitHub item (issue, PR, comment, discussion) to provide visual feedback about the workflow status:

```yaml
on:
  issues:
    types: [opened]
  reaction: "eyes"
```

The reaction is added to the triggering item. For issues/PRs, a comment with the workflow run link is also created. For comment events in command workflows, the comment is edited to include the run link.

**Available reactions:** `+1` 👍, `-1` 👎, `laugh` 😄, `confused` 😕, `heart` ❤️, `hooray` 🎉, `rocket` 🚀, `eyes` 👀

**Job outputs** (`add_reaction`): `reaction_id`, `comment_id` (issues/PRs only), `comment_url` (issues/PRs only)

### Disable Workflow After Configuration (`disable-workflow-after:`)

Automatically disable workflow triggering after a deadline to control costs and prevent workflows from running indefinitely.

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  disable-workflow-after: "+7d"  # Disable after 7 days from compilation
```

Accepts absolute dates (`YYYY-MM-DD`, `MM/DD/YYYY`, `DD/MM/YYYY`, `January 2 2006`, `1st June 2025`, ISO 8601) or relative deltas calculated from compilation time. Recompiling the workflow resets the stop time.

**Relative Time Format:**
- **`+Xmo`** - months (e.g., `+3mo` = 3 months)
- **`+Xw`** - weeks (e.g., `+2w` = 2 weeks)  
- **`+Xd`** - days (e.g., `+7d` = 7 days)
- **`+Xh`** - hours (e.g., `+25h` = 25 hours)
- **`+Xm`** - minutes (e.g., `+90m` = 90 minutes)

:::caution
Be careful with time units: `+10m` means 10 **minutes**, not months. Use `+10mo` for 10 months.
:::

**Legacy field:** The deprecated `stop-after` field is still supported for backward compatibility but will show a warning.

## Related Documentation

- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [LabelOps Guide](/gh-aw/guides/labelops/) - Label-based automation workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization