## MCP Response Size Limits

MCP tool responses have a **25,000 token limit**. When GitHub API responses exceed this limit, workflows must retry with pagination parameters, wasting turns and tokens.

### Common Scenarios

**Problem**: Fetching large result sets without pagination
- `list_pull_requests` with many PRs (75,897 tokens in one case)
- `pull_request_read` with large diff/comments (31,675 tokens observed)
- `search_issues`, `search_code` with many results

**Solution**: Use proactive pagination to stay under token limits

### Pagination Best Practices

#### 1. Use `perPage` Parameter

Limit results per request to prevent oversized responses:

```bash
# Good: Fetch PRs in small batches
list_pull_requests --perPage 10

# Good: Get issue with limited comments
issue_read --method get_comments --perPage 20

# Bad: Default pagination may return too much data
list_pull_requests  # May exceed 25k tokens
```

#### 2. Common `perPage` Values

- **10-20**: For detailed items (PRs with diffs, issues with comments)
- **50-100**: For simpler list operations (commits, branches, labels)
- **1-5**: For exploratory queries or schema discovery

#### 3. Handle Pagination Loops

When you need all results:

```bash
# Step 1: Fetch first page
result=$(list_pull_requests --perPage 20 --page 1)

# Step 2: Check if more pages exist
# Most list operations return metadata about total count or next page

# Step 3: Fetch subsequent pages if needed
result=$(list_pull_requests --perPage 20 --page 2)
```

### Tool-Specific Guidance

#### Pull Requests

```bash
# Fetch recent PRs in small batches
list_pull_requests --state all --perPage 10 --sort updated --direction desc

# Get PR details without full diff/comments
pull_request_read --method get --pullNumber 123

# Get PR files separately if needed
pull_request_read --method get_files --pullNumber 123 --perPage 30
```

#### Issues

```bash
# List issues with pagination
list_issues --perPage 20 --page 1

# Get issue comments in batches
issue_read --method get_comments --issue_number 123 --perPage 20
```

#### Code Search

```bash
# Search with limited results
search_code --query "function language:go" --perPage 10
```

### Error Messages to Watch For

If you see these errors, add pagination:

- `MCP tool "list_pull_requests" response (75897 tokens) exceeds maximum allowed tokens (25000)`
- `MCP tool "pull_request_read" response (31675 tokens) exceeds maximum allowed tokens (25000)`
- `Response too large for tool [tool_name]`

### Performance Tips

1. **Start small**: Use `perPage: 10` initially, increase if needed
2. **Fetch incrementally**: Get overview first, then details for specific items
3. **Avoid wildcards**: Don't fetch all data when you need specific items
4. **Use filters**: Combine `perPage` with state/label/date filters to reduce results

### Example Workflow Pattern

```markdown
# Analyze Recent Pull Requests

1. Fetch 10 most recent PRs (stay under token limit)
2. For each PR, get summary without full diff
3. If detailed analysis needed, fetch files for specific PR separately
4. Process results incrementally rather than loading everything at once
```

This proactive approach eliminates retry loops and reduces token consumption.
