---
title: Sub-Issues
description: Organize work into hierarchical issue structures for GitHub Copilot agents using parent-child relationships
---

Sub-issues enable hierarchical organization of work items in GitHub. A parent issue represents a broader objective, while sub-issues break that objective into smaller, focused tasks—each suitable for assignment to a GitHub Copilot agent.

## Why Use Sub-Issues

Sub-issues transform large features into agent-ready tasks:

- **Right-size work** - Break epics into tasks that fit a single PR
- **Track progress** - Parent issues show completion status of all sub-issues
- **Reduce noise** - Filter out sub-issues from the main issues list using `no:parent-issue`
- **Maintain context** - Sub-issues link back to the broader objective
- **Enable parallel work** - Independent sub-issues can be assigned to multiple agents

## Creating Sub-Issues

Two approaches create sub-issues: inline during issue creation, or linking existing issues afterward.

### Method 1: Create with Parent Reference

Use `create-issue` with the `parent` field to create sub-issues linked to a parent issue in a single workflow run. This approach uses temporary IDs to reference parent issues before they exist.

```aw wrap
---
on:
  workflow_dispatch:
safe-outputs:
  create-issue:
    title-prefix: "[task] "
    labels: [task, ai-generated]
    max: 6
---

# Planning Workflow

Create a parent tracking issue, then create sub-issues linked to it.

## Step 1: Create Parent Issue

Create the parent issue first with a temporary ID:

{
  "type": "create_issue",
  "temporary_id": "aw_abc123def456",
  "title": "Implement authentication system",
  "body": "Tracking issue for auth implementation.\n\n## Tasks\n- Add middleware\n- Create endpoints\n- Add tests"
}

## Step 2: Create Sub-Issues

Create sub-issues referencing the parent's temporary ID:

{
  "type": "create_issue",
  "parent": "aw_abc123def456",
  "title": "Add authentication middleware",
  "body": "Create JWT middleware in src/middleware/auth.js"
}
```

The workflow automatically:
1. Creates the parent issue and records its actual number
2. Resolves `aw_abc123def456` to the real issue number (e.g., `#42`)
3. Creates sub-issues linked to that parent
4. Replaces any `#aw_abc123def456` references in issue bodies with `#42`

**Temporary ID Format**: `aw_` followed by 12 hexadecimal characters (e.g., `aw_abc123def456`).

### Method 2: Link Existing Issues

Use `link-sub-issue` to create parent-child relationships between existing issues. This approach works when issues already exist and need organization.

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * *"
safe-outputs:
  link-sub-issue:
    max: 10
---

# Issue Organizer

Analyze issues and link related ones as parent-child pairs.
```

The agent outputs:

```json
{
  "type": "link_sub_issue",
  "parent_issue_number": 42,
  "sub_issue_number": 45
}
```

**Configuration options** for `link-sub-issue`:

```yaml wrap
safe-outputs:
  link-sub-issue:
    parent-required-labels: [epic]       # Parent must have these labels
    parent-title-prefix: "[Epic]"        # Parent must match prefix
    sub-required-labels: [task]          # Sub-issue must have these labels
    sub-title-prefix: "[Task]"           # Sub-issue must match prefix
    max: 10                              # Maximum links per run
    target-repo: "owner/repo"            # Cross-repository linking
```

## Filtering Sub-Issues

GitHub's search supports the `no:parent-issue` filter to exclude sub-issues from results. Use this to keep the main issues list focused on top-level work.

**Filter sub-issues from the issues page**:

Navigate to `/issues?q=no:parent-issue` in your repository, or use:

```bash
gh issue list --search "no:parent-issue" --state open
```

This shows only parent issues and standalone issues, hiding all sub-issues.

**Combine with other filters**:

```bash
# Open issues without parents, excluding bot-created
gh issue list --search "no:parent-issue -label:ai-generated" --state open

# Only parent issues with the "epic" label
gh issue list --search "no:parent-issue label:epic" --state open
```

## Real-World Example: Plan Workflow

The [`plan.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/plan.md) workflow demonstrates creating parent-child issue structures:

```aw wrap
---
on:
  command:
    name: plan
    events: [issue_comment, discussion_comment]
safe-outputs:
  create-issue:
    title-prefix: "[plan] "
    labels: [plan, ai-generated]
    max: 6
---

# Planning Assistant

Analyze the issue or discussion and break it down:

1. Create a parent tracking issue with a temporary_id
2. Create sub-issues (max 5) as children of that parent
3. Each sub-issue should be completable in a single PR
```

**Workflow**:
1. User comments `/plan` on an issue or discussion
2. Agent analyzes the request and creates a parent tracking issue
3. Agent creates focused sub-issues linked to the parent
4. Each sub-issue is ready for assignment to GitHub Copilot

## Assigning Sub-Issues to Copilot

Sub-issues integrate naturally with the ResearchPlanAssign pattern. After creating structured sub-issues, assign them to GitHub Copilot for implementation:

```aw wrap
---
safe-outputs:
  create-issue:
    assignees: copilot
    max: 5
---
```

Or use `assign-to-agent` for existing issues:

```aw wrap
---
safe-outputs:
  assign-to-agent:
    name: "copilot"
---
```

Each sub-issue becomes an independent task that Copilot can work on in parallel, creating separate pull requests.

## Issue Arborist Pattern

The [`issue-arborist.md`](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/issue-arborist.md) workflow demonstrates automated issue organization:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * *"
steps:
  - name: Fetch issues data
    run: |
      gh issue list --repo ${{ github.repository }} \
        --search "no:parent-issue" \
        --state all \
        --json number,title,body,labels \
        --limit 100 \
        > /tmp/gh-aw/issues-data/issues.json
safe-outputs:
  link-sub-issue:
    max: 10
---

# Issue Arborist

Analyze the last 100 issues (excluding sub-issues) and identify
parent-child relationships to link.
```

The workflow:
1. Fetches issues excluding those already linked as sub-issues
2. Analyzes relationships between issues
3. Links related issues as parent-child pairs
4. Reports findings in a discussion

## Best Practices

**Right-size sub-issues** - Each sub-issue should represent work completable in a single pull request. Avoid tasks that are too large (need further breakdown) or too small (merge into parent).

**Clear acceptance criteria** - Sub-issues assigned to agents need unambiguous completion criteria. Include specific files, functions, and expected outcomes.

**Use meaningful prefixes** - Title prefixes like `[task]`, `[epic]`, or `[plan]` make filtering and identification easier.

**Limit sub-issue depth** - Keep hierarchies shallow (parent → sub-issue). Deep nesting complicates tracking and agent assignment.

**Reference parent context** - Sub-issue bodies should link to or summarize relevant parent issue context so agents have full information.

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Complete reference for `create-issue` and `link-sub-issue`
- [ResearchPlanAssign Strategy](/gh-aw/guides/researchplanassign/) - Pattern for structured work breakdown
- [Campaigns](/gh-aw/guides/campaigns/) - Coordinating multiple related issues
