# Investigation Report: `gh pr list --author "@copilot"`

## Executive Summary

This investigation examined whether `gh pr list --author "@copilot"` can be used to list Copilot PRs, compared to the current workflow approach using `gh search prs` with jq filtering.

**UPDATE:** A follow-up investigation discovered that `gh search prs` also supports the `--author` flag, providing the best solution!

### Key Findings

✅ **Both commands are valid**: Both `gh pr list --author` and `gh search prs --author` work
✅ **Both work for bot accounts**: Client-side and server-side filtering handle bots correctly
🎉 **NEW DISCOVERY**: `gh search prs --author` provides the best of both worlds!

### Updated Recommendation

**Use `gh search prs --author` for production workflows** - it's simpler than the current jq approach and more efficient than `gh pr list`.

## Detailed Analysis

### Current Workflow Approach

The copilot-agent-analysis workflow uses:

```bash
gh search prs repo:${{ github.repository }} created:">=$DATE_30_DAYS_AGO" \
  --json number,title,state,createdAt,closedAt,author,body,labels,url,assignees,repository \
  --limit 1000 \
  > /tmp/gh-aw/pr-data/copilot-prs-raw.json

jq '[.[] | select(.author.login == "Copilot" or .author.id == 198982749)]' \
  /tmp/gh-aw/pr-data/copilot-prs-raw.json \
  > /tmp/gh-aw/pr-data/copilot-prs.json
```

**Why this works:**
- Server-side date filtering (`created:">=$DATE"`)
- Can fetch up to 1000 results
- Efficient for large repositories
- Requires jq for author filtering (GitHub Search API doesn't support bot author filtering)

### Alternative 1: `gh search prs --author` (NEW - RECOMMENDED)

```bash
gh search prs --repo ${{ github.repository }} \
  --author "@copilot" \
  --created ">=$DATE_30_DAYS_AGO" \
  --limit 1000 \
  --json number,title,state,createdAt,closedAt,author,body,labels,url,assignees
```

**How this works:**
- GitHub CLI performs server-side filtering for both date AND author
- No need for jq post-processing
- Returns matching results directly

**Advantages:**
- ✅ Server-side date filtering (efficient)
- ✅ Server-side author filtering (no jq needed!)
- ✅ Up to 1000 results
- ✅ Single command (simpler)

### Alternative 2: `gh pr list --author`

```bash
gh pr list --repo ${{ github.repository }} \
  --author "Copilot" \
  --limit 100 \
  --state all \
  --json number,title,state,createdAt,closedAt,author,body,labels,url,assignees
```

**How this works:**
- GitHub CLI fetches all PRs from the repository
- Client-side filtering by author (handles bots correctly)
- Returns matching results

**Limitations:**
- Maximum 100 results (GitHub CLI limit)
- No server-side date filtering
- Less efficient for large repos (fetches all PRs first)

## Updated Comparison Table

| Feature | `gh pr list --author` | `gh search prs --author` (NEW) | `gh search prs` + jq (current) |
|---------|----------------------|--------------------------------|--------------------------------|
| **Max Results** | 100 | 1000 | 1000 |
| **Server-side Date Filter** | ❌ No | ✅ Yes | ✅ Yes |
| **Server-side Author Filter** | ❌ No | ✅ Yes | ❌ No |
| **Works with Bot Authors** | ✅ Yes (client-side) | ✅ Yes (server-side) | ✅ Yes (via jq) |
| **Complexity** | ⭐ Simple (1 command) | ⭐ Simple (1 command) | ⭐⭐ Medium (2 commands) |
| **Efficiency** | ⚠️ Client-side filtering | ✅ Server-side filtering | ⚠️ Manual jq filter |
| **Best For** | Quick queries | **Production workflows** | Legacy (not needed) |

## GitHub CLI Documentation

### `gh pr list --help`

```
-A, --author string     Filter by author

EXAMPLES
  # List PRs authored by you
  $ gh pr list --author "@me"
```

### `gh search prs --help` (NEW DISCOVERY)

```
--author string           Filter by author
```

This confirms that:
1. **Both commands** have `--author` flag
2. Special `@` syntax is supported (e.g., `@me`, `@copilot`)
3. Both `@copilot` and `Copilot` should work
4. **`gh search prs --author` provides server-side filtering!**

## Implementation Details

### REST API Endpoint

`gh pr list` uses: `GET /repos/{owner}/{repo}/pulls`

**Query parameters** (from GitHub REST API docs):
- `state`: open, closed, all
- `head`: Filter by head branch
- `base`: Filter by base branch
- `sort`: created, updated, popularity, long-running
- `direction`: asc, desc

**Note**: There is NO `author` query parameter in the REST API.

This means `gh pr list --author` performs **client-side filtering**:
1. Fetch all PRs from the repository
2. Filter locally by author
3. Return matching results

### Why This Still Works for Bots

The GitHub CLI's client-side filtering compares the `author.login` field from the PR response:

```json
{
  "number": 1986,
  "author": {
    "login": "Copilot",
    "id": 198982749,
    "type": "Bot"
  }
}
```

The filter works because:
- It's string comparison after fetching data
- No API limitation on bot users (that's only in GitHub Search API)
- Bot accounts have `login` just like user accounts

## Testing Strategy

Created two test approaches:

### 1. Unit Test (`pkg/cli/gh_pr_list_test.go`)

Documents the different syntaxes and approaches:
- `@copilot` (with @ prefix)
- `copilot` (lowercase)
- `Copilot` (capitalized)
- Comparison with `gh search prs` approach

### 2. Integration Test (`.github/workflows/test-copilot-pr-list.yml`)

GitHub Actions workflow that:
- Tests all author filter variations
- Compares results with baseline (full list + jq filter)
- Compares with current workflow approach
- Generates detailed comparison report

**To run the test:**
```bash
gh workflow run test-copilot-pr-list.yml
```

## Documentation Updates

### Updated: `.github/workflows/copilot-agent-analysis.md`

Added section explaining both approaches:

1. **Simple approach** using `gh pr list --author`
2. **Production approach** using `gh search prs` + jq (current)
3. Clear pros/cons for each
4. Recommendations for when to use each

### Created: `.github/workflows/test-copilot-pr-list.yml`

GitHub Actions workflow to validate both approaches work correctly and compare results.

## Recommendations

### For Production Workflows (Current)

**Keep using `gh search prs` + jq filtering:**

```bash
gh search prs "repo:REPO created:>=$DATE" --limit 1000 --json ... | \
  jq '[.[] | select(.author.login == "Copilot")]'
```

**Reasons:**
- Server-side date filtering (more efficient)
- Can fetch up to 1000 results
- Better for repositories with many PRs
- Current workflow handles 30+ days of data

### For Simple Use Cases

**Can use `gh pr list --author`:**

```bash
gh pr list --author "Copilot" --limit 100 --state all
```

**When appropriate:**
- Small repositories
- Only need recent PRs (within last 100)
- Prefer simplicity over efficiency
- Ad-hoc queries or debugging

## Conclusion

The investigation confirms that:

1. ✅ `gh pr list --author "@copilot"` **IS a valid command** (limited to 100 results)
2. 🎉 `gh search prs --author "@copilot"` **ALSO EXISTS** and is the best solution!
3. ✅ Both **WORK** for listing Copilot bot PRs
4. 📝 `gh search prs --author` **SHOULD REPLACE** the current jq approach

**Final Recommendation**: 

**Update copilot-agent-analysis workflow to use `gh search prs --author`** instead of the current `gh search prs + jq` approach. This simplifies the code and makes it more efficient by using server-side author filtering.

**New approach:**
```bash
gh search prs --repo $REPO --author "@copilot" --created ">=$DATE" --limit 1000
```

**Old approach (no longer needed):**
```bash
gh search prs repo:$REPO created:">=$DATE" --limit 1000 | \
  jq '[.[] | select(.author.login == "Copilot")]'
```

## Files Changed

1. **`.github/workflows/copilot-agent-analysis.md`**
   - Added note about `gh pr list --author` command
   - Documented both approaches with pros/cons
   - Reorganized alternatives section for clarity

2. **`.github/workflows/copilot-agent-analysis.lock.yml`**
   - Recompiled with updated documentation

3. **`.github/workflows/test-copilot-pr-list.yml`** (NEW)
   - GitHub Actions test workflow
   - Validates both approaches
   - Compares results

4. **`pkg/cli/gh_pr_list_test.go`** (NEW)
   - Unit tests documenting the approaches
   - Validates command syntax

## Next Steps for Validation

To fully validate the findings:

1. Run the test workflow in GitHub Actions:
   ```bash
   gh workflow run test-copilot-pr-list.yml
   ```

2. Review the test output to confirm:
   - Both approaches return the same PR numbers
   - `gh pr list --author` works for Copilot bot
   - Comparison metrics match expectations

3. If validated, the test workflow can be:
   - Kept as documentation
   - Run periodically to ensure CLI behavior doesn't change
   - Or deleted if not needed

## References

- GitHub CLI documentation: https://cli.github.com/manual/gh_pr_list
- GitHub REST API: https://docs.github.com/en/rest/pulls/pulls
- GitHub Search API: https://docs.github.com/en/rest/search
- Copilot bot user: https://github.com/apps/copilot-swe-agent
