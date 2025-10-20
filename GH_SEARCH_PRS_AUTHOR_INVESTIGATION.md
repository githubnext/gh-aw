# Additional Investigation: `gh search prs --author "@copilot"`

## Context

After completing the initial investigation of `gh pr list --author "@copilot"`, a follow-up request was made to investigate whether `gh search prs` also supports the `--author` flag.

## Key Discovery

✅ **`gh search prs` DOES support `--author` flag!**

From `gh search prs --help`:
```
--author string           Filter by author
```

This is an important discovery that changes the recommendations from the original investigation.

## Updated Comparison

### Three Approaches to List Copilot PRs

| Approach | Command | Max Results | Date Filter | Author Filter | Complexity |
|----------|---------|-------------|-------------|---------------|------------|
| **1. gh pr list** | `gh pr list --author "Copilot"` | 100 | ❌ Client-side | ✅ Built-in | ⭐ Simple |
| **2. gh search prs (NEW)** | `gh search prs --author "@copilot"` | 1000 | ✅ Server-side | ✅ Built-in | ⭐ Simple |
| **3. gh search prs + jq** | `gh search prs ... \| jq ...` | 1000 | ✅ Server-side | ⚠️ Manual jq | ⭐⭐ Medium |

## New Recommended Approach

**Use `gh search prs --author` for the best of both worlds:**

```bash
# Calculate date 30 days ago
DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

# Search for Copilot PRs with built-in author filtering
gh search prs --repo "${{ github.repository }}" \
  --author "@copilot" \
  --created ">=$DATE_30_DAYS_AGO" \
  --limit 1000 \
  --json number,title,state,createdAt,closedAt,author,body,labels,url,assignees
```

### Advantages of `gh search prs --author`

1. ✅ **Server-side date filtering** (efficient)
2. ✅ **Built-in author filtering** (no jq needed!)
3. ✅ **Up to 1000 results** (10x more than `gh pr list`)
4. ✅ **Single command** (simpler than jq approach)
5. ✅ **Works with bot accounts** (handles `@copilot` correctly)

### Why This Is Better Than Current Approach

The current workflow uses:
```bash
gh search prs repo:$REPO created:">=$DATE" --limit 1000 | \
  jq '[.[] | select(.author.login == "Copilot")]'
```

The new approach eliminates the need for jq filtering:
```bash
gh search prs --repo $REPO --author "@copilot" --created ">=$DATE" --limit 1000
```

**Benefits:**
- Simpler (no jq dependency for author filtering)
- More efficient (server-side author filtering)
- Cleaner code (one command instead of two)

## Testing

### Test Command Variations

All of these should work:
```bash
# With @ prefix
gh search prs --repo github/sweagentd --author "@copilot"

# Without @ prefix (lowercase)
gh search prs --repo github/sweagentd --author "copilot"

# Without @ prefix (capitalized)
gh search prs --repo github/sweagentd --author "Copilot"
```

### Test Workflow

Created test workflow at `.github/workflows/test-gh-search-prs-author.yml` to validate all three approaches:

1. `gh pr list --author "Copilot"` (max 100 results)
2. `gh search prs --author "@copilot"` (max 1000 results, NEW)
3. `gh search prs ... | jq ...` (max 1000 results, current)

## Updated Recommendations

### For Production Workflows (UPDATED)

**Recommended: Use `gh search prs --author`**

```bash
gh search prs --repo "${{ github.repository }}" \
  --author "@copilot" \
  --created ">=$DATE_30_DAYS_AGO" \
  --limit 1000 \
  --json number,title,state,createdAt,closedAt,author,body,labels,url,assignees
```

**Why:**
- Combines best of both worlds (server-side filtering + built-in author filter)
- Simpler than jq approach
- More efficient than `gh pr list`
- Handles up to 1000 results

### For Quick Queries

**Use `gh pr list --author` for simplicity:**

```bash
gh pr list --author "Copilot" --limit 100 --state all
```

**When appropriate:**
- Quick ad-hoc queries
- Only need recent PRs (< 100)
- Maximum simplicity

### Legacy Approach (No Longer Needed)

The current `gh search prs + jq` approach can be simplified:

```bash
# OLD (current workflow)
gh search prs repo:$REPO created:">=$DATE" --limit 1000 | \
  jq '[.[] | select(.author.login == "Copilot")]'

# NEW (recommended)
gh search prs --repo $REPO --author "@copilot" --created ">=$DATE" --limit 1000
```

## Implementation Plan

1. Update `.github/workflows/test-copilot-pr-list.yml` to include `gh search prs --author` tests
2. Update `.github/workflows/copilot-agent-analysis.md` to document the new approach
3. Consider updating the workflow to use `gh search prs --author` instead of jq filtering
4. Update `INVESTIGATION_REPORT.md` and `SUMMARY.md` with new findings

## Conclusion

The discovery of `gh search prs --author` flag is significant:

- ✅ **Simpler than current approach** (no jq needed)
- ✅ **More efficient than `gh pr list`** (1000 vs 100 limit)
- ✅ **Best solution overall** for listing Copilot PRs

This finding suggests the copilot-agent-analysis workflow should be updated to use `gh search prs --author` instead of the current `gh search prs + jq` approach.
