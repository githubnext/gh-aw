---
name: Campaign - Org-Wide Rollout
description: Coordinate changes across 100+ repos with phased rollout, dependency tracking, and rollback
timeout-minutes: 480  # 8 hours for large orgs
strict: true

on:
  workflow_dispatch:
    inputs:
      rollout_type:
        description: 'Type of rollout'
        type: choice
        required: true
        options:
          - dependency-upgrade
          - policy-enforcement
          - tooling-migration
          - security-hardening
      target_repos:
        description: 'Repo pattern (e.g., "githubnext/*" or comma-separated list)'
        required: true
        default: 'githubnext/*'
      change_description:
        description: 'What is being rolled out?'
        required: true
      batch_size:
        description: 'Number of repos per batch'
        required: false
        default: '10'
      approval_required:
        description: 'Require approval between batches?'
        required: false
        default: 'true'

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot

tools:
  github:
    toolsets: [repos, issues, pull_requests, search]
  repo-memory:
    branch-name: memory/campaigns
    file-glob: "memory/campaigns/org-rollout-*/**"

safe-outputs:
  create-issue:
    labels: [campaign-tracker, org-rollout]
  create-pull-request:
    labels: [campaign-pr, org-rollout]
  add-comment: {}
  add-labels: {}
---

# Campaign: Organization-Wide Rollout

**Purpose**: Coordinate changes across 100+ repositories that GitHub Actions cannot orchestrate.

**Campaign ID**: `org-rollout-${{ github.run_id }}`

**Rollout Type**: `${{ github.event.inputs.rollout_type }}`

## Why Campaigns Solve This (Not GitHub Actions)

**Problem**: "Update all 200 repos from Node 18 → Node 20" or "Add CODEOWNERS to all repos" requires:
- Discovery of all affected repos
- Dependency analysis (which repos depend on which)
- Phased rollout (batch 1 → test → batch 2)
- Per-repo tracking (success/failure/blocked)
- Rollback capability if failures exceed threshold
- Cross-repo coordination (wait for dependencies)
- Executive reporting (progress dashboard)

**GitHub Actions fails**: Each repo runs independently, no cross-repo orchestration, no phased rollout, no dependency awareness

**Basic agentic workflow fails**: Single repo context, no multi-repo coordination, no progress tracking across org

**Campaign solves**: Central orchestration + phased execution + dependency tracking + rollback + reporting

## Rollout Phases

### Phase 1: Discovery & Planning

#### 1. Discover Target Repos

**Query GitHub** for repos matching `${{ github.event.inputs.target_repos }}`:
- If pattern like "githubnext/*": List all org repos
- If comma-separated: Parse specific repos
- Filter: Active repos (not archived), with required access

**Store discovery** in `memory/campaigns/org-rollout-${{ github.run_id }}/discovery.json`:
```json
{
  "campaign_id": "org-rollout-${{ github.run_id }}",
  "started": "[timestamp]",
  "rollout_type": "${{ github.event.inputs.rollout_type }}",
  "change_description": "${{ github.event.inputs.change_description }}",
  "target_pattern": "${{ github.event.inputs.target_repos }}",
  "discovered_repos": [
    {"name": "repo1", "language": "JavaScript", "stars": 150, "active": true},
    {"name": "repo2", "language": "Python", "stars": 89, "active": true}
  ],
  "total_repos": 147,
  "batch_size": ${{ github.event.inputs.batch_size }},
  "total_batches": 15
}
```

#### 2. Analyze Dependencies

**For each discovered repo**:
- Check if it's a dependency of other repos (package.json, go.mod, etc.)
- Check if it depends on other discovered repos
- Build dependency graph

**Store dependency graph** in `memory/campaigns/org-rollout-${{ github.run_id }}/dependencies.json`:
```json
{
  "dependency_graph": {
    "repo1": {
      "depends_on": [],
      "depended_by": ["repo3", "repo5"],
      "priority": "high"
    },
    "repo2": {
      "depends_on": ["repo1"],
      "depended_by": [],
      "priority": "normal"
    }
  },
  "rollout_order": [
    "batch1": ["repo1", "repo7", "repo12"],  // No dependencies
    "batch2": ["repo2", "repo8"],  // Depend on batch1
    "batch3": ["repo3", "repo4"]   // Depend on batch2
  ]
}
```

#### 3. Create Command Center Issue

Use `create-issue`:

**Title**: `🚀 ORG ROLLOUT: ${{ github.event.inputs.change_description }}`

**Labels**: `campaign-tracker`, `tracker:org-rollout-${{ github.run_id }}`, `org-rollout`, `type:${{ github.event.inputs.rollout_type }}`

**Body**:
```markdown
# Org-Wide Rollout Campaign

**Campaign ID**: `org-rollout-${{ github.run_id }}`
**Type**: ${{ github.event.inputs.rollout_type }}
**Change**: ${{ github.event.inputs.change_description }}

## Scope

**Target Pattern**: `${{ github.event.inputs.target_repos }}`
**Discovered Repos**: [count]
**Total Batches**: [count] (batch size: ${{ github.event.inputs.batch_size }})
**Approval Required**: ${{ github.event.inputs.approval_required }}

## Rollout Strategy

**Phased Rollout** (respecting dependencies):
1. **Batch 1**: Core dependencies (repos with no dependencies)
2. **Batch 2-N**: Dependent repos (after dependencies succeed)

**Success Criteria** (per batch):
- ✅ All PRs created successfully
- ✅ 90% of PRs merged without conflicts
- ✅ No critical failures in CI/tests
- ✅ Rollback threshold not exceeded (<10% failures)

**Rollback Trigger**:
- If >10% of batch fails CI/tests
- If critical dependency breaks
- If manual intervention required

## Progress Dashboard

| Batch | Repos | Status | Success Rate | Issues |
|-------|-------|--------|--------------|--------|
| 1 | 10 | ✅ Complete | 100% (10/10) | 0 |
| 2 | 10 | 🔄 In Progress | 80% (8/10) | 2 blocked |
| 3 | 10 | ⏳ Waiting | - | - |

**Overall Progress**: [X]% ([Y]/[total] repos complete)

## Query Rollout Work

```bash
gh issue list --search "tracker:org-rollout-${{ github.run_id }}"
gh pr list --search "tracker:org-rollout-${{ github.run_id }}"
```

**Campaign Data**: `memory/campaigns/org-rollout-${{ github.run_id }}/`

---

**Updates will be posted after each batch completes**
```

### Phase 2: Batch Execution

#### 4. Execute Batch 1 (Foundational Repos)

**For each repo in Batch 1** (no dependencies):

1. **Create PR** with change:
   - Branch: `campaign/org-rollout-${{ github.run_id }}`
   - Title: `[${{ github.event.inputs.rollout_type }}] ${{ github.event.inputs.change_description }}`
   - Labels: `campaign-pr`, `tracker:org-rollout-${{ github.run_id }}`
   - Body:
     ```markdown
     ## Org-Wide Rollout
     
     **Campaign**: org-rollout-${{ github.run_id }}
     **Type**: ${{ github.event.inputs.rollout_type }}
     **Change**: ${{ github.event.inputs.change_description }}
     
     ## What Changed
     
     [AI-generated description of specific changes to this repo]
     
     ## Testing
     
     - [ ] CI/tests pass
     - [ ] Manual verification (if required)
     
     ## Rollout Status
     
     **Batch**: 1 of [total]
     **Dependent Repos**: [list repos that depend on this one]
     
     Command center: [link to command center issue]
     ```

2. **Create tracking issue** for repo:
   - Title: `[Rollout Tracking] ${{ github.event.inputs.change_description }} - [repo-name]`
   - Labels: `tracker:org-rollout-${{ github.run_id }}`, `repo-tracking`
   - Body: Link to PR, status, blockers

3. **Monitor PR status**:
   - Check CI/test results
   - Track merge status
   - Identify blockers

#### 5. Batch 1 Analysis

**After Batch 1 completes (all PRs merged or failed)**:

**Store batch results** in `memory/campaigns/org-rollout-${{ github.run_id }}/batch-1-results.json`:
```json
{
  "batch_number": 1,
  "completed": "[timestamp]",
  "repos": [
    {
      "name": "repo1",
      "pr_number": 123,
      "status": "merged",
      "ci_passed": true,
      "merge_time_hours": 2.5,
      "blockers": []
    },
    {
      "name": "repo2",
      "pr_number": 124,
      "status": "failed",
      "ci_passed": false,
      "error": "Test failures in auth module",
      "blockers": ["needs manual fix"]
    }
  ],
  "success_rate": 0.90,
  "total": 10,
  "succeeded": 9,
  "failed": 1,
  "avg_merge_time_hours": 3.2,
  "rollback_triggered": false
}
```

**Update command center** with batch results:

```markdown
## ✅ Batch 1 Complete

**Success Rate**: 90% (9/10 repos)
**Avg Time to Merge**: 3.2 hours
**Blockers**: 1 repo needs manual intervention

**Details**:
- ✅ repo1 - Merged in 2.5h
- ✅ repo2 - Merged in 1.8h
- ...
- ❌ repo10 - CI failed (auth test failures)

**Next**: Proceeding with Batch 2
```

#### 6. Human Approval Checkpoint (if enabled)

**If `${{ github.event.inputs.approval_required }}` is true**:

Add comment to command center:

```markdown
## 🚦 Approval Required: Batch 2

**Batch 1 Results**: 90% success (9/10)

**Batch 2 Plan**: 10 repos (depends on Batch 1 completions)

**Repos in Batch 2**:
- repo11 (depends on: repo1 ✅)
- repo12 (depends on: repo3 ✅)
- ...

**AI Recommendation**: ✅ Proceed - success rate above threshold

**Risks**:
- 1 repo in Batch 1 still failing (repo10)
- If repo10 is critical dependency, may impact Batch 2

---

**👤 Human Decision Required**

Reply with:
- "approve-batch-2" - Proceed with Batch 2
- "fix-batch-1" - Pause and fix failing repo10 first
- "rollback" - Revert all Batch 1 changes
- "adjust-batch-2 [repo-list]" - Modify which repos in Batch 2
```

**Pause execution until human responds**

#### 7. Execute Remaining Batches

Repeat steps 4-6 for each batch, respecting:
- Dependency order (batch N only starts after dependencies in batch N-1 succeed)
- Approval gates (if enabled)
- Rollback threshold (stop if >10% failure rate)

### Phase 3: Completion & Learning

#### 8. Final Summary

When all batches complete, add to command center:

```markdown
## 🎉 Rollout Complete

**Duration**: [X] hours
**Total Repos**: [count]
**Success Rate**: [Y]% ([Z] repos)

**Breakdown**:
- ✅ Succeeded: [count] repos
- ❌ Failed: [count] repos (require manual intervention)
- ⏭️ Skipped: [count] repos (dependencies failed)

**Timing**:
- Fastest merge: [repo] - [minutes]
- Slowest merge: [repo] - [hours]
- Average merge: [hours]

**Blockers Encountered**:
- [Blocker type A]: [count] repos
- [Blocker type B]: [count] repos

**Failed Repos** (need manual attention):
- repo10 - [reason] - [tracking issue]
- repo47 - [reason] - [tracking issue]

## Impact

**Repos Updated**: [count] / [total]
**Coverage**: [percentage]%
**Time Saved vs Manual**: ~[hours] (estimated [X] hours manual × [repos])

## Next Steps

- [ ] Address [count] failed repos manually
- [ ] Monitor deployed changes for [timeframe]
- [ ] Document learnings for future rollouts

See complete campaign data: `memory/campaigns/org-rollout-${{ github.run_id }}/`

---

Campaign closed.
```

#### 9. Generate Learnings

Create `memory/campaigns/org-rollout-${{ github.run_id }}/learnings.json`:
```json
{
  "campaign_id": "org-rollout-${{ github.run_id }}",
  "rollout_type": "${{ github.event.inputs.rollout_type }}",
  "total_repos": 147,
  "success_rate": 0.94,
  "duration_hours": 18.5,
  "learnings": {
    "what_worked_well": [
      "Dependency-aware batching prevented cascading failures",
      "Approval gates caught issues in Batch 3 before wider impact",
      "Automated PR creation saved ~40 hours of manual work"
    ],
    "what_went_wrong": [
      "Batch 5 had higher failure rate (15%) - auth module changes not tested",
      "Dependency detection missed some implicit dependencies",
      "PR merge time varied widely (1h - 8h) due to different CI configs"
    ],
    "common_blockers": {
      "ci_test_failures": 8,
      "merge_conflicts": 3,
      "missing_permissions": 2,
      "manual_review_required": 4
    },
    "recommendations_for_future": [
      "Add pre-rollout testing for common change types",
      "Improve dependency detection (analyze actual imports, not just manifests)",
      "Consider weekend/off-hours batches to reduce merge time variance",
      "Add automated rollback for failed batches"
    ]
  },
  "roi_estimate": {
    "time_saved_hours": 120,
    "cost_ai_dollars": 45,
    "cost_engineering_hours": 8,
    "roi_multiplier": "15x"
  }
}
```

## Why This Campaign Cannot Be Done Without Campaigns

**Cross-repo orchestration**: Coordinate 100+ repos from central command
**Dependency awareness**: Respect dependency graph in rollout order
**Phased execution**: Batch-by-batch with approval gates
**Rollback capability**: Automatic rollback if failure threshold exceeded
**Progress tracking**: Dashboard showing per-repo status across org
**Human-in-loop**: Approval gates between batches for risk management
**Learning capture**: What worked, what didn't, for future rollouts
**ROI tracking**: Time saved vs manual × cost

**GitHub Actions**: Each repo independent, no cross-repo orchestration, no phased rollout concept
**Basic workflows**: Single repo context, no multi-repo coordination, no progress aggregation

## Output

Provide summary:
- Campaign ID: `org-rollout-${{ github.run_id }}`
- Command center issue: #[number]
- Rollout type: ${{ github.event.inputs.rollout_type }}
- Total repos: [count]
- Success rate: [percentage]%
- Duration: [hours]
- Batches executed: [count]
- Approvals required: [count]
- Rollbacks triggered: [count]
- Memory location: `memory/campaigns/org-rollout-${{ github.run_id }}/`
- Learnings captured for future rollouts
