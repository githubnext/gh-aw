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
      deploy_env:
        description: 'Target environment'
        required: false
        type: environment
        default: staging
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
- `environment` - Dropdown selection of GitHub environments configured in the repository

The `environment` input type automatically populates a dropdown with environments configured in repository Settings → Environments. It returns the environment name as a string and supports a `default` value. Unlike the `manual-approval:` field, using an `environment` input does not enforce environment protection rules—it only provides the environment name as a string value for use in your workflow logic.

### Scheduled Triggers (`schedule:`)

Run workflows on a recurring schedule using human-friendly expressions or [cron syntax](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#schedule).

**Fuzzy Scheduling (Recommended):**

Use fuzzy schedules to automatically scatter execution times and avoid load spikes:

```yaml wrap
on:
  schedule: daily  # Compiler assigns a unique scattered time per workflow
```

Use the `around` constraint for a preferred time with flexibility:

```yaml wrap
on:
  schedule: daily around 14:00  # Scatters within ±1 hour (13:00-15:00)
```

For workflows that should only run during specific hours (like business hours), use the `between` constraint:

```yaml wrap
on:
  schedule: daily between 9:00 and 17:00  # Scatters within 9am-5pm range
```

The compiler assigns each workflow a unique, deterministic execution time based on the file path, ensuring load distribution and consistency across recompiles. UTC offsets are supported on any time expression (e.g., `daily between 9am and 5pm utc-5`).

For a fixed time, use standard cron syntax. Add an optional `timezone` field to interpret the cron in a specific IANA timezone instead of UTC:

```yaml wrap
on:
  schedule:
    - cron: "30 6 * * 1"  # Monday at 06:30 UTC
    - cron: "0 9 15 * *"  # 15th of month at 09:00 UTC
    - cron: "30 9 * * 1-5"
      timezone: "America/New_York"  # 9:30 AM EST/EDT Mon-Fri
```

| Format | Example | Result | Notes |
|--------|---------|--------|-------|
| **Hourly (Fuzzy)** | `hourly` | `58 */1 * * *` | Compiler assigns scattered minute |
| **Daily (Fuzzy)** | `daily` | `43 5 * * *` | Compiler assigns scattered time |
| | `daily around 14:00` | `20 14 * * *` | Scattered within ±1 hour (13:00-15:00) |
| | `daily between 9:00 and 17:00` | `37 13 * * *` | Scattered within range (9:00-17:00) |
| | `daily between 9am and 5pm utc-5` | `12 18 * * *` | With UTC offset (9am-5pm EST → 2pm-10pm UTC) |
| | `daily around 3pm utc-5` | `33 19 * * *` | With UTC offset (3 PM EST → 8 PM UTC) |
| **Weekly (Fuzzy)** | `weekly` or `weekly on monday` | `43 5 * * 1` | Compiler assigns scattered time |
| | `weekly on friday around 5pm` | `18 16 * * 5` | Scattered within ±1 hour |
| **Intervals** | `every 10 minutes` | `*/10 * * * *` | Minimum 5 minutes |
| | `every 2h` | `53 */2 * * *` | Fuzzy: scattered minute offset |
| | `0 */2 * * *` | `0 */2 * * *` | Cron syntax for fixed times |

**Time formats:** `HH:MM` (24-hour), `midnight`, `noon`, `1pm`-`12pm`, `1am`-`12am`
**UTC offsets:** Add `utc+N` or `utc-N` to any time (e.g., `daily around 14:00 utc-5`)

Human-friendly formats are automatically converted to standard cron expressions, with the original format preserved as a comment in the generated workflow file.

### Issue Triggers (`issues:`)

Trigger on issue events. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#issues).

```yaml wrap
on:
  issues:
    types: [opened, edited, labeled]
```

#### Issue Locking (`lock-for-agent:`)

Prevent concurrent modifications to an issue during workflow execution by setting `lock-for-agent: true`:

```yaml wrap
on:
  issues:
    types: [opened, edited]
    lock-for-agent: true
```

When enabled, the issue is locked at workflow start and unlocked after completion (or before safe-output processing). The unlock step uses `always()` to ensure cleanup even on failure. Useful for workflows that make multiple sequential updates to an issue or need to prevent race conditions.

**Requirements:**
- Requires `issues: write` permission (automatically added to activation and conclusion jobs)
- Pull requests are silently skipped (they cannot be locked via the issues API)
- Already-locked issues are skipped without error

**Example workflow:**
```aw wrap
---
on:
  issues:
    types: [opened]
    lock-for-agent: true
permissions:
  contents: read
safe-outputs:
  add-comment:
    max: 3
---

# Issue Processor with Locking

Process the issue and make multiple updates without interference
from concurrent modifications.

Context: "${{ steps.sanitized.outputs.text }}"
```

### Pull Request Triggers (`pull_request:`)

Trigger on pull request events. [Full event reference](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request).

**Code availability:** When triggered by a pull request event, the coding agent has access to both the PR branch and the default branch.

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

The compiler uses repository ID comparison for reliable fork detection that is not affected by repository renames.

### Comment Triggers

**Note:** `issue_comment` events also fire for comments on pull requests (GitHub models PR comments as issue comments). When a comment is on a pull request, the coding agent has access to both the PR branch and the default branch.

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

#### Comment Locking (`lock-for-agent:`)

For `issue_comment` events, you can lock the parent issue during workflow execution:

```yaml wrap
on:
  issue_comment:
    types: [created, edited]
    lock-for-agent: true
```

This prevents concurrent modifications to the issue while processing the comment. The locking behavior is identical to the `issues:` trigger (see [Issue Locking](#issue-locking-lock-for-agent) above for full details).

**Note:** Pull request comments are silently skipped as pull requests cannot be locked via the issues API.

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

Workflows with `workflow_run` triggers include automatic security protections:

- **Repository/fork validation:** The compiler injects repository ID and fork checks, rejecting cross-repository or fork-triggered runs.
- **Branch restrictions required:** Include `branches` to limit triggering branches; without them the compiler warns (or errors in strict mode).

See the [Security Architecture](/gh-aw/introduction/architecture/) for details.

### Command Triggers (`slash_command:`)

The `slash_command:` trigger creates workflows that respond to `/command-name` mentions in issues, pull requests, and comments. See [Command Triggers](/gh-aw/reference/command-triggers/) for complete documentation including event filtering, context text, reactions, and examples.

### Label Command Trigger (`label_command:`)

The `label_command:` trigger activates a workflow when a specific label is applied to an issue, pull request, or discussion, and **automatically removes that label** so it can be re-applied to re-trigger. This treats a label as a one-shot command rather than a persistent state marker.

```yaml wrap
# Fires on issues, pull_request, and discussion by default
on:
  label_command: deploy

# Restrict to specific event types
on:
  label_command:
    name: deploy
    events: [pull_request]

# Disable automatic label removal (label stays on the item after activation)
on:
  label_command:
    name: deploy
    remove_label: false

# Shorthand string form
on: "label-command deploy"
```

The compiler generates `issues`, `pull_request`, and/or `discussion` events with `types: [labeled]`, adds a `workflow_dispatch` trigger with `item_number` for manual testing, and injects a label removal step in the activation job. The matched label name is exposed as `needs.activation.outputs.label_command`.

The `remove_label` field (boolean, default `true`) controls whether the label is automatically removed after activation. Set to `false` to keep the label on the item — useful when the label represents persistent state rather than a one-shot command. When `remove_label: false`, the workflow does not need `issues: write` or `pull-requests: write` permissions for label removal.

`label_command` can be combined with `slash_command:` — the workflow activates when either condition is met. See [LabelOps](/gh-aw/patterns/label-ops/) for patterns and examples.

### Label Filtering (`names:`)

Filter issue and pull request triggers by label names using the `names:` field. Unlike `label_command`, the label stays on the item after the workflow runs.

```yaml wrap
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, critical, security]
```

Use convenient shorthand for label-based triggers:

```yaml wrap
on: issue labeled bug
on: issue labeled bug, enhancement, priority-high  # Multiple labels
on: pull_request labeled needs-review, ready-to-merge
```

All shorthand formats compile to standard GitHub Actions syntax and automatically include the `workflow_dispatch` trigger. Supported for `issue`, `pull_request`, and `discussion` events. See [LabelOps workflows](/gh-aw/patterns/label-ops/) for automation examples.

### Reactions (`reaction:`)

Enable emoji reactions on triggering items (issues, PRs, comments, discussions) to provide visual workflow status feedback:

```yaml wrap
on:
  issues:
    types: [opened]
  reaction: "eyes"
```

The reaction is added to the triggering item. For issues/PRs, a comment with the workflow run link is created. For comment events in command workflows, the comment is edited to include the run link.

**Available reactions:** `+1` 👍, `-1` 👎, `laugh` 😄, `confused` 😕, `heart` ❤️, `hooray` 🎉, `rocket` 🚀, `eyes` 👀

### Activation Token (`on.github-token:`, `on.github-app:`)

Configure a custom GitHub token or GitHub App for the activation job **and all skip-if search checks**. The activation job posts the initial reaction and status comment on the triggering item, and skip-if checks use the same token to query the GitHub Search API. By default all of these operations use the workflow's `GITHUB_TOKEN`.

Use `github-token:` to supply a PAT or custom token:

```yaml wrap
on:
  issues:
    types: [opened]
  reaction: "eyes"
  github-token: ${{ secrets.MY_TOKEN }}
```

Use `github-app:` to mint a short-lived installation token instead:

```yaml wrap
on:
  issues:
    types: [opened]
  reaction: "rocket"
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_KEY }}
```

The `github-app` object accepts the same fields as the GitHub App configuration used elsewhere in the framework (`app-id`, `private-key`, and optionally `owner` and `repositories`). The token is minted once in the pre-activation job and is shared across the reaction step, the status comment step, and any skip-if search steps.

Both `github-token` and `github-app` can be defined in a **shared agentic workflow** and will be automatically inherited by any workflow that imports it (first-wins strategy). This means a central CentralRepoOps shared workflow can define the app config once and all importing workflows benefit automatically:

```yaml wrap
# shared-ops.md - define app config once
on:
  workflow_call:
  github-app:
    app-id: ${{ secrets.ORG_APP_ID }}
    private-key: ${{ secrets.ORG_APP_PRIVATE_KEY }}
    owner: myorg
```

```yaml wrap
# any-workflow.md - inherits github-app from the import
imports:
  - .github/workflows/shared/shared-ops.md
on:
  schedule:
    - cron: "*/30 * * * *"
  skip-if-no-match:
    query: "org:myorg label:agent-fix is:issue is:open"
    scope: none
```

> [!NOTE]
> `github-token` and `github-app` affect only the activation job (reactions, status comments, and skip-if searches). For the agent job, configure tokens via `tools.github.github-token`/`tools.github.github-app` or `safe-outputs.github-token`/`safe-outputs.github-app`. See [Authentication](/gh-aw/reference/auth/) for a full overview.

### Stop After Configuration (`stop-after:`)

Automatically disable workflow triggering after a deadline to control costs.

```yaml wrap
on: weekly on monday
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

Sets the `environment` on the activation job for human-in-the-loop approval before execution. The value must match a configured environment in repository Settings → Environments (approval rules, required reviewers, wait timers). See [GitHub's environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) for configuration details.

### Skip-If-Match Condition (`skip-if-match:`)

Conditionally skip workflow execution when a GitHub search query has matches. Useful for preventing duplicate scheduled runs or waiting for prerequisites.

```yaml wrap
on: daily
  skip-if-match: 'is:issue is:open in:title "[daily-report]"'  # Skip if any match
```

```yaml wrap
on: weekly on monday
  skip-if-match:
    query: "is:pr is:open label:urgent"
    max: 3  # Skip if 3 or more PRs match
```

A pre-activation check runs the search query against the current repository. If matches reach or exceed the threshold (default `max: 1`), the workflow is skipped. The query is automatically scoped to the current repository and supports all standard GitHub search qualifiers (`is:`, `label:`, `in:title`, `author:`, etc.).

#### Cross-Repo and Org-Wide Queries

By default the query is scoped to the current repository. Use `scope: none` to disable this and search across an entire org. For cross-repo or org-wide searches that require elevated permissions, configure `github-token` or `github-app` at the top-level `on:` section — the same token is shared across all skip-if checks and the activation job:

```yaml wrap
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-match:
    query: "org:myorg label:ops:in-progress is:issue is:open"
    scope: none
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
    owner: myorg
```

| Field | Location | Description |
|-------|----------|-------------|
| `scope: none` | inside `skip-if-match` | Disables the automatic `repo:owner/repo` qualifier |
| `github-token` | top-level `on:` | Custom PAT or token for all skip-if searches (e.g. `${{ secrets.CROSS_ORG_TOKEN }}`) |
| `github-app` | top-level `on:` | Mints a short-lived installation token shared across all skip-if steps; requires `app-id` and `private-key` |

`github-token` and `github-app` are mutually exclusive. String shorthand always uses the default `GITHUB_TOKEN` scoped to the current repository.

### Skip-If-No-Match Condition (`skip-if-no-match:`)

Conditionally skip workflow execution when a GitHub search query has **no matches** (or fewer than the minimum required). This is the opposite of `skip-if-match`.

```yaml wrap
on: weekly on monday
  skip-if-no-match: 'is:pr is:open label:ready-to-deploy'  # Skip if no matches
```

```yaml wrap
on:
  workflow_dispatch:
  skip-if-no-match:
    query: "is:issue is:open label:urgent"
    min: 3  # Only run if 3 or more issues match
```

A pre-activation check runs the search query against the current repository. If matches are below the threshold (default `min: 1`), the workflow is skipped. Can be combined with `skip-if-match` for complex conditions.

The same `scope: none` field available on `skip-if-match` works identically here. Authentication (`github-token` / `github-app`) is configured at the top-level `on:` section and is shared across all skip-if checks — a single mint step is emitted for both:

```yaml wrap
on:
  schedule:
    - cron: "*/15 * * * *"
  skip-if-no-match:
    query: "org:myorg label:agent-fix -label:ops:agentic is:issue is:open"
    scope: none
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
    owner: myorg
```

### Pre-Activation Steps (`on.steps:`)

Inject custom deterministic steps directly into the pre-activation job. Steps run after all built-in checks (membership, stop-time, skip-if, etc.) and **before** agent execution. This saves one workflow job compared to the multi-job pattern and keeps filtering logic co-located with the trigger configuration.

```yaml wrap
on:
  issues:
    types: [opened]
  steps:
    - name: Check issue label
      id: label_check
      env:
        LABELS: ${{ toJSON(github.event.issue.labels.*.name) }}
      run: echo "$LABELS" | grep -q '"bug"'
      # exits 0 (outcome: success) if the label is found, 1 (outcome: failure) if not

if: needs.pre_activation.outputs.label_check_result == 'success'
```

Each step with an `id` automatically gets an output `<id>_result` wired to `${{ steps.<id>.outcome }}` (values: `success`, `failure`, `cancelled`, `skipped`). This lets you gate the workflow on whether the step **succeeded or failed** via its exit code.

To pass an explicit value rather than relying on exit codes, set a step output and re-expose it via `jobs.pre-activation.outputs`:

```yaml wrap
on:
  issues:
    types: [opened]
  steps:
    - name: Check issue label
      id: label_check
      env:
        LABELS: ${{ toJSON(github.event.issue.labels.*.name) }}
      run: |
        if echo "$LABELS" | grep -q '"bug"'; then
          echo "has_bug_label=true" >> "$GITHUB_OUTPUT"
        else
          echo "has_bug_label=false" >> "$GITHUB_OUTPUT"
        fi

jobs:
  pre-activation:
    outputs:
      has_bug_label: ${{ steps.label_check.outputs.has_bug_label }}

if: needs.pre_activation.outputs.has_bug_label == 'true'
```

Explicit outputs defined in `jobs.pre-activation.outputs` take precedence over auto-wired `<id>_result` outputs on key collision.

### Pre-Activation Permissions (`on.permissions:`)

Grant additional GitHub token permission scopes to the pre-activation job. Use when `on.steps:` make GitHub API calls that require permissions beyond the defaults.

```yaml wrap
on:
  schedule: every 30m
  permissions:
    issues: read
    pull-requests: read
  steps:
    - name: Search for candidate issues
      id: search
      uses: actions/github-script@v8
      with:
        script: |
          const issues = await github.rest.issues.listForRepo(context.repo);
          core.setOutput('has_issues', issues.data.length > 0 ? 'true' : 'false');

jobs:
  pre-activation:
    outputs:
      has_issues: ${{ steps.search.outputs.has_issues }}

if: needs.pre_activation.outputs.has_issues == 'true'
```

Supported permission scopes: `actions`, `checks`, `contents`, `deployments`, `discussions`, `issues`, `packages`, `pages`, `pull-requests`, `repository-projects`, `security-events`, `statuses`.

`on.permissions` is merged on top of any permissions already required by the pre-activation job (e.g., `contents: read` for dev-mode checkout, `actions: read` for rate limiting).

## Trigger Shorthands

Instead of writing full YAML trigger configurations, you can use natural-language shorthand strings with `on:`. The compiler expands these into standard GitHub Actions trigger syntax and automatically includes `workflow_dispatch` so the workflow can also be run manually.

For label-based shorthands (`on: issue labeled bug`, `on: pull_request labeled needs-review`), see [Label Filtering](#label-filtering-names) above. For the label-command pattern, see [Label Command Trigger](#label-command-trigger-label_command) above.

### Push and Pull Request

```yaml wrap
on: push to main                    # Push to specific branch
on: push tags v*                    # Push tags matching pattern
on: pull_request opened             # PR with activity type
on: pull_request merged             # PR merged (maps to closed + merge condition)
on: pull_request affecting src/**   # PR touching paths (opened, synchronize, reopened)
on: pull_request opened affecting docs/**  # Activity type + path filter
```

`pull` is an alias for `pull_request`. Valid activity types: `opened`, `edited`, `closed`, `reopened`, `synchronize`, `assigned`, `unassigned`, `labeled`, `unlabeled`, `review_requested`, `merged`.

### Issues and Discussions

```yaml wrap
on: issue opened                    # Issue with activity type
on: issue opened labeled bug        # Issue opened with specific label (adds job condition)
on: discussion created              # Discussion with activity type
```

Valid issue types: `opened`, `edited`, `closed`, `reopened`, `assigned`, `unassigned`, `labeled`, `unlabeled`, `deleted`, `transferred`. Valid discussion types: `created`, `edited`, `deleted`, `transferred`, `pinned`, `unpinned`, `labeled`, `unlabeled`, `locked`, `unlocked`, `category_changed`, `answered`, `unanswered`.

### Other Shorthands

```yaml wrap
on: manual                          # workflow_dispatch (run manually)
on: manual with input version       # workflow_dispatch with a string input
on: workflow completed ci-test       # Trigger after another workflow completes
on: comment created                 # Issue or PR comment created
on: release published               # Release event (published, created, prereleased, etc.)
on: repository starred              # Repository starred (maps to watch event)
on: repository forked               # Repository forked
on: dependabot pull request         # PR from Dependabot (adds actor condition)
on: security alert                  # Code scanning alert
on: code scanning alert             # Alias for security alert (code scanning alert)
on: api dispatch custom-event       # Repository dispatch with custom event type
```

## Related Documentation

- [Schedule Syntax](/gh-aw/reference/schedule-syntax/) - Complete schedule format reference
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration
- [LabelOps](/gh-aw/patterns/label-ops/) - Label-based automation workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
