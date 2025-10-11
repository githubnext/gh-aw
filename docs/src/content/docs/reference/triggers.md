---
title: Triggers
description: Triggers in GitHub Agentic Workflows
sidebar:
  order: 325
---

The `on:` section uses standard GitHub Actions syntax to define workflow triggers. Here are some common examples:

```yaml
on:
  issues:
    types: [opened]
```

GitHub Agentic Workflows supports all standard GitHub Actions triggers plus additional enhancements for cost control, user feedback, and advanced filtering.

## Stop After Configuration (`stop-after:`)

You can add a `stop-after:` option within the `on:` section as a cost-control measure to automatically disable workflow triggering after a deadline:

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+25h"  # 25 hours from compilation time
```

**Relative time delta (calculated from compilation time):**
```yaml
on:
  issues:
    types: [opened]
  stop-after: "+25h"      # 25 hours from now
```

**Supported absolute date formats:**
- Standard: `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD`
- US format: `MM/DD/YYYY HH:MM:SS`, `MM/DD/YYYY`  
- European: `DD/MM/YYYY HH:MM:SS`, `DD/MM/YYYY`
- Readable: `January 2, 2006`, `2 January 2006`, `Jan 2, 2006`
- Ordinals: `1st June 2025`, `June 1st 2025`, `23rd December 2025`
- ISO 8601: `2006-01-02T15:04:05Z`

**Supported delta units:**
- `d` - days
- `h` - hours
- `m` - minutes

Note that if you specify a relative time, it is calculated at the time of workflow compilation, not when the workflow runs. If you re-compile your workflow, e.g. after a change, the effective stop time will be reset.

## Reactions (`reaction:`)

You can add a `reaction:` option within the `on:` section to enable emoji reactions on the triggering GitHub item (issue, PR, comment, discussion) to provide visual feedback about the workflow status:

```yaml
on:
  issues:
    types: [opened]
  reaction: "eyes"
```

**Behavior:**
- **For `issues` and `pull_request` events**: Adds the emoji reaction AND creates a comment with a link to the workflow run
- **For comment events** (`issue_comment`, `pull_request_review_comment`): Adds the emoji reaction and edits the comment to include the workflow run link (command workflows only)

**Outputs:**
The `add_reaction` job exposes the following outputs for use by downstream jobs:
- `reaction_id`: The ID of the created reaction
- `comment_id`: The ID of the created comment (for `issues`/`pull_request` events)
- `comment_url`: The URL of the created comment (for `issues`/`pull_request` events)

**Available reactions:**
- `+1` (👍)
- `-1` (👎)
- `laugh` (😄)
- `confused` (😕)
- `heart` (❤️)
- `hooray` (🎉)
- `rocket` (🚀)
- `eyes` (👀)

## Command Triggers (`command:`)

An additional kind of trigger called `command:` is supported, see [Command Triggers](/gh-aw/reference/command-triggers/) for special `/my-bot` triggers and context text functionality.

> [!NOTE]
> Command workflows automatically enable the "eyes" (👀) reaction by default. This can be customized by explicitly specifying a different reaction in the `reaction:` field.

## Label Filtering (`names:`)

When using `labeled` or `unlabeled` event types for `issues` or `pull_request` triggers, you can filter to specific label names using the `names:` field:

```yaml
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, critical, security]
```

**How it works:**
- The `names:` field is removed from the final workflow YAML and commented out for documentation
- A conditional `if` expression is automatically generated to check if the label name matches
- The workflow only runs when one of the specified labels triggers the event

**Syntax options:**

```yaml
# Single label name
names: bug

# Multiple label names (array)
names: [bug, enhancement, feature]
```

**Example for pull requests:**

```yaml
on:
  pull_request:
    types: [labeled]
    names: ready-for-review
```

This filtering is especially useful for [LabelOps workflows](/gh-aw/guides/labelops/) where specific labels trigger different automation behaviors.

## Common Trigger Patterns

### Issue-based Workflows
```yaml
on:
  issues:
    types: [opened, edited, labeled]
  reaction: "eyes"
```

### Pull Request Workflows
```yaml
on:
  pull_request:
    types: [opened, synchronize, labeled]
    names: [ready-for-review, needs-review]
  reaction: "rocket"
```

### Scheduled Workflows
```yaml
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM
  stop-after: "+7d"     # Stop after a week
```

### Comment-based Workflows
```yaml
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
  reaction: "eyes"
```

## Related Documentation

- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [LabelOps Guide](/gh-aw/guides/labelops/) - Label-based automation workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization