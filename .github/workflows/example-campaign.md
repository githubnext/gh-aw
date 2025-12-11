---
name: Example Campaign - Issue Cleanup
description: Example showing campaign pattern - analyze, create tracked work, store results
timeout-minutes: 15
strict: true

on:
  workflow_dispatch:
    inputs:
      max_issues:
        description: 'Maximum issues to close'
        required: false
        default: '20'

permissions:
  contents: read
  issues: read

engine: copilot

# Campaign pattern uses standard primitives:
# 1. tracker-id: Links all work (set via label in epic issue)
# 2. repo-memory: Stores baseline, results, learnings
# 3. safe-outputs: Coordinates actions

tools:
  github:
    toolsets: [repos, issues]
  repo-memory:
    branch-name: memory/campaigns
    file-glob: "issue-cleanup-*/**"

safe-outputs:
  create-issue:
    max: 1
    labels: [campaign-tracker]
  close-issue:
    max: 30
  add-comment:
    max: 30
---

# Issue Cleanup Campaign Example

This demonstrates the **campaign pattern**: using tracker-id + repo-memory + coordination for stateful initiatives.

**Campaign ID**: `issue-cleanup-${{ github.run_id }}`

## Campaign Pattern Steps

### 1. Store Baseline (repo-memory)

Create file `memory/campaigns/issue-cleanup-${{ github.run_id }}/baseline.json`:
```json
{
  "campaign_id": "issue-cleanup-${{ github.run_id }}",
  "started": "[current date]",
  "stale_issues_found": [count],
  "goal": "Close stale issues with no activity >6 months"
}
```

### 2. Create Tracking Issue

Use `create-issue` to create ONE epic issue with **tracker-id label**:

**Title**: "Campaign: Issue Cleanup - ${{ github.run_id }}"

**Labels**: `campaign-tracker`, `tracker:issue-cleanup-${{ github.run_id }}`

**Body**:
```markdown
# Issue Cleanup Campaign

**Campaign ID**: `issue-cleanup-${{ github.run_id }}`
**Tracker ID**: `issue-cleanup-${{ github.run_id }}`
**Started**: [current date]

## Goal
Close up to ${{ github.event.inputs.max_issues }} stale issues (>6 months inactive)

## Baseline
- Stale issues found: [count]
- Target to close: ${{ github.event.inputs.max_issues }}

## Success Criteria
- Issues closed with clear explanation
- No false positives (keep-open labeled issues preserved)

## Progress
This issue will be updated as work completes.

**Query campaign work**:
```bash
gh issue list --search "tracker:issue-cleanup-${{ github.run_id }}"
```

**View campaign data**:
```bash
gh repo view --json defaultBranchRef \
  --jq '.defaultBranchRef.target.tree.entries[] | 
  select(.name=="memory") | .oid' | \
  xargs -I {} gh api repos/{owner}/{repo}/git/trees/{} --jq '.tree'
```
```

### 3. Execute Work

Find and close stale issues:

**For each stale issue** (no activity >6 months, not labeled "keep-open"):
1. Add comment explaining closure:
   ```
   This issue is being closed as part of cleanup campaign [issue-cleanup-${{ github.run_id }}].
   
   Reason: No activity in >6 months
   
   If this should remain open, please reopen and add the "keep-open" label.
   
   Campaign tracker: #[epic-issue-number]
   ```

2. Apply label: `closed-by-campaign`
3. Close issue with reason: "not_planned"

**Max ${{ github.event.inputs.max_issues }} issues processed**

### 4. Store Results (repo-memory)

Create file `memory/campaigns/issue-cleanup-${{ github.run_id }}/results.json`:
```json
{
  "campaign_id": "issue-cleanup-${{ github.run_id }}",
  "completed": "[current date]",
  "baseline_count": [count],
  "closed_count": [actual count],
  "duration_minutes": [time taken],
  "issues_closed": [list of issue numbers]
}
```

### 5. Update Tracking Issue

Add final comment to epic issue:

```markdown
## Campaign Complete ✅

**Duration**: [X] minutes
**Results**:
- Stale issues found: [count]
- Issues closed: [count]
- Remaining stale: [count]

**Data stored**: `memory/campaigns/issue-cleanup-${{ github.run_id }}/`

This campaign demonstrates the pattern:
1. ✅ tracker-id links all work
2. ✅ repo-memory stores baseline + results
3. ✅ Epic issue provides visibility
4. ✅ Safe-outputs coordinate actions

Campaign closed.
```

Close the epic issue.

## Campaign Pattern Explained

**This example shows how campaigns work without special infrastructure:**

- **tracker-id**: `issue-cleanup-${{ github.run_id }}` links everything
- **repo-memory**: Stores baseline.json + results.json for history
- **Epic issue**: Human-visible tracking and reporting
- **Safe-outputs**: Coordinates issue creation, comments, closures

**For longer campaigns**, add:
- Separate worker workflows (trigger on labels)
- Scheduled monitor workflow (daily progress reports)
- Learnings file in repo-memory (what worked/didn't)

See [Campaign Guide](/gh-aw/guides/campaigns/) for multi-workflow patterns.

## Output

Provide summary:
- Campaign ID: `issue-cleanup-${{ github.run_id }}`
- Epic issue: #[number]
- Issues closed: [count]
- Memory location: `memory/campaigns/issue-cleanup-${{ github.run_id }}/`
- Query: `gh issue list --search "tracker:issue-cleanup-${{ github.run_id }}"`
