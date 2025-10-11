---
title: Triggers
description: Triggers in GitHub Agentic Workflows
sidebar:
  order: 400
---

The `on:` section uses standard GitHub Actions syntax to define workflow triggers. Here are some common examples:

```yaml
on:
  issues:
    types: [opened]
```

GitHub Agentic Workflows supports all standard GitHub Actions triggers plus additional enhancements for reactions, cost control, and advanced filtering.

### Dispatch Triggers (`workflow_dispatch:`)

You can create manual triggers using `workflow_dispatch:` to run workflows on-demand from the GitHub UI or API.

```yaml
on:
    workflow_dispatch:
```

See GitHub Docs for more details: [Workflow syntax for GitHub Actions - on](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#on).

### Scheduled Triggers (`schedule:`)
```yaml
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM
  stop-after: "+7d"     # Stop after a week
```

See GitHub Docs for more details: [Events that trigger workflows - Schedule](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule).

### Issue Triggers (`issues:`)
```yaml
on:
  issues:
    types: [opened, edited, labeled]
```

See GitHub Docs for more details: [Events that trigger workflows - Issues](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#issues).

### Pull Request Triggers (`pull_request:`)
```yaml
on:
  pull_request:
    types: [opened, synchronize, labeled]
    names: [ready-for-review, needs-review]
  reaction: "rocket"
```

See GitHub Docs for more details: [Events that trigger workflows - Pull Request](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request).

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

An additional kind of trigger called `command:` is supported, see [Command Triggers](/gh-aw/reference/command-triggers/) for special `/my-bot` triggers and context text functionality.

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

### Stop After Configuration (`stop-after:`)

An additional configuration option `stop-after:` is available within the `on:` section as a cost-control measure to automatically disable workflow triggering after a deadline:

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

## Related Documentation

- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [LabelOps Guide](/gh-aw/guides/labelops/) - Label-based automation workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization