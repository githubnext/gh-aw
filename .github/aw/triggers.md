---
description: Trigger patterns for GitHub Agentic Workflows — events, fuzzy scheduling, fork security, slash commands, and label commands.
---

## Trigger Patterns

### Standard GitHub Events

```yaml
on:
  issues:
    types: [opened, edited, closed]
  pull_request:
    types: [opened, edited, closed]
    forks: ["*"]              # Allow from all forks (default: same-repo only)
  push:
    branches: [main]
  schedule:
    - cron: "0 9 * * 1"  # Monday 9AM UTC
  workflow_dispatch:    # Manual trigger
```

### `workflow_run` Failure-Triage Pattern

Use this when reacting to failures from another workflow in the same repository:

```yaml
on:
  workflow_run:
    workflows: ["CI", "Deploy"]
    types: [completed]
  workflow_dispatch:
```

Then gate analysis to failure outcomes:

```yaml
if: contains(fromJson('["failure","timed_out","cancelled","action_required"]'), github.event.workflow_run.conclusion)
```

These are "non-success outcomes requiring triage"; keep the list explicit so readers can tighten (e.g., only `failure`) or broaden it.

No-op expectations for this pattern:

- `noop` when the monitored run concludes `success`.
- `noop` when the same failure already has an open incident issue (duplicate suppression).

#### Fuzzy Scheduling

Use fuzzy scheduling instead of exact cron to distribute execution times. Avoids load spikes and the "Monday wall of work" from weekend accumulation.

**Basic Fuzzy Schedules:**

```yaml
on:
  schedule: daily on weekdays    # Monday-Friday only (recommended for daily workflows)
  schedule: daily                # All 7 days
  schedule: weekly               # Once per week
  schedule: hourly               # Every hour
```

**Examples with Intervals:**

```yaml
on:
  schedule: every 2 hours on weekdays    # Every 2 hours, Monday-Friday
  schedule: every 6 hours                # Every 6 hours, all days
```

**Why Prefer Weekday Schedules:**

- Avoids Monday backlog from weekend accumulation
- Aligns with team business hours
- Notifications fire when team members are active

The compiler converts fuzzy schedules to deterministic cron (e.g., `daily on weekdays` → `43 5 * * 1-5`), scatters execution to avoid load spikes, and adds `workflow_dispatch:` for manual runs.

**Recommended Pattern:**

```yaml
# ✅ GOOD - Weekday schedule avoids Monday wall of work
on:
  schedule: daily on weekdays

# ⚠️ ACCEPTABLE - But may create Monday backlog
on:
  schedule: daily
```

#### Fork Security for Pull Requests

By default, `pull_request` triggers **block all forks** and only allow PRs from the same repository. Use the `forks:` field to explicitly allow forks:

```yaml
# Default: same-repo PRs only (forks blocked)
on:
  pull_request:
    types: [opened]

# Allow all forks
on:
  pull_request:
    types: [opened]
    forks: ["*"]

# Allow specific fork patterns
on:
  pull_request:
    types: [opened]
    forks: ["trusted-org/*", "trusted-user/repo"]
```

### Command Triggers (/mentions)

```yaml
on:
  slash_command:
    name: my-bot  # Responds to /my-bot in issues/comments
```

This automatically creates conditions to match `/my-bot` mentions in issue bodies and comments.

You can restrict where commands are active using the `events:` field:

```yaml
on:
  slash_command:
    name: my-bot
    events: [issues, issue_comment]  # Only in issue bodies and issue comments
```

**Supported event identifiers:**

- `issues` - Issue bodies (opened, edited, reopened)
- `issue_comment` - Comments on issues only (excludes PR comments)
- `pull_request_comment` - Comments on pull requests only (excludes issue comments)
- `pull_request` - Pull request bodies (opened, edited, reopened)
- `pull_request_review_comment` - Pull request review comments
- `*` - All comment-related events (default)

**Note**: `issue_comment` and `pull_request_comment` both map to GitHub Actions' `issue_comment` event with filtering to distinguish them.

### Label Command Triggers

Trigger workflows when specific labels are added to issues, PRs, or discussions:

```yaml
# Shorthand: trigger on any labeled event
on: label-command my-label

# Or with explicit configuration
on:
  label_command:
    name: ai-review        # Single label name (or use names: [...] for multiple)
    events: [pull_request] # Optional: restrict to issues, pull_request, discussion (default: all three)
    strategy: decentralized # Optional: route labeled events via generated agentic_commands.yml
    remove_label: false    # Optional: remove triggering label after activation (default: true)
```

Use `names:` for multiple labels that activate the same workflow:

```yaml
on:
  label_command:
    names: [ai-review, copilot-review]
    events: [pull_request]
```

By default, the triggering label is automatically removed after the workflow activates (`remove_label: true`). Set `remove_label: false` to keep the label.

The activated label name is exposed to downstream jobs as `${{ needs.activation.outputs.label_command }}`.

### Semi-Active Agent Pattern

```yaml
on:
  schedule:
    - cron: "0/10 * * * *"  # Every 10 minutes
  issues:
    types: [opened, edited, closed]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, closed]
  push:
    branches: [main]
  workflow_dispatch:
```
