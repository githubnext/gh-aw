---
title: Ephemerals
description: Features for automatically expiring workflow resources and reducing noise in your repositories
sidebar:
  order: 9
---

GitHub Agentic Workflows includes several features designed to automatically expire resources and reduce noise in your repositories. These "ephemeral" features help keep your repository clean by automatically cleaning up temporary issues, discussions, pull requests, and workflow runs after they've served their purpose.

## Why Use Ephemerals?

Ephemerals control costs by stopping scheduled workflows at deadlines, reduce clutter by auto-expiring issues and discussions, keep status timelines clean by hiding older comments, and isolate automation via the SideRepoOps pattern.

## Expiration Features

### Workflow Stop-After

Automatically disable workflow triggering after a deadline to control costs and prevent indefinite execution.

```yaml wrap
on: weekly on monday
  stop-after: "+25h"  # 25 hours from compilation time
```

**Accepted formats**:
- **Absolute dates**: `YYYY-MM-DD`, `MM/DD/YYYY`, `DD/MM/YYYY`, `January 2 2006`, `1st June 2025`, ISO 8601
- **Relative deltas**: `+7d`, `+25h`, `+1d12h30m` (calculated from compilation time)

The minimum granularity is hours - minute-only units (e.g., `+30m`) are not allowed. Recompiling the workflow resets the stop time.

At the deadline, new runs are prevented while existing runs complete. The stop time persists through recompilation; use `gh aw compile --refresh-stop-time` to reset it. Common uses: trial periods, experimental features, orchestrated initiatives, and cost-controlled schedules.

See [Triggers Reference](/gh-aw/reference/triggers/#stop-after-configuration-stop-after) for complete documentation.

### Safe Output Expiration

Auto-close issues, discussions, and pull requests after a specified time period. This generates a maintenance workflow that runs automatically at appropriate intervals.

#### Issue Expiration

```yaml wrap
safe-outputs:
  create-issue:
    expires: 7  # Auto-close after 7 days
    labels: [automation, agentic]
```

#### Discussion Expiration

```yaml wrap
safe-outputs:
  create-discussion:
    expires: 3  # Auto-close after 3 days as "OUTDATED"
    category: "general"
```

#### Pull Request Expiration

```yaml wrap
safe-outputs:
  create-pull-request:
    expires: 14  # Auto-close after 14 days (same-repo only)
    draft: true
```

**Supported formats**:
- **Integer**: Number of days (e.g., `7` = 7 days)
- **Relative time**: `2h`, `7d`, `2w`, `1m`, `1y`

Hours less than 24 are treated as 1 day minimum for expiration calculation.

**Maintenance workflow frequency**: The generated `agentics-maintenance.yml` workflow runs at the minimum required frequency based on the shortest expiration time across all workflows:

| Shortest Expiration | Maintenance Frequency |
|---------------------|----------------------|
| 1 day or less | Every 2 hours |
| 2 days | Every 6 hours |
| 3-4 days | Every 12 hours |
| 5+ days | Daily |

**Expiration markers**: The system adds a visible checkbox line with an XML comment to the body of created items:
```markdown
- [x] expires <!-- gh-aw-expires: 2026-01-14T15:30:00.000Z --> on Jan 14, 2026, 3:30 PM UTC
```

The maintenance workflow searches for items with this expiration format (checked checkbox with the XML comment) and automatically closes them with appropriate comments and resolution reasons. Users can uncheck the checkbox to prevent automatic expiration.

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete documentation.

### Close Older Issues

Automatically close older issues with the same workflow-id marker when creating new ones. This keeps your issues focused on the latest information.

```yaml wrap
safe-outputs:
  create-issue:
    close-older-issues: true  # Close previous reports
```

When a new issue is created, up to 10 older issues with the same workflow-id marker are closed as "not planned" with a comment linking to the new issue. Requires `GH_AW_WORKFLOW_ID` to be set and appropriate repository permissions. Ideal for weekly reports and recurring analyses where only the latest result matters.

## Noise Reduction Features

### Hide Older Comments

Minimize previous comments from the same workflow before posting new ones. Useful for status update workflows where only the latest information matters.

```yaml wrap
safe-outputs:
  add-comment:
    hide-older-comments: true
    allowed-reasons: [outdated]  # Optional: restrict hiding reasons
```

Before posting, the system finds and minimizes previous comments from the same workflow (identified by `GITHUB_WORKFLOW`). Comments are hidden, not deleted. Use `allowed-reasons` to restrict which minimization reason is applied: `spam`, `abuse`, `off_topic`, `outdated` (default), or `resolved`. Useful for status updates, build notifications, and health checks where only the latest result matters.

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#hide-older-comments) for complete documentation.

### SideRepoOps Pattern

Run agentic workflows from a separate "side" repository that targets your main codebase. This isolates AI-generated issues, comments, and workflow runs from your main repository, keeping automation infrastructure separate from production code.

See [SideRepoOps](/gh-aw/patterns/side-repo-ops/) for complete setup and usage documentation.

### Text Sanitization

Control which GitHub repository references (`#123`, `owner/repo#456`) are allowed in workflow output text. When configured, references to unlisted repositories are escaped with backticks to prevent GitHub from creating timeline items.

```yaml wrap
safe-outputs:
  allowed-github-references: []  # Escape all references
  create-issue:
    target-repo: "my-org/main-repo"
```

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete documentation.

### Use Discussions Instead of Issues

For ephemeral content, use GitHub discussions instead of issues. Discussions are better suited for temporary content, questions, and updates that don't require long-term tracking.

```yaml wrap
safe-outputs:
  create-discussion:
    category: "general"
    expires: 7  # Auto-close after 7 days
    close-older-discussions: true
```

**Why discussions for ephemeral content?**

| Feature | Issues | Discussions |
|---------|--------|-------------|
| **Purpose** | Long-term tracking | Conversations & updates |
| **Searchability** | High priority in search | Lower search weight |
| **Project boards** | Native integration | Limited integration |
| **Auto-close** | Supported with maintenance workflow | Supported with maintenance workflow |
| **Timeline noise** | Can clutter project tracking | Separate from development work |

Ephemeral discussions work well for weekly reports, periodic analyses, temporary announcements, time-bound Q&A, and community updates.

**Combining features**:

```yaml wrap
safe-outputs:
  create-discussion:
    category: "Status Updates"
    expires: 14  # Close after 2 weeks
    close-older-discussions: true  # Replace previous reports
```

This keeps the "Status Updates" category clean: previous reports are closed on creation and all discussions auto-close after 14 days.

## Related Documentation

- [Triggers Reference](/gh-aw/reference/triggers/) - Complete trigger configuration including `stop-after`
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - All safe output types and expiration options
- [SideRepoOps](/gh-aw/patterns/side-repo-ops/) - Complete setup for side repository operations
- [Authentication](/gh-aw/reference/auth/) - Authentication and security considerations
- [Orchestration](/gh-aw/patterns/orchestration/) - Orchestrating multi-workflow initiatives
