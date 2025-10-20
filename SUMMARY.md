# Investigation Complete: `gh pr list --author "@copilot"`

## Problem Statement

> Investigate if this command lists the copilot PR (compare to full list):
> 
> gh pr list --author "@copilot"

## Answer

**YES**, the command `gh pr list --author "@copilot"` (or `--author "Copilot"`) successfully lists Copilot PRs.

However, it has important limitations compared to the current workflow approach.

## Quick Comparison

| Approach | Command | Max Results | Filtering | Best For |
|----------|---------|-------------|-----------|----------|
| **New** | `gh pr list --author "Copilot"` | 100 | Client-side | Small repos, ad-hoc queries |
| **Current** | `gh search prs ... \| jq ...` | 1000 | Server-side | Production, large repos |

## What We Discovered

### 1. Command Syntax ✅

All of these work:
```bash
gh pr list --author "@copilot"    # With @ prefix (like @me)
gh pr list --author "copilot"     # Lowercase
gh pr list --author "Copilot"     # Capitalized (matches bot login)
```

### 2. How It Works

`gh pr list --author` performs **client-side filtering**:
1. Fetches all PRs from the repository (up to limit)
2. Filters locally by author.login field
3. Returns matching results

This works for bot accounts because the filtering happens after fetching data, not at the API level.

### 3. Limitations

- ⚠️ **Maximum 100 results** (GitHub CLI limitation)
- ⚠️ **No server-side date filtering** (fetches all PRs first)
- ⚠️ **Less efficient** for large repositories

### 4. Current Workflow is Better for Production

The copilot-agent-analysis workflow uses:
```bash
gh search prs "repo:$REPO created:>=$DATE" --limit 1000 | \
  jq '[.[] | select(.author.login == "Copilot")]'
```

**Advantages:**
- ✅ Server-side date filtering (more efficient)
- ✅ Can fetch up to 1000 results
- ✅ Better for repositories with many PRs
- ✅ Handles 30+ days of data reliably

## Recommendation

**Keep the current approach** for the copilot-agent-analysis workflow, but document `gh pr list --author` as a simpler alternative for:
- Ad-hoc debugging
- Small repositories
- When you only need recent PRs (< 100)

## Testing

To validate this investigation, we created:

1. **Test Workflow** (`.github/workflows/test-copilot-pr-list.yml`)
   - Run with: `gh workflow run test-copilot-pr-list.yml`
   - Compares both approaches side-by-side
   - Validates results match

2. **Unit Tests** (`pkg/cli/gh_pr_list_test.go`)
   - Documents valid syntaxes
   - Explains the differences

## Documentation Updated

1. **Investigation Report** (`INVESTIGATION_REPORT.md`)
   - Detailed analysis
   - Comparison tables
   - Technical details

2. **Workflow Documentation** (`.github/workflows/copilot-agent-analysis.md`)
   - Added note about `gh pr list --author`
   - Documented both approaches with pros/cons
   - Clear recommendations

## Files Changed

- `.github/workflows/copilot-agent-analysis.md` - Updated documentation
- `.github/workflows/copilot-agent-analysis.lock.yml` - Recompiled
- `.github/workflows/test-copilot-pr-list.yml` - NEW test workflow
- `INVESTIGATION_REPORT.md` - NEW detailed analysis
- `pkg/cli/gh_pr_list_test.go` - NEW unit tests
- `SUMMARY.md` - THIS file

## Conclusion

The command **DOES work** for listing Copilot PRs, but the **current workflow approach is more robust** for production use.

**Investigation Status: ✅ COMPLETE**
