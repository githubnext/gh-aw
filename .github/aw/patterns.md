---
description: Patterns and scenarios for crafting efficient agentic workflows
infer: false
---

# Agentic Workflow Patterns

This file provides scenarios and guidance on how to craft efficient agentic workflows. Use these patterns to build workflows that are secure, maintainable, and effective.

## Pattern Index

- [ChatOps Pattern](#chatops-pattern) - Interactive automation via slash commands
- [IssueOps Pattern](#issueops-pattern) - Automated issue management and triage
- [DailyOps Pattern](#dailyops-pattern) - Scheduled incremental improvements
- [DispatchOps Pattern](#dispatchops-pattern) - Manual workflow execution with inputs
- [LabelOps Pattern](#labelops-pattern) - Label-triggered automation
- [ProjectOps Pattern](#projectops-pattern) - GitHub Projects board automation
- [MultiRepoOps Pattern](#multirepoops-pattern) - Cross-repository automation
- [SideRepoOps Pattern](#siderepoops-pattern) - Isolated experimentation in side repositories
- [TrialOps Pattern](#trialops-pattern) - Testing workflows before deployment

## Core Principles

### Security First

1. **Minimal Permissions**: Start with `permissions: read-all` and use `safe-outputs` for write operations
2. **Sanitized Context**: Use `${{ needs.activation.outputs.text }}` instead of raw event fields
3. **Network Constraints**: Explicitly configure `network:` allowlists when needed
4. **Fork Protection**: PR workflows block forks by default; use `forks:` only for trusted sources
5. **Role-Based Access**: Use `roles:` to restrict who can trigger workflows

### Safe Outputs Architecture

Always prefer `safe-outputs` over granting write permissions to the main job:

```yaml
# ✅ GOOD: Minimal permissions + safe-outputs
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
  create-issue:
  create-pull-request:

# ❌ BAD: Direct write permissions
permissions:
  contents: read
  issues: write
  pull-requests: write
```

### Fuzzy Scheduling

For daily/weekly workflows, use fuzzy scheduling to avoid load spikes:

```yaml
# ✅ GOOD: Fuzzy schedule (compiler distributes times)
on:
  schedule: daily

# ❌ BAD: Fixed time (concentrates all workflows)
on:
  schedule:
    - cron: "0 0 * * *"
```

**Note**: `workflow_dispatch:` is automatically added by the compiler when using fuzzy scheduling.

### Keep Frontmatter Minimal

Omit fields with sensible defaults:

```yaml
# ✅ GOOD: Minimal frontmatter
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  add-comment:
---

# ❌ BAD: Unnecessary fields
---
engine: copilot          # Default, no need to specify
timeout-minutes: 30      # Has sensible default
on:
  issues:
    types: [opened]
  workflow_dispatch:      # Auto-added by compiler for some triggers
permissions:
  contents: read
safe-outputs:
  add-comment:
---
```

---

## ChatOps Pattern

**Trigger**: Slash commands in issues, PRs, or comments  
**Use Case**: Interactive automation requiring human judgment  
**Example Scenarios**: Code review, deployment, analysis, team collaboration

### Pattern Structure

```aw
---
on:
  slash_command:
    name: bot-name
    events: [pull_request_comment]  # Restrict to PR comments
permissions:
  contents: read
  pull-requests: read
safe-outputs:
  add-comment:
  create-pull-request-review-comment:
---

# Bot Name

Respond to /bot-name mentions in pull request comments.

Your task: ${{ needs.activation.outputs.text }}
```

### Key Characteristics

- **Command-triggered**: Responds to `/command-name` mentions
- **Context-aware**: Use `needs.activation.outputs.text` for sanitized content
- **Event filtering**: Restrict with `events:` field (issues, pull_request_comment, etc.)
- **Permission-controlled**: Default access for admin/maintainer/write roles
- **Interactive**: Provides on-demand assistance

### Event Filtering

```yaml
# Only in issue bodies and issue comments
events: [issues, issue_comment]

# Only in PR comments (excludes issue comments)
events: [pull_request_comment]

# Only in PR review comments
events: [pull_request_review_comment]

# All comment contexts (default)
events: [*]
```

### Security Considerations

- Default role access: `[admin, maintainer, write]`
- Customize with `roles:` for stricter control
- Avoid `roles: all` in public repositories
- Treat `needs.activation.outputs.text` as untrusted user input

### Real-World Examples

- **[grumpy-reviewer.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/grumpy-reviewer.md)** - `/grumpy` triggers code review with personality
- Code deployment trigger - `/deploy staging` initiates deployment
- Analysis trigger - `/analyze security` performs security scan

---

## IssueOps Pattern

**Trigger**: Issue events (opened, edited, labeled)  
**Use Case**: Automated issue management and triage  
**Example Scenarios**: Auto-triage, categorization, initial responses, quality checks

### Pattern Structure

```aw
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
  add-labels:
    allowed: [bug, enhancement, question, documentation]
    max: 2
---

# Issue Triage Assistant

Analyze new issue content and provide helpful guidance.

Issue content: "${{ needs.activation.outputs.text }}"

Categorize and add appropriate labels, then respond with next steps.
```

### Key Characteristics

- **Event-triggered**: Activates on issue lifecycle events
- **Automated triage**: Categorizes and labels automatically
- **Initial response**: Provides immediate feedback to reporters
- **Read-only main job**: Uses safe-outputs for write operations

### Common Configurations

```yaml
# Label addition with constraints
safe-outputs:
  add-labels:
    allowed: [bug, enhancement, question, documentation]
    max: 2

# Comment with count limit
safe-outputs:
  add-comment:
    max: 1

# Issue creation for tracking
safe-outputs:
  create-issue:
    title-prefix: "[triage] "
    labels: [needs-review]
    expires: 7d  # Auto-close after 7 days
```

### Sub-Issue Patterns

Break large work into sub-tasks using parent-child hierarchies:

```aw
---
safe-outputs:
  create-issue:
    title-prefix: "[task] "
    max: 6
---

# Planning Assistant

Create parent issue with temporary ID, then sub-issues:

{"type": "create_issue", "temporary_id": "aw_abc123def456", "title": "Epic", "body": "..."}
{"type": "create_issue", "parent": "aw_abc123def456", "title": "Task 1", "body": "..."}
```

**Tip**: Filter sub-issues from `/issues` with `no:parent-issue` query parameter.

### Real-World Examples

- **[issue-classifier.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/issue-classifier.md)** - Auto-classifies as bug/feature
- Auto-responder - Provides instant feedback to issue reporters
- Duplicate detector - Identifies and links duplicate issues

---

## DailyOps Pattern

**Trigger**: Scheduled (daily/weekly)  
**Use Case**: Incremental improvements over time  
**Example Scenarios**: Code quality, dependency updates, documentation sync, technical debt

### Pattern Structure

```aw
---
on:
  schedule: daily  # Fuzzy scheduling (compiler distributes times)
permissions:
  contents: read
  actions: read
safe-outputs:
  create-pull-request:
    title-prefix: "[daily] "
    labels: [automation, improvement]
    reviewers: copilot
    draft: false
    skip-if-match: 'is:pr is:open in:title "[daily]"'  # Avoid duplicates
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"
    close-older-discussions: true  # Keep only latest
tools:
  cache-memory: true  # Maintain state across runs
---

# Daily Improvement Bot

Make small incremental improvements to [aspect] of the codebase.

Review recent changes, identify opportunities, implement improvements.

Track progress in discussion. Create draft PR when changes are ready.
```

### Key Characteristics

- **Scheduled execution**: Runs daily/weekly on weekdays
- **Incremental progress**: Small, reviewable changes
- **State persistence**: Uses `cache-memory` for continuity
- **Progress tracking**: Creates/updates discussions
- **Duplicate prevention**: Uses `skip-if-match` to avoid multiple open PRs
- **Draft PRs**: Signals need for review

### Phased Approach

1. **Research** - Analyze current state, create discussion
2. **Configuration** - Define improvement steps, create config PR
3. **Execution** - Make changes, verify, create draft PRs

### Discussion Tracking

```yaml
# Create discussion for tracking
safe-outputs:
  create-discussion:
    title-prefix: "${{ github.workflow }}"
    category: "ideas"
    close-older-discussions: true  # Close older discussions

# Or update existing discussion
safe-outputs:
  add-comment:
    target: "4750"  # Specific discussion number
    discussion: true
```

### Persistent Memory

```yaml
tools:
  cache-memory: true  # Persists at /tmp/gh-aw/cache-memory/
```

State persists across runs for tracking progress, metrics, and knowledge.

### Best Practices

- Schedule weekdays only (avoid weekend buildup)
- Keep changes reviewable in 5-10 minutes
- Use draft PRs to signal review needed
- Track progress in discussions
- Handle failures by creating issues
- Enable `workflow_dispatch` for manual testing
- Use `close-older-discussions: true` to prevent clutter
- Use `skip-if-match` to prevent duplicate PRs

### Real-World Examples

- **[daily-doc-updater.md](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/daily-doc-updater.md)** - Syncs docs with merged PRs
- **daily-fact.md** - Posts daily facts to discussion thread
- Test improvement - Systematically adds tests for coverage
- Performance optimization - Identifies and implements perf improvements

---

## DispatchOps Pattern

**Trigger**: Manual execution (`workflow_dispatch`)  
**Use Case**: On-demand tasks, testing, operational commands  
**Example Scenarios**: Research, deployments, testing, debugging

### Pattern Structure

```aw
---
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      depth:
        description: 'Analysis depth'
        type: choice
        options:
          - brief
          - detailed
        default: brief
permissions:
  contents: read
safe-outputs:
  create-discussion:
---

# Research Assistant

Research topic: "${{ github.event.inputs.topic }}"
Depth: ${{ github.event.inputs.depth }}

Provide analysis with key findings and recommendations.
```

### Key Characteristics

- **Manual trigger**: Run via GitHub UI or CLI
- **Custom inputs**: String, boolean, choice, environment types
- **Testing-friendly**: Perfect for workflow development
- **On-demand execution**: No waiting for events or schedules

### Input Types

```yaml
inputs:
  # Free-form text
  query:
    type: string
    required: true
  
  # True/false checkbox
  include_examples:
    type: boolean
    default: true
  
  # Dropdown selection
  priority:
    type: choice
    options: [low, medium, high]
    default: medium
  
  # Environment selector
  deploy_env:
    type: environment
```

### Running Workflows

```bash
# Via CLI
gh aw run workflow-name --raw-field topic="AI safety"

# With multiple inputs
gh aw run research \
  --raw-field topic="quantum computing" \
  --raw-field priority=high

# Wait for completion
gh aw run research --raw-field topic="AI" --wait

# Specific branch
gh aw run research --ref feature-branch
```

### Conditional Logic

```markdown
{{#if (eq github.event.inputs.include_code "true")}}
Include code snippets in analysis.
{{else}}
Describe patterns without code.
{{/if}}
```

### Security Considerations

- Respects same role-based access as other triggers
- Use `roles:` to restrict sensitive operations
- Use `manual-approval:` for production workflows
- Validate and sanitize all string inputs
- Never pass secrets via inputs

### Development Pattern

Add `workflow_dispatch` to all workflows during development for testing:

```yaml
on:
  issues:
    types: [opened]
  workflow_dispatch:  # For testing without creating real issues
```

### Real-World Examples

- Research workflows - On-demand topic research
- Operational commands - Manual cleanup, sync, audit tasks
- Testing harness - Test event-triggered workflows without events

---

## LabelOps Pattern

**Trigger**: Label events (labeled, unlabeled)  
**Use Case**: Label-triggered automation  
**Example Scenarios**: Status changes, workflow routing, categorization

### Pattern Structure

```aw
---
on:
  issues:
    types: [labeled]
  pull_request:
    types: [labeled]
permissions:
  contents: read
safe-outputs:
  add-comment:
---

# Label Handler

Handle label change: ${{ github.event.label.name }}

Respond based on label type and update workflow accordingly.
```

### Key Characteristics

- **Label-triggered**: Activates when labels are added/removed
- **Workflow routing**: Different actions for different labels
- **Status management**: Track issue/PR lifecycle
- **Team coordination**: Signal different workflows

### Common Use Cases

```yaml
# Handle specific labels
on:
  issues:
    types: [labeled]

# Conditional logic based on label
```

In workflow body:

```markdown
{{#if (contains github.event.label.name "bug")}}
Process as bug report: validate, assign, prioritize.
{{else if (contains github.event.label.name "enhancement")}}
Process as feature request: gather requirements, estimate.
{{/if}}
```

### Real-World Examples

- Status transitions - Move issues through workflow stages
- Assignment automation - Auto-assign based on labels
- Priority routing - Different workflows for different priorities

---

## ProjectOps Pattern

**Trigger**: Various (issues, PRs, schedule)  
**Use Case**: GitHub Projects board automation  
**Example Scenarios**: Status updates, card management, field updates

### Pattern Structure

```aw
---
on:
  issues:
    types: [opened, closed]
  pull_request:
    types: [opened, closed, merged]
permissions:
  contents: read
safe-outputs:
  update-project:
    max: 20
    github-token: ${{ secrets.PROJECTS_PAT }}
---

# Project Board Manager

Manage project board based on issue/PR events.

Update status, fields, and card positions automatically.
```

### Key Characteristics

- **Board automation**: Updates project cards and fields
- **Status management**: Moves cards through columns
- **Multi-item support**: Can update multiple items per run
- **Requires PAT**: Needs token with projects:write scope

### Project Operations

```json
// Add existing issue/PR to project
{"type": "update_project", "project": "https://github.com/orgs/myorg/projects/42", "content_type": "issue", "content_number": 123, "fields": {"Status": "In Progress"}}

// Create draft issue in project
{"type": "update_project", "project": "https://github.com/orgs/myorg/projects/42", "content_type": "draft_issue", "draft_title": "Task", "draft_body": "Description", "fields": {"Status": "Todo"}}
```

**Important**: Use full project URL (e.g., `https://github.com/orgs/myorg/projects/42`), not just project name or number.

### Security

- Requires Personal Access Token (PAT) with `projects:write` scope
- Store PAT as repository secret
- Not supported for cross-repository operations

### Real-World Examples

- Status automation - Move cards when issues/PRs change state
- Sprint planning - Auto-populate project boards with new work
- Progress tracking - Update custom fields based on activity

---

## MultiRepoOps Pattern

**Trigger**: Various (schedule, workflow_dispatch)  
**Use Case**: Cross-repository automation  
**Example Scenarios**: Organization-wide reports, dependency updates, policy enforcement

### Pattern Structure

```aw
---
on:
  schedule: weekly
permissions:
  contents: read
  actions: read
tools:
  github:
    toolsets: [default, repos, search]
safe-outputs:
  create-discussion:
network:
  allowed:
    - defaults
    - github
---

# Multi-Repo Analyzer

Analyze repositories across the organization.

Use GitHub tools to search repos, analyze patterns, create summary.
```

### Key Characteristics

- **Organization-wide**: Operates across multiple repositories
- **GitHub tools**: Uses `github` toolsets for cross-repo access
- **Read-only by default**: Respects repository permissions
- **Centralized reporting**: Creates summaries in main repo

### GitHub Tool Configuration

```yaml
tools:
  github:
    toolsets: [default, repos, search]  # Enable search across repos
    read-only: true
```

### Cross-Repository Safe Outputs

```yaml
safe-outputs:
  create-issue:
    target-repo: "owner/other-repo"
  add-comment:
    target-repo: "owner/other-repo"
```

### Best Practices

- Use `read-only: true` for GitHub tools when only reading
- Request minimal permissions needed
- Rate limit API calls to avoid throttling
- Use search tools efficiently
- Cache results with `cache-memory`

### Real-World Examples

- Organization reports - Weekly activity summaries across all repos
- Dependency scanning - Check for outdated deps across org
- Policy compliance - Verify all repos meet standards

---

## SideRepoOps Pattern

**Trigger**: Various  
**Use Case**: Isolated experimentation in side repositories  
**Example Scenarios**: Testing workflows, experiments, sandboxed development

### Pattern Structure

```aw
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
safe-outputs:
  create-issue:
    target-repo: "user/test-repo"
  create-pull-request:
    target-repo: "user/test-repo"
---

# Side Repo Experiment

Experiment with workflow in isolated side repository.

All outputs go to target-repo, keeping main repo clean.
```

### Key Characteristics

- **Isolated testing**: Workflow runs in main repo, outputs go to side repo
- **Safe experimentation**: No impact on production repo
- **Cross-repo outputs**: Uses `target-repo` field
- **Full functionality**: Test all safe-outputs features

### Use Cases

- Workflow development - Test changes without affecting main repo
- Training environments - Learn workflow patterns safely
- Demos and examples - Show capabilities without cluttering real repo

### Real-World Examples

- Testing new workflow patterns before production deployment
- Demo workflows for documentation and tutorials
- Training repositories for team onboarding

---

## TrialOps Pattern

**Trigger**: CLI command (`gh aw trial`)  
**Use Case**: Local workflow testing before deployment  
**Example Scenarios**: Development, debugging, validation

### Pattern Structure

```bash
# Test workflow locally without committing
gh aw trial ./workflow.md --raw-field topic="test"

# Test with staged mode (safe-outputs dry-run)
gh aw trial ./workflow.md --staged

# Test with specific inputs
gh aw trial ./research.md \
  --raw-field topic="quantum computing" \
  --raw-field depth=detailed
```

### Key Characteristics

- **Local execution**: Runs on your machine, not GitHub Actions
- **No GitHub impact**: Safe-outputs are dry-run only
- **Fast iteration**: Test changes immediately
- **Full debugging**: Access to local logs and state

### Workflow

1. Edit workflow markdown file
2. Run `gh aw trial ./workflow.md` with test inputs
3. Review output and behavior
4. Iterate on prompt and configuration
5. Commit when satisfied

### Staged Mode

The `🎭` emoji marks preview mode across all safe output types:

```yaml
safe-outputs:
  create-issue:  # Shows 🎭 in title during trial
  add-comment:   # Shows 🎭 in body during trial
```

### Real-World Examples

- Workflow development - Test prompts and logic locally
- Debugging - Investigate issues without affecting production
- Validation - Verify changes before creating PR

---

## Pattern Selection Guide

### Choose ChatOps When:
- ✅ Need human judgment on timing
- ✅ Workflow requires context from conversation
- ✅ Interactive assistance is valuable
- ✅ Multiple team members need access to automation

### Choose IssueOps When:
- ✅ Fully automated response is appropriate
- ✅ Immediate triage/categorization needed
- ✅ Initial response can be templated
- ✅ Want to reduce human response time

### Choose DailyOps When:
- ✅ Large goal can be broken into small pieces
- ✅ Changes should be incremental and reviewable
- ✅ Progress tracking across runs is important
- ✅ Work doesn't need immediate execution

### Choose DispatchOps When:
- ✅ Timing requires human judgment
- ✅ Testing workflows during development
- ✅ On-demand operational tasks
- ✅ Research or analysis needed occasionally

### Choose LabelOps When:
- ✅ Labels indicate state transitions
- ✅ Different workflows for different categories
- ✅ Team uses labels for coordination
- ✅ Status tracking through labels

### Choose ProjectOps When:
- ✅ Using GitHub Projects for planning
- ✅ Need to update board status automatically
- ✅ Tracking work across multiple repos
- ✅ Custom fields need automation

### Choose MultiRepoOps When:
- ✅ Analysis spans multiple repositories
- ✅ Organization-wide consistency needed
- ✅ Centralized reporting required
- ✅ Cross-repo coordination beneficial

### Choose SideRepoOps When:
- ✅ Testing new patterns before production
- ✅ Learning workflow capabilities
- ✅ Creating demos or examples
- ✅ Want isolated experimentation

### Choose TrialOps When:
- ✅ Developing new workflows locally
- ✅ Debugging workflow issues
- ✅ Testing prompt changes rapidly
- ✅ Validating before deployment

---

## Anti-Patterns to Avoid

### ❌ Granting Unnecessary Permissions

```yaml
# BAD: Too many permissions
permissions:
  contents: write
  issues: write
  pull-requests: write
  discussions: write
```

**Instead**: Use minimal permissions + safe-outputs:

```yaml
# GOOD: Minimal permissions
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
  add-comment:
```

### ❌ Fixed Cron Schedules

```yaml
# BAD: All workflows at same time
on:
  schedule:
    - cron: "0 0 * * *"
```

**Instead**: Use fuzzy scheduling:

```yaml
# GOOD: Compiler distributes times
on:
  schedule: daily
```

### ❌ Raw Event Fields

```markdown
<!-- BAD: Unsanitized user input -->
Analyze this: "${{ github.event.issue.title }}"
```

**Instead**: Use sanitized context:

```markdown
<!-- GOOD: Sanitized context -->
Analyze this: "${{ needs.activation.outputs.text }}"
```

### ❌ Verbose Frontmatter

```yaml
# BAD: Unnecessary fields
---
engine: copilot
timeout-minutes: 30
on:
  issues:
    types: [opened]
  workflow_dispatch:
---
```

**Instead**: Omit defaults:

```yaml
# GOOD: Minimal frontmatter
---
on:
  issues:
    types: [opened]
---
```

### ❌ Unrestricted Fork Access

```yaml
# BAD: Allow all forks
on:
  pull_request:
    forks: ["*"]
```

**Instead**: Default block or specific allowlist:

```yaml
# GOOD: Block forks (default)
on:
  pull_request:
    types: [opened]

# OR: Specific allowlist
on:
  pull_request:
    forks: ["trusted-org/*"]
```

### ❌ Manual GitHub Tool Lists

```yaml
# BAD: Long tool list
tools:
  github:
    allowed:
      - get_repository
      - list_commits
      - get_commit
      - search_issues
      - list_issues
      - get_issue
```

**Instead**: Use toolsets:

```yaml
# GOOD: Toolset grouping
tools:
  github:
    toolsets: [default]
```

---

## Quick Reference

### Essential Safe-Outputs

| Operation | Use For | Key Config |
|-----------|---------|------------|
| `add-comment` | Respond to issues/PRs | `max`, `target` |
| `add-labels` | Categorize issues/PRs | `allowed`, `max` |
| `create-issue` | Track work, bugs, features | `title-prefix`, `labels`, `expires` |
| `create-discussion` | Reports, status, audits | `category`, `close-older-discussions` |
| `create-pull-request` | Code changes | `draft`, `reviewers`, `skip-if-match` |
| `update-project` | Project board automation | `github-token` (PAT) |

### Essential Tools

| Tool | Use For | Common Config |
|------|---------|---------------|
| `github` | GitHub API access | `toolsets: [default]` |
| `edit` | File editing | Auto-enabled with sandbox |
| `bash` | Shell commands | Auto-enabled with sandbox |
| `web-fetch` | Fetch web content | `network: allowed: [...]` |
| `web-search` | Search web | `network: allowed: [...]` |
| `playwright` | Browser automation | `network: allowed: [...]` |
| `cache-memory` | Persistent state | `cache-memory: true` |

### Common Triggers

| Trigger | When It Runs | Typical Use |
|---------|--------------|-------------|
| `issues: [opened]` | New issue created | Auto-triage, initial response |
| `pull_request: [opened]` | New PR created | Auto-review, checks |
| `slash_command:` | /command in comments | Interactive assistance |
| `schedule: daily` | Daily (fuzzy time) | Incremental improvements |
| `workflow_dispatch:` | Manual via UI/CLI | Testing, on-demand tasks |
| `issues: [labeled]` | Label added to issue | Status transitions |

---

## Additional Resources

- **[GitHub Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)** - Complete documentation
- **[.github/aw/github-agentic-workflows.md](github-agentic-workflows.md)** - Full schema reference
- **[.github/aw/create.md](create.md)** - Workflow creation guide
- **[Real Workflows](.github/workflows/)** - Production examples in this repository

---

**Last Updated**: 2026-01-12
