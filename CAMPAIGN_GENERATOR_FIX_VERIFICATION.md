# Campaign Generator Workflow Skip - Fix Verification

**Date:** December 17, 2025  
**Analyzed Run:** https://github.com/githubnext/gh-aw/actions/runs/20305511262  
**Status:** ✅ **RESOLVED** - Fix already applied to main branch

## Executive Summary

The workflow run 20305511262 that was flagged for analysis represents the **last occurrence** of a timing bug that has since been fixed. The issue was identified, analyzed, and resolved through PR #6740, which was merged after this run occurred.

## Timeline

| Time (UTC) | Event | Status |
|------------|-------|--------|
| 14:02:31 | PR #6732 merged (analysis document created) | Analysis complete |
| 14:04:58 | Run 20305511262 triggered with OLD code | ❌ All steps skipped |
| 14:08:25 | PR #6740 created (fix implementation) | Fix in progress |
| 14:38:54 | PR #6740 merged to main | ✅ Fix deployed |

## The Bug (Now Fixed)

### Root Cause
GitHub applies labels from issue forms AFTER firing the `opened` event. The old workflow configuration triggered on both events:

```yaml
# OLD CONFIGURATION (before fix)
on:
  issues:
    types: [opened, labeled]
if: contains(github.event.issue.labels.*.name, 'campaign')
```

### Problem Flow
1. User submits issue form with 'campaign' label selected
2. GitHub creates issue and fires `opened` event (labels not yet applied)
3. Workflow triggers, `pre_activation` evaluates label check → **FALSE**
4. All jobs skipped due to failed condition
5. GitHub applies labels and fires `labeled` event
6. Workflow triggers again, label check → **TRUE**
7. Workflow runs successfully

**Result:** Two workflow runs per issue (one skipped, one successful)

## The Fix (Already Applied)

### Solution
Changed workflow to only trigger on `labeled` event:

```yaml
# NEW CONFIGURATION (current)
on:
  issues:
    types: [labeled]  # Removed 'opened'
if: contains(github.event.issue.labels.*.name, 'campaign')
```

### Benefits
✅ Single workflow run per issue  
✅ Label is always present when workflow triggers  
✅ No wasted GitHub Actions minutes  
✅ Cleaner workflow run history  

### Trade-off
⚠️ Issues created via API with labels pre-applied won't auto-trigger (remove/re-add label to trigger manually)

This is acceptable since issue forms are the primary use case.

## Verification

### Source File (`.github/workflows/campaign-generator.md`)
```yaml
on:
  issues:
    types: [labeled]  ✅ Fixed
    lock-for-agent: true
```

### Compiled Lock File (`.github/workflows/campaign-generator.lock.yml`)
```yaml
"on":
  issues:
    lock-for-agent: true
    types:
    - labeled  ✅ Fixed (line 233)
```

### Test Updated
The test file `pkg/workflow/campaign_trigger_test.go` was also updated to enforce the single-event trigger pattern.

## Related Documentation

- **Analysis Document:** `CAMPAIGN_GENERATOR_SKIP_ANALYSIS.md` (comprehensive root cause analysis)
- **Fix PR:** #6740 (implementation)
- **Analysis PR:** #6732 (investigation)
- **Affected Run:** 20305511262 (last occurrence before fix)

## Conclusion

The workflow skip issue in run 20305511262 was caused by a race condition between GitHub's issue form label application and workflow trigger evaluation. This has been **fully resolved** in the current main branch.

**No further action is required.** Future issue form submissions will trigger the workflow exactly once when labels are applied, with all steps executing successfully.

## Monitoring Recommendation

To verify the fix is working as expected in production:

1. Create a test issue using the campaign generator issue form
2. Verify only ONE workflow run is created (not two)
3. Verify all steps execute successfully (none skipped)
4. Confirm workflow triggers when label is applied, not when issue is opened
