# Campaign Generator Workflow Skip Analysis

**Investigation Date:** December 17, 2025  
**Workflow Run:** https://github.com/githubnext/gh-aw/actions/runs/20301688253  
**Related Issue:** #6721 (deleted)  
**Related PR:** #6722

## Executive Summary

The campaign generator workflow was skipped due to a timing mismatch between when GitHub applies labels from issue forms and when the workflow's label check is evaluated. While PR #6722 added support for the `labeled` event type, it did not eliminate the initial skip that occurs on the `opened` event, resulting in the workflow running twice (once skipped, once successful) for each issue creation.

## Background

### GitHub Issue Forms Label Timing

When a user submits an issue using a GitHub issue form:

1. **Step 1**: GitHub creates the issue and fires the `opened` event
2. **Step 2**: GitHub applies labels specified in the form (fires `labeled` event for each label)

There is a brief delay between steps 1 and 2, meaning labels are **not present** when the `opened` event fires.

### Campaign Generator Workflow Configuration

The campaign generator workflow `.github/workflows/campaign-generator.md` has:

```yaml
on:
  issues:
    types: [opened, labeled]
if: contains(github.event.issue.labels.*.name, 'campaign')
```

## Root Cause Analysis

### The Compilation Process

The workflow compiler (`pkg/workflow/compiler_jobs.go`) takes the top-level `if` condition and applies it to the `pre_activation` job:

**Source Code (lines 901-911):**
```go
// Pre-activation job uses the user's original if condition (data.If)
// The workflow_run safety check is NOT applied here - it's only on the activation job
// Don't include conditions that reference custom job outputs (those belong on the agent job)
var jobIfCondition string
if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
    jobIfCondition = data.If
}

job := &Job{
    Name:        constants.PreActivationJobName,
    If:          jobIfCondition,  // Applied here
    RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
    Permissions: permissions,
    Steps:       steps,
    Outputs:     outputs,
}
```

**Compiled Result (`.github/workflows/campaign-generator.lock.yml` line 8727):**
```yaml
pre_activation:
  if: contains(github.event.issue.labels.*.name, 'campaign')
  runs-on: ubuntu-slim
  outputs:
    activated: ${{ steps.check_membership.outputs.is_team_member == 'true' }}
```

### The Problem

The `pre_activation` job is the **first job** in the workflow chain. If its `if` condition evaluates to `false`, the entire workflow is skipped because all subsequent jobs depend on it (directly or indirectly).

### Execution Flow

#### Scenario 1: Issue Opened Event (Label Not Yet Applied)

```
User submits issue form
    ↓
GitHub creates issue (opened event) ← No 'campaign' label yet
    ↓
Campaign generator triggers
    ↓
pre_activation if condition evaluates:
  contains(github.event.issue.labels.*.name, 'campaign') → FALSE
    ↓
pre_activation job SKIPPED
    ↓
activation job SKIPPED (needs pre_activation)
    ↓
agent job SKIPPED (needs activation)
    ↓
All safe output jobs SKIPPED
    ↓
Workflow conclusion: SKIPPED
```

#### Scenario 2: Label Applied Event

```
GitHub applies 'campaign' label (labeled event)
    ↓
Campaign generator triggers AGAIN
    ↓
pre_activation if condition evaluates:
  contains(github.event.issue.labels.*.name, 'campaign') → TRUE
    ↓
pre_activation job RUNS
    ↓
Team membership check passes
    ↓
activation job RUNS
    ↓
agent job RUNS
    ↓
Safe output jobs RUN
    ↓
Workflow conclusion: SUCCESS
```

## Impact Assessment

### User Experience
- ❌ Initial workflow run appears as "skipped" - confusing
- ❌ Delay between issue creation and workflow execution
- ✅ Eventually runs successfully when label is applied

### Resource Usage
- ❌ Consumes GitHub Actions minutes for skipped run
- ❌ Creates unnecessary workflow run entries
- ❌ Pollutes workflow run history

### Technical Debt
- ⚠️ Workflow triggers twice for single user action
- ⚠️ Hidden complexity in label timing dependency
- ⚠️ Non-obvious behavior for workflow maintainers

## Why PR #6722 Was Incomplete

PR #6722 added the `labeled` event type:

```yaml
on:
  issues:
    types: [opened, labeled]  # Added 'labeled'
```

This **does** fix the original problem where the workflow never ran at all (when only `opened` was configured). However, it doesn't address the inefficiency and confusion of:

1. First run being skipped (wasted resources)
2. Second run succeeding (actual execution)
3. Two workflow runs appearing for one issue

## Solution Options

### Option 1: Event-Specific Conditional (Recommended)

Modify the `if` condition to check both the event type and the label:

```yaml
if: |
  (github.event.action == 'labeled' && contains(github.event.issue.labels.*.name, 'campaign')) ||
  (github.event.action == 'opened')
```

**Pros:**
- Single workflow run per issue
- Allows pre-activation to run on `opened` event
- Label check still enforced before main workflow runs

**Cons:**
- More complex condition
- Requires compiler support for event-specific conditionals in activation logic

### Option 2: Remove Top-Level If, Add Job-Level If on Activation

Remove the top-level `if` and add it only to the `activation` job:

**In `.md` file:**
```yaml
on:
  issues:
    types: [opened, labeled]
# Remove: if: contains(github.event.issue.labels.*.name, 'campaign')
```

**Modify compiler to add label check to activation job's if condition** (after pre-activation passes).

**Pros:**
- Pre-activation always runs (checks team membership)
- Label check happens before expensive operations
- Clear separation of concerns

**Cons:**
- Requires compiler changes
- Changes workflow architecture pattern

### Option 3: Only Trigger on labeled Event

Simplify to only trigger on the `labeled` event:

```yaml
on:
  issues:
    types: [labeled]
if: contains(github.event.issue.labels.*.name, 'campaign')
```

**Pros:**
- Simple and clear
- No wasted runs
- Label is always present when workflow runs

**Cons:**
- ⚠️ **Won't work for issues created via API/manually with label already applied**
- ⚠️ **Won't re-run if label is removed and re-added**
- Less flexible

### Option 4: Move Label Check to Activation Job (Cleanest Architecture)

The top-level `if` should be reserved for basic trigger filtering. The label check should be part of the activation logic since it's a runtime condition.

**Recommended Architecture:**

```yaml
# Top-level: Only basic event filtering
on:
  issues:
    types: [opened, labeled]

# No top-level if condition
```

**In Compiler:** Add label check as part of activation job's condition (after team membership is verified).

**Pros:**
- Clean separation: pre-activation handles auth, activation handles business logic
- No wasted runs
- More maintainable

**Cons:**
- Requires careful compiler changes
- Breaking change for existing workflows

## Recommendation

**Option 1 (Event-Specific Conditional)** is the quickest fix with minimal architectural changes. However, **Option 4 (Move Label Check to Activation Job)** is the cleanest long-term solution that properly separates concerns.

### Immediate Fix (Option 1)

Modify `campaign-generator.md`:

```yaml
if: |
  github.event.action == 'labeled' && contains(github.event.issue.labels.*.name, 'campaign')
```

Remove `opened` from the event types since we only want to run when the label is actually applied:

```yaml
on:
  issues:
    types: [labeled]
```

This ensures the workflow only triggers when it can actually run successfully.

### Long-Term Fix (Option 4)

1. Modify compiler to distinguish between "trigger filtering" and "runtime conditions"
2. Apply top-level `if` conditions at the activation job level (not pre-activation)
3. Keep pre-activation focused only on authorization and safety checks
4. This makes the architecture more intuitive and reduces confusion

## Testing Recommendations

### Test Case 1: Issue Form Submission
1. Create issue using issue form with 'campaign' label selected
2. Verify workflow triggers exactly once
3. Verify workflow runs successfully (not skipped)

### Test Case 2: Manual Issue with Label
1. Create issue manually
2. Add 'campaign' label
3. Verify workflow triggers once
4. Verify workflow runs successfully

### Test Case 3: Issue Without Campaign Label
1. Create issue with different label
2. Verify workflow does not trigger (or triggers and skips appropriately)

### Test Case 4: Label Removed and Re-added
1. Create issue with 'campaign' label
2. Remove label
3. Re-add label
4. Verify workflow triggers on re-add

## Conclusion

The campaign generator workflow skip issue is a consequence of GitHub's issue form label timing and how the workflow compiler applies top-level `if` conditions to the `pre_activation` job. While PR #6722 made the workflow eventually run (via the `labeled` event), it doesn't eliminate the initial skip on the `opened` event.

The recommended immediate fix is to only trigger on the `labeled` event, ensuring labels are always present when the workflow runs. The long-term architectural improvement is to move runtime conditions (like label checks) from `pre_activation` to `activation`, keeping authorization concerns separate from business logic conditions.

## References

- **Workflow Run:** https://github.com/githubnext/gh-aw/actions/runs/20301688253
- **PR #6722:** https://github.com/githubnext/gh-aw/pull/6722
- **Compiler Source:** `pkg/workflow/compiler_jobs.go` (lines 768-919)
- **Workflow Source:** `.github/workflows/campaign-generator.md`
- **Compiled Workflow:** `.github/workflows/campaign-generator.lock.yml`

## Appendix: Workflow Run Details

**Run ID:** 20301688253  
**Name:** Campaign Generator  
**Conclusion:** skipped  
**Event:** issues  
**Head SHA:** 03897cc0f4c806ccb6bdfd9d6adec04378137c51  
**Created:** 2025-12-17T11:44:49Z  

**All Jobs Skipped:**
- `pre_activation` - skipped
- `activation` - skipped
- `agent` - skipped
- `detection` - skipped
- `add_comment` - skipped
- `create_pull_request` - skipped
- `conclusion` - skipped
