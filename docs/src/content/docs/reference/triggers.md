---
title: Triggers
description: Triggers in GitHub Agentic Workflows
sidebar:
  order: 400
---

The `on:` section uses standard GitHub Actions syntax to define workflow triggers. For example:

```yaml wrap
on:
  issues:
    types: [opened]
```

## Trigger Types

GitHub Agentic Workflows supports all standard GitHub Actions triggers plus additional enhancements for reactions, cost control, and advanced filtering.

### Dispatch Triggers (`workflow_dispatch:`)

Run workflows manually from the GitHub UI, API, or via `gh aw run`/`gh aw trial`. [Full syntax reference](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#on).

**Basic trigger:**
```yaml wrap
on:
  workflow_dispatch:
```

**With input parameters:**
```yaml wrap
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      priority:
        description: 'Task priority'
        required: false
        type: choice
        options:
          - low
          - medium
          - high
        default: medium
```

#### Accessing Inputs in Markdown

Use `${{ github.event.inputs.INPUT_NAME }}` expressions to access workflow_dispatch inputs in your markdown content:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
permissions:
  contents: read
safe-outputs:
  create-discussion:
---

# Research Assistant

Research the following topic: "${{ github.event.inputs.topic }}"

Provide a comprehensive summary with key findings and recommendations.
```

**Supported input types:**
- `string` - Free-form text input
- `boolean` - True/false checkbox
- `choice` - Dropdown selection with predefined options
- `environment` - Repository environment selector

### Scheduled Triggers (`schedule:`)

Run workflows on a recurring schedule using human-friendly expressions or [cron syntax](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule).

**Fuzzy Scheduling (Recommended for Daily Workflows):**

To distribute workflow execution times and prevent load spikes, use fuzzy schedules that let the compiler automatically scatter execution times:

```yaml wrap
on:
  schedule: daily  # Compiler scatters execution time deterministically
```

```yaml wrap
on:
  schedule:
    - cron: daily  # Each workflow gets a different scattered time
```

For workflows that need to run around a specific time (with some flexibility), use the `around` constraint:

```yaml wrap
on:
  schedule: daily around 14:00  # Scatters within ±1 hour (13:00-15:00)
```

The compiler deterministically assigns each workflow a unique time throughout the day based on the workflow file path. This ensures:
- **Load distribution**: Workflows run at different times, reducing server load spikes
- **Consistency**: The same workflow always gets the same execution time across recompiles
- **Simplicity**: No need to manually coordinate schedules across multiple workflows
- **Flexibility with constraints**: Use `around` to hint preferred times while still distributing load

**Human-Friendly Format:**

```yaml wrap
on:
  schedule: daily at 02:00  # Shorthand string format (not recommended - creates load spikes)
```

```yaml wrap
on:
  schedule:
    - cron: weekly on monday at 09:00
    - cron: monthly on 15 at 12:00
```

:::caution[Fixed Times Create Load Spikes]
Using explicit times like `0 0 * * *` or `daily at midnight` causes all workflows to run simultaneously, creating server load spikes. Similarly, hourly intervals with fixed minute offsets like `0 */2 * * *` synchronize all workflows to run at the same minute of each hour, and weekly schedules with fixed times like `weekly on monday at 09:00` cause all workflows to run at the same time each week. The compiler will warn you about these patterns. Use fuzzy schedules (`daily`, `every Nh`, `weekly`, or `weekly on <day>`) instead.
:::

**Supported Formats:**
- **Daily (Fuzzy)**: `daily` → Scattered time like `43 5 * * *` (compiler determines)
- **Daily (Fuzzy Around)**: `daily around HH:MM` → Scattered time within ±1 hour of target
  - `daily around 14:00` → `20 14 * * *` (scattered between 13:00-15:00)
  - `daily around 9am` → `38 8 * * *` (scattered between 08:00-10:00)
  - `daily around midnight` → `27 0 * * *` (scattered between 23:00-01:00)
  - `daily around noon` → Scattered time between 11:00-13:00
  - **With UTC offsets**: `daily around 3pm utc-5` → `33 19 * * *` (3 PM EST → scattered around 8 PM UTC)
  - **With time zones**: `daily around 14:00 utc+9` → `47 5 * * *` (2 PM JST → scattered around 5 AM UTC)
  - Supports all time formats: `HH:MM`, `midnight`, `noon`, `Npm`, `Nam` with optional UTC offsets
- **Daily (Fixed)**: `daily at HH:MM` or `daily at midnight/noon` or `daily at Npm/Nam`
  - `daily at 02:00` → `0 2 * * *` (⚠️ Warning: fixed time)
  - `daily at midnight` → `0 0 * * *` (⚠️ Warning: fixed time)
  - `daily at 3pm` → `0 15 * * *` (⚠️ Warning: fixed time)
  - `daily at 6am` → `0 6 * * *` (⚠️ Warning: fixed time)
- **Weekly (Fuzzy)**: `weekly` or `weekly on <day>` or `weekly on <day> around HH:MM`
  - `weekly` → Scattered day and time like `43 5 * * 3` (compiler determines)
  - `weekly on monday` → Scattered time like `43 5 * * 1` (compiler determines)
  - `weekly on friday` → Scattered time like `28 14 * * 5` (compiler determines)
  - `weekly on monday around 09:00` → Scattered time within ±1 hour like `32 9 * * 1` (08:00-10:00)
  - `weekly on friday around 5pm` → Scattered time within ±1 hour like `18 16 * * 5` (16:00-18:00)
  - `weekly on sunday around midnight` → Scattered time within ±1 hour like `47 23 * * 0` (23:00-01:00)
  - **With UTC offsets**: `weekly on monday around 08:00 utc+9` → Scattered around 11 PM UTC previous day
  - Supports all time formats: `HH:MM`, `midnight`, `noon`, `Npm`, `Nam` with optional UTC offsets
- **Weekly (Fixed)**: `weekly on <day> at HH:MM` or `weekly on <day> at Npm/Nam`
  - `weekly on monday at 06:30` → `30 6 * * 1` (⚠️ Warning: fixed time)
  - `weekly on friday at 17:00` → `0 17 * * 5` (⚠️ Warning: fixed time)
  - `weekly on friday at 5pm` → `0 17 * * 5` (⚠️ Warning: fixed time)
- **Monthly**: `monthly on <day>` or `monthly on <day> at HH:MM` or `monthly on <day> at Npm/Nam`
  - `monthly on 15 at 09:00` → `0 9 15 * *`
  - `monthly on 1` → `0 0 1 * *`
  - `monthly on 15 at 9am` → `0 9 15 * *`
- **Intervals**: `every N minutes/hours` or `every Nm/Nh/Nd/Nw/Nmo` (minimum 5 minutes)
  - `every 10 minutes` → `*/10 * * * *` (minute intervals don't scatter)
  - **Hourly (Fuzzy)**: `every 2h` → Scattered minute like `53 */2 * * *` (compiler determines)
  - **Hourly (Fuzzy)**: `every 1h` → Scattered minute like `28 */1 * * *` (compiler determines)
  - **Hourly (Fixed)**: `0 */2 * * *` → (⚠️ Warning: fixed minute offset)
  - `every 1d` → `0 0 * * *`
  - `every 1w` → `0 0 * * 0`
  - `every 1mo` → `0 0 1 * *`
- **UTC Offsets**: Add `utc+N` or `utc-N` or `utc+HH:MM` to convert from local time to UTC
  - `daily at 02:00 utc+9` → `0 17 * * *` (2 AM JST → 5 PM UTC previous day)
  - `daily at 14:00 utc-5` → `0 19 * * *` (2 PM EST → 7 PM UTC)
  - `weekly on monday at 09:30 utc+05:30` → `0 4 * * 1` (9:30 AM IST → 4 AM UTC)
  - `daily at 3pm utc+9` → `0 6 * * *` (3 PM JST → 6 AM UTC)
- **Time Formats**: `HH:MM` (24-hour), `midnight`, `noon`, `Npm` (1pm-12pm), `Nam` (1am-12am)
  - `12am` = midnight (00:00)
  - `12pm` = noon (12:00)
  - `1am` = 01:00, `11pm` = 23:00

The human-friendly format is automatically converted to standard cron expressions, with the original format preserved as a comment in the generated workflow file.

**Standard Cron Format:**

```yaml wrap
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9 AM UTC
  stop-after: "+7d"      # Stop after a week
```

### Issue Triggers (`issues:`)

Trigger on issue events. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#issues).

```yaml wrap
on:
  issues:
    types: [opened, edited, labeled]
```

### Pull Request Triggers (`pull_request:`)

Trigger on pull request events. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request).

```yaml wrap
on:
  pull_request:
    types: [opened, synchronize, labeled]
    names: [ready-for-review, needs-review]
  reaction: "rocket"
```

#### Fork Filtering (`forks:`)

Pull request workflows block forks by default for security. Use the `forks:` field to allow specific fork patterns:

```yaml wrap
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
```yaml wrap
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

```yaml wrap
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

**Automatic repository and fork validation:** The compiler automatically injects repository ID and fork checks to prevent cross-repository attacks and fork execution. This safety condition ensures workflows only execute when triggered by workflow runs from the same repository and not from forked repositories.

**Branch restrictions required:** Include `branches` to limit which branch workflows can trigger the event. Without branch restrictions, the compiler emits warnings (or errors in strict mode). This prevents unexpected execution for workflow runs on all branches.

See the [Security Guide](/gh-aw/guides/security/#workflow_run-trigger-security) for detailed security behavior and implementation.

### Command Triggers (`slash_command:`)

The `slash_command:` trigger creates workflows that respond to `/command-name` mentions in issues, pull requests, and comments. See [Command Triggers](/gh-aw/reference/command-triggers/) for complete documentation.

**Basic Configuration:**
```yaml wrap
on:
  slash_command:
    name: my-bot
```

**Shorthand Format (String):**
```yaml wrap
on:
  slash_command: "my-bot"
```

**Shorthand Format (Slash Command):**
```yaml wrap
on: /my-bot
```

This ultra-short syntax automatically expands to include `slash_command` and `workflow_dispatch` triggers, similar to how `on: daily` expands to include schedule and workflow_dispatch.

**With Event Filtering:**
```yaml wrap
on:
  slash_command:
    name: summarize
    events: [issues, issue_comment]  # Only in issue bodies and comments
```

**Complete Workflow Example:**
```aw wrap
---
on:
  slash_command:
    name: code-review
    events: [pull_request, pull_request_comment]
permissions:
  contents: read
  pull-requests: write
tools:
  github:
    toolsets: [pull_requests]
safe-outputs:
  add-comment:
    max: 5
timeout-minutes: 10
---

# Code Review Assistant

When someone mentions /code-review in a pull request or PR comment,
analyze the code changes and provide detailed feedback.

The current context is: "${{ needs.activation.outputs.text }}"

Review the pull request changes and add helpful review comments on specific
lines of code where improvements can be made.
```

The command must appear as the **first word** in the comment or body text. Command workflows automatically add the "eyes" (👀) reaction and edit comments with workflow run links.

:::note[Deprecated Syntax]
The `command:` trigger field is deprecated. Use `slash_command:` instead.
:::

### Label Filtering (`names:`)

An additional kind of issue and pull request trigger is available in GitHub Agentic Workflows to specific label names using the `names:` field:

```yaml wrap
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, critical, security]
```

This filtering is especially useful for [LabelOps workflows](/gh-aw/examples/issue-pr-events/labelops/) where specific labels trigger different automation behaviors.

#### Shorthand Syntax for Label Triggers

GitHub Agentic Workflows provides convenient shorthand syntax for label-based triggers:

**Basic format:**
```yaml wrap
on: issue labeled bug
```

**Multiple labels (space-separated):**
```yaml wrap
on: issue labeled bug enhancement priority-high
```

**Multiple labels (comma-separated):**
```yaml wrap
on: issue labeled bug, enhancement, priority-high
```

**With explicit item type:**
```yaml wrap
on: pull_request labeled needs-review, ready-to-merge
```

All shorthand formats compile to the standard GitHub Actions syntax:

```yaml wrap
on:
  issues:  # or pull_request
    types: [labeled]
    names:
      - bug
      - enhancement
      - priority-high
```

**Supported entity types:**
- `issue labeled <labels>` - Issue label events
- `pull_request labeled <labels>` - Pull request label events
- `discussion labeled <labels>` - Discussion label events (GitHub Actions doesn't support `names` for discussions, so only the `types` filter is applied)

The shorthand syntax automatically includes `workflow_dispatch` trigger, similar to how `on: daily` expands to include both schedule and workflow_dispatch.

### Reactions (`reaction:`)

An additional option  `reaction:` is available within the `on:` section to enable emoji reactions on the triggering GitHub item (issue, PR, comment, discussion) to provide visual feedback about the workflow status:

```yaml wrap
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

```yaml wrap
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+25h"  # 25 hours from compilation time
```

Accepts absolute dates (`YYYY-MM-DD`, `MM/DD/YYYY`, `DD/MM/YYYY`, `January 2 2006`, `1st June 2025`, ISO 8601) or relative deltas (`+7d`, `+25h`, `+1d12h30m`) calculated from compilation time. The minimum granularity is hours - minute-only units (e.g., `+30m`) are not allowed. Recompiling the workflow resets the stop time.

### Manual Approval Gates (`manual-approval:`)

Require manual approval before workflow execution using GitHub environment protection rules:

```yaml wrap
on:
  workflow_dispatch:
  manual-approval: production
```

The `manual-approval` field sets the `environment` on the activation job, enabling manual approval gates configured in repository or organization settings. This provides human-in-the-loop control for sensitive operations.

The field accepts a string environment name that must match a configured environment in the repository. Configure approval rules, required reviewers, and wait timers in repository Settings → Environments. See [GitHub's environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) for environment configuration details.

### Skip-If-Match Condition (`skip-if-match:`)

Conditionally skip workflow execution when a GitHub search query has matches. Useful for preventing duplicate scheduled runs or waiting for prerequisites.

**Basic Usage (String Format):**
```yaml wrap
on:
  schedule:
    - cron: "0 13 * * 1-5"
  skip-if-match: 'is:issue is:open in:title "[daily-report]"'
```

**Advanced Usage (Object Format with Threshold):**
```yaml wrap
on:
  schedule:
    - cron: "0 9 * * 1"
  skip-if-match:
    query: "is:pr is:open label:urgent"
    max: 3  # Skip if 3 or more PRs match
```

**How it works:**
1. A pre-activation check runs the search query against the current repository
2. If the number of matches reaches or exceeds the threshold, the workflow is skipped
3. The query is automatically scoped to the current repository
4. String format implies `max: 1` (skip if any matches found)

**Common Use Cases:**

Prevent duplicate scheduled reports:
```yaml wrap
on:
  schedule:
    - cron: "0 9 * * 1"
  skip-if-match: 'is:issue is:open label:weekly-summary'
```

Wait for deployment PRs to close:
```yaml wrap
on:
  workflow_dispatch:
  skip-if-match: "is:pr is:open label:deployment"
```

Skip if processing queue is full (3+ items):
```yaml wrap
on:
  schedule:
    - cron: "0 */6 * * *"
  skip-if-match:
    query: "is:issue is:open label:needs-processing"
    max: 3
```

The search uses GitHub's issue/PR search API with efficient `per_page=1` query. Supports all standard GitHub search qualifiers (`is:`, `label:`, `in:title`, `author:`, etc.).

## Related Documentation

- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [LabelOps Guide](/gh-aw/examples/issue-pr-events/labelops/) - Label-based automation workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization