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
  stop-after: "+7d"     # Stop after a week
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

#### Fork Filtering (`forks:`)

Pull request workflows block forks by default for security. Use the `forks:` field to allow specific fork patterns:

```yaml
on:
  pull_request:
    types: [opened, synchronize]
    forks: ["trusted-org/*"]  # Allow forks from trusted-org
```

**Available patterns:**
- `["*"]` - Allow all forks (use with caution)
- `["owner/*"]` - Allow forks from specific organization or user
- `["owner/repo"]` - Allow specific repository
- Omit `forks` field - Default behavior (same-repository PRs only)

The compiler uses repository ID comparison for reliable fork detection that is not affected by repository renames. See the [Security Guide](/gh-aw/guides/security/#fork-protection-for-pull-request-triggers) for detailed security implications.

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

### Workflow Run Triggers (`workflow_run:`)

Trigger workflows after another workflow completes. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#workflow_run).

```yaml
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches:
      - main
      - develop
```

#### Security Protections

Workflows with `workflow_run` triggers include automatic security protections:

**Automatic repository validation:** The compiler automatically injects a repository ID check to prevent cross-repository attacks. This safety condition ensures workflows only execute when triggered by workflow runs from the same repository.

**Branch restrictions required:** Include `branches` to limit which branch workflows can trigger the event. Without branch restrictions, the compiler emits warnings (or errors in strict mode). This prevents unexpected execution for workflow runs on all branches.

See the [Security Guide](/gh-aw/guides/security/#workflow_run-trigger-security) for detailed security behavior and implementation.

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
    toolset: [pull_requests]
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

### Stop After Configuration (`stop-after:`)

Automatically disable workflow triggering after a deadline to control costs.

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+25h"  # 25 hours from compilation time
```

Accepts absolute dates (`YYYY-MM-DD`, `MM/DD/YYYY`, `DD/MM/YYYY`, `January 2 2006`, `1st June 2025`, ISO 8601) or relative deltas (`+7d`, `+25h`, `+1d12h30m`) calculated from compilation time. The minimum granularity is hours - minute-only units (e.g., `+30m`) are not allowed. Recompiling the workflow resets the stop time.

### Manual Approval Gates (`manual-approval:`)

Require manual approval before workflow execution using GitHub environment protection rules:

```yaml
on:
  workflow_dispatch:
  manual-approval: production
```

The `manual-approval` field sets the `environment` on the activation job, enabling manual approval gates configured in repository or organization settings. This provides human-in-the-loop control for sensitive operations.

The field accepts a string environment name that must match a configured environment in the repository. Configure approval rules, required reviewers, and wait timers in repository Settings → Environments. See [GitHub's environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) for environment configuration details.

## Related Documentation

- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [LabelOps Guide](/gh-aw/guides/labelops/) - Label-based automation workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization