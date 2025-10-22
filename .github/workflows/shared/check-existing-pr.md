---
tools:
  github:
    allowed:
      - search_pull_requests
      - list_pull_requests
---

## Check for Existing Pull Request

Before creating a new pull request, verify if there's already an open PR from this workflow to avoid duplicates.

### Search Strategy

Use both title prefix and labels for highly effective PR matching:

1. **Search by title prefix and labels**: Use `search_pull_requests` with a query combining:
   - Repository scope: `repo:${{ github.repository }}`
   - PR filter: `is:pr`
   - State filter: `is:open`
   - Title pattern matching the workflow's title prefix
   - Label filter matching the workflow's automation labels

2. **Fallback search**: If needed, use `list_pull_requests` with `state: open` and manually filter by:
   - Title prefix match
   - Label match (both "automation" and workflow-specific labels)

### Example Search Pattern

For a workflow that creates PRs with:
- Title prefix: `[ca]` (CLI automation)
- Labels: `automation`, `dependencies`

Use this search query:
```
repo:${{ github.repository }} is:pr is:open "[ca]" label:automation label:dependencies
```

This pattern is highly effective because it combines:
- **Exact repository scoping**: Limits search to current repository
- **PR type filtering**: Only pull requests, not issues
- **State filtering**: Only open PRs
- **Title prefix matching**: Matches the unique workflow prefix in quotes for exact matching
- **Label combination**: Requires BOTH automation and workflow-specific labels

### Implementation

When checking for existing PRs:

1. **First attempt** - Use `search_pull_requests`:
   ```
   query: 'repo:${{ github.repository }} is:pr is:open "[your-prefix]" label:automation label:workflow-label'
   ```

2. **Verify results**: Check if any PRs are returned
   - If found: Extract PR number, URL, and branch name
   - If none: Proceed with creating a new PR

3. **Handle edge cases**:
   - Multiple matching PRs: Use the most recently created/updated
   - PR closed recently: Confirm it's truly open before reporting

### Stop Condition

If an existing open PR is found:
- Print a clear message: "An open pull request already exists for this workflow"
- Include PR details: number, URL, title
- Stop execution immediately without creating a new PR
- Exit successfully (not as an error)

### Integration with Workflows

Import this shared workflow in your agentic workflow frontmatter:
```yaml
imports:
  - shared/check-existing-pr.md
```

Then, in your workflow instructions, add as the first step:
```
### 0. Check for Existing Pull Request
Before starting any work, check if there's already an open pull request from this workflow:
- Use the search pattern with title prefix "[your-prefix]" and labels
- If found, print message and stop execution
- If not found, proceed with workflow tasks
```

### Search Query Examples

**For CLI version checker** (`[ca]` prefix, `automation` + `dependencies` labels):
```
repo:${{ github.repository }} is:pr is:open "[ca]" label:automation label:dependencies
```

**For tidy workflow** (`[tidy]` prefix, `automation` + `maintenance` labels):
```
repo:${{ github.repository }} is:pr is:open "[tidy]" label:automation label:maintenance
```

**For security fixes** (`[security-fix]` prefix, `security` + `automated-fix` labels):
```
repo:${{ github.repository }} is:pr is:open "[security-fix]" label:security label:automated-fix
```

This approach ensures extremely effective PR detection by requiring all criteria to match, making false positives virtually impossible.
