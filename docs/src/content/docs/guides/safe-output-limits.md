---
title: Safe Output Limits Guidelines
description: Guidelines for setting appropriate safe output limits based on workflow categories to prevent spam and ensure consistency.
sidebar:
  order: 6
---

Safe output limits control how many GitHub issues, PRs, comments, and discussions your agentic workflows can create. These guidelines help you choose appropriate limits based on your workflow's purpose and category, preventing spam while enabling legitimate automation.

## Why Limits Matter

Without thoughtful limits, workflows can:

- **Create spam**: Multiple workflows generating 5-10 issues daily quickly overwhelms teams
- **Consume API quota**: Excessive operations waste GitHub API resources
- **Fragment conversation**: Too many issues scatter discussion across multiple threads
- **Reduce signal-to-noise**: Important notifications get lost in automated noise

**Good limits** ensure workflows create valuable outputs without overwhelming users or systems.

## Workflow Categories

Different workflow types serve different purposes and need different limits. Understanding your workflow's category helps you choose appropriate limits.

### Meta-Orchestrators

**Examples**: Campaign Manager, Workflow Health Monitor, Agent Performance Analyzer

**Purpose**: Strategic oversight and coordination across multiple workflows or teams. These workflows analyze system-wide patterns and create high-level strategic issues for teams to address.

**Recommended Limits**:
```yaml
safe-outputs:
  create-issue:
    max: 3-5        # Strategic issues only
  add-comment:
    max: 10-15      # Follow-ups and coordination
  create-discussion:
    max: 2-3        # Reports and analysis
```

**Rationale**: Meta-orchestrators need to create issues to coordinate teams but should focus on strategic, high-value items. Most communication should happen through comments and discussions rather than creating new issues.

**Examples in Repository**:
- `campaign-manager.md`: `create-issue: 5`, `add-comment: 10`
- `agent-performance-analyzer.md`: `create-issue: 5`, `create-discussion: 2`, `add-comment: 10`

### Daily/Hourly Monitors

**Examples**: daily-*, hourly-*, scheduled metrics collectors

**Purpose**: Regular monitoring and reporting on repository metrics, health, or status. These workflows run on schedule and provide informational updates.

**Recommended Limits**:
```yaml
safe-outputs:
  create-issue:
    max: 0-1               # Critical alerts only (prefer 0)
  create-discussion:
    max: 1                 # Preferred for reports
    expires: 3d            # Auto-close old discussions
    close-older-discussions: true  # Keep only current
  add-comment:
    max: 1-3               # Updates to existing items
```

**Rationale**: Monitoring workflows generate repetitive, informational content that works better as discussions than issues. Issues should be reserved for critical alerts requiring action. Use `expires` and `close-older-discussions` to keep only current information visible.

**Examples in Repository**:
- `daily-code-metrics.md`: `create-discussion: 1` with `expires: 3d`
- `static-analysis-report.md`: `create-discussion: 1`
- `artifacts-summary.md`: `create-discussion: 1` with `close-older-discussions: true`

### Worker Workflows

**Examples**: Fixers, updaters, code formatters, dependency updaters

**Purpose**: Perform specific tasks like fixing issues, updating dependencies, or formatting code. These workflows do actual work rather than creating meta-issues.

**Recommended Limits**:
```yaml
safe-outputs:
  create-issue:
    max: 0-1        # Report completion or errors only
  create-pull-request:
    max: 1-3        # Actual fixes
  add-comment:
    max: 1-5        # Progress updates
```

**Rationale**: Workers should produce PRs (actual work) rather than issues (meta-work). Issues should only be created to report failures or request human intervention. Most communication happens through PR comments.

**Examples in Repository**:
- `tidy.md`: `create-pull-request: 1`
- `security-fix-pr.md`: `create-pull-request: 1`
- `jsweep.md`: `create-pull-request: 1`

### Campaign Orchestrators

**Examples**: *.campaign.g.md workflows, campaign generators

**Purpose**: Coordinate multiple related work items as part of a larger initiative. These workflows break down large projects into manageable tasks.

**Recommended Limits**:
```yaml
safe-outputs:
  create-issue:
    max: 5-10           # Based on campaign scope
  update-project:
    max: 20-50          # Project board updates
  add-comment:
    max: 10-20          # Coordination
  create-pull-request:
    max: 1-5            # Direct fixes
```

**Rationale**: Campaigns coordinate multiple work items, so higher issue limits are justified. However, limits should be proportional to campaign scope. Use project board integration to organize work rather than creating excessive standalone issues.

**Examples in Repository**:
- `campaign-generator.md`: Creates campaign-related issues based on scope
- Campaigns use `update-project` to organize work on boards

### Analysis/Research Workflows

**Examples**: Analyzers, auditors, reporters, research assistants

**Purpose**: Analyze data, audit systems, or research topics, then report findings. These workflows provide information rather than create work items.

**Recommended Limits**:
```yaml
safe-outputs:
  create-discussion:
    max: 1              # Preferred for reports
  create-issue:
    max: 0-2            # Only for actionable findings
  add-comment:
    max: 1-3            # Share results in context
```

**Rationale**: Analysis should inform decision-making, not create busywork. Use discussions for reports and only create issues for specific, actionable findings that require follow-up. Prefer commenting on existing items to keep conversation in context.

**Examples in Repository**:
- `copilot-pr-nlp-analysis.md`: `create-discussion: 1`
- `schema-consistency-checker.md`: `create-discussion: 1`
- `prompt-clustering-analysis.md`: `create-discussion: 1`

### PR/Event Responders

**Examples**: Dev Hawk, review bots, CI status reporters

**Purpose**: Respond to specific events (PRs, issues, workflow runs) with contextual feedback. These workflows provide targeted assistance in response to user actions.

**Recommended Limits**:
```yaml
safe-outputs:
  add-comment:
    max: 1-3            # Targeted feedback
  create-issue:
    max: 0              # Comment on existing items
  update-pull-request:
    max: 1              # Update PR body/title
```

**Rationale**: Responders should provide feedback in context (on the PR/issue that triggered them) rather than creating new issues that fragment conversation. Higher comment limits may be justified for complex multi-part feedback.

**Examples in Repository**:
- `dev-hawk.md`: `add-comment: 1`
- `ci-coach.md`: `add-comment: 1`
- `breaking-change-checker.md`: `add-comment: 1`

## Special Considerations

### Multi-Purpose Workflows

Some workflows serve multiple roles. For example:
- **speckit-dispatcher.md**: Dispatches to various commands, needs flexibility: `create-issue: 5`, `add-comment: 5`
- **q.md**: General-purpose assistant: `create-pull-request: 1`, `add-comment: 1`

**Guideline**: Base limits on the most permissive category the workflow belongs to, but document why higher limits are needed.

### Command-Triggered Workflows

Workflows triggered by user commands (e.g., `/plan`, `/research`) can justify higher limits since they run on-demand rather than automatically:
- `plan.md`: `create-issue: 6` (creates project plan with sub-tasks)
- `research.md`: `create-discussion: 1` (research reports)

**Guideline**: On-demand workflows can have higher limits than automated ones, but should still be proportional to the command's purpose.

### Staged Mode

All workflows should support staged mode for testing. Staged mode previews outputs without actually creating them. Limits apply equally to both staged and live modes.

```yaml
# Test workflow with staged mode
gh aw run workflow.md --staged
```

## Exception Process

If your workflow needs limits that exceed these guidelines, document:

1. **Why the limit is needed**: Specific use case or requirement
2. **Safeguards**: How you prevent over-creation (e.g., filtering, throttling)
3. **Review period**: When to reassess the limit (e.g., after 30 days)
4. **Fallback**: What happens if the limit is hit

**Example Exception Documentation**:
```yaml
# In workflow frontmatter or comment
safe-outputs:
  create-issue:
    max: 15  # Exception: Campaign scope analysis
    # Why: Analyzes 10+ repositories, creates 1 issue per repo
    # Safeguards: Only runs monthly, filters to active repos only
    # Review: Reassess after 3 months (2025-03-15)
    # Fallback: Creates discussion with summary if limit hit
```

## Default Limits Reference

When safe output types are enabled without explicit `max:`, these defaults apply:

| Safe Output Type | Default Max | Notes |
|------------------|-------------|-------|
| `create-issue` | 1 | Create GitHub issues |
| `create-pull-request` | 1 | Create pull requests |
| `create-discussion` | 1 | Create discussions |
| `add-comment` | 1 | Add comments to issues/PRs/discussions |
| `close-issue` | 1 | Close issues |
| `close-pull-request` | 1 | Close PRs (default 1, up to 10) |
| `close-discussion` | 1 | Close discussions |
| `update-issue` | 1 | Update issue fields |
| `update-pull-request` | 1 | Update PR fields |
| `update-project` | 10 | Update project boards |
| `add-labels` | 3 | Add labels |
| `add-reviewer` | 3 | Add PR reviewers |
| `hide-comment` | 5 | Hide comments |
| `link-sub-issue` | 1 | Link parent-child issues |
| `create-pull-request-review-comment` | 10 | Code review comments |
| `upload-assets` | 10 | Upload files to assets branch |
| `assign-to-agent` | 1 | Assign Copilot agent |
| `assign-to-user` | 1 | Assign users |
| `assign-milestone` | 1 | Assign milestones |

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for complete documentation of each type.

## Examples by Category

### Example: Daily Monitor (Good)

```yaml
---
name: Daily Metrics Report
on: daily
safe-outputs:
  create-discussion:
    category: "reports"
    max: 1
    expires: 3d
    close-older-discussions: true
---

Analyze repository metrics and create a discussion with findings.
Use discussions not issues since this is informational.
```

**Why this is good**:
- Uses discussions for reports (not issues)
- Limits to 1 discussion
- Auto-expires old reports
- Keeps only current report visible

### Example: Worker Workflow (Good)

```yaml
---
name: Dependency Updater
on: workflow_dispatch
safe-outputs:
  create-pull-request:
    max: 1
    draft: true
  add-comment:
    max: 2
---

Update dependencies and create a PR with changes.
Comment on the PR with update details.
```

**Why this is good**:
- Creates PRs (actual work) not issues (meta-work)
- Limits PR creation to 1
- Minimal comments for essential updates

### Example: Campaign Orchestrator (Good)

```yaml
---
name: Multi-Repo Modernization Campaign
on: workflow_dispatch
safe-outputs:
  create-issue:
    max: 10  # One per target repository
    labels: [campaign, modernization]
  update-project:
    max: 20  # Track all issues on board
  add-comment:
    max: 15  # Coordinate across issues
---

Create modernization issues for 10 target repositories.
Track on project board. Coordinate through comments.
```

**Why this is good**:
- Limits proportional to scope (10 repos)
- Uses labels to identify campaign issues
- Integrates with project board for organization
- Documents why limits are higher

### Example: What to Avoid

```yaml
# ❌ BAD: Daily workflow creating too many issues
---
name: Daily Security Audit
on: daily
safe-outputs:
  create-issue:
    max: 10  # Too high for daily automated run
---

# ❌ BAD: Analysis workflow creating issues instead of discussion
---
name: Code Quality Report
on: weekly
safe-outputs:
  create-issue:
    max: 5  # Should be create-discussion instead
---

# ❌ BAD: Worker creating issues instead of PRs
---
name: Formatting Fixer
on: push
safe-outputs:
  create-issue:
    max: 3  # Should create-pull-request with fixes
---
```

## Best Practices

1. **Start small**: Begin with lower limits, increase only if genuinely needed
2. **Use discussions for reports**: Reserve issues for actionable work items
3. **Comment in context**: Prefer commenting on existing items over creating new ones
4. **Enable auto-expiration**: Use `expires:` for time-bound content
5. **Close older items**: Use `close-older-discussions: true` to keep current
6. **Test in staged mode**: Always test with `--staged` before live runs
7. **Document exceptions**: Clearly explain why higher limits are needed
8. **Review periodically**: Reassess limits as workflows evolve

## Impact of Good Limits

Following these guidelines results in:

- **Less spam**: Teams receive targeted, valuable notifications
- **Better UX**: Users get relevant, actionable information
- **Clear intent**: Limits reflect workflow purpose
- **Consistency**: Similar workflows configured similarly
- **Resource efficiency**: Less API quota consumed

## Related Documentation

- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Complete safe output documentation
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Organizing workflows
- [Command Triggers](/gh-aw/reference/command-triggers/) - On-demand workflows
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options
