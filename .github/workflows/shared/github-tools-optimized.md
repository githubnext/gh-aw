---
# Optimized GitHub tool configuration
# Import this in your workflow with: imports: [shared/github-tools-optimized.md]

# This configuration avoids tools that cause token overflow
# and prioritizes efficient server-side filtering

# AVOID: list_pull_requests - Can return 150k+ tokens (exceeds 25k MCP limit)
# USE: search_pull_requests - Efficient server-side filtering, respects limits
---

# GitHub Tools Optimization Guide

This shared configuration provides optimized GitHub tool usage based on live analysis of token overflow issues.

## The Problem: Token Overflow

The MCP protocol has a **25,000 token limit** per tool response. Some GitHub API tools can exceed this:

- **list_pull_requests**: Can return 150k+ tokens when listing many PRs
- **list_issues**: Similar token overflow risk with large repositories
- **list_commits**: Can overflow on active repositories

### Real-world Example

From workflow run analysis:
```
MCP tool "list_pull_requests" response (150414 tokens) exceeds maximum allowed tokens (25000)
```

This caused workflows to fail and waste tokens on unusable responses.

## Recommended Tools

### ✅ Use These (Efficient)

- **search_pull_requests** - Server-side filtering, respects token limits
- **search_issues** - Server-side filtering with query syntax
- **pull_request_read** - Get specific PR details (controlled token usage)
- **issue_read** - Get specific issue details (controlled token usage)
- **get_file_contents** - Specific file retrieval (predictable size)
- **list_commits** - Use with pagination and limits
- **get_commit** - Get specific commit details

### ⚠️ Avoid These (Token Overflow Risk)

- **list_pull_requests** - Use `search_pull_requests` instead
- **list_issues** - Use `search_issues` instead
- Avoid listing operations without filters

## Usage Examples

### Before (Causes Token Overflow)
```yaml
tools:
  github:
    allowed:
      - list_pull_requests  # ❌ Can return 150k+ tokens
```

### After (Optimized)
```yaml
tools:
  github:
    allowed:
      - search_pull_requests  # ✅ Efficient server-side filtering
      - pull_request_read     # ✅ Controlled token usage
```

### Using search_pull_requests

In your workflow prompt, use search syntax:
```
Search for merged pull requests created in the last week:
- Use search_pull_requests with query: "is:pr is:merged created:>=2024-01-01"
```

### Using gh CLI (Alternative)

For pre-fetching data in workflow steps:
```bash
# Server-side filtering with gh CLI
gh search prs --repo ${{ github.repository }} \
  --author "@copilot" \
  --created ">=2024-01-01" \
  --limit 100 \
  --json number,title,state
```

## Import This Configuration

Add to your workflow frontmatter:
```yaml
imports:
  - shared/github-tools-optimized.md
```

Then configure GitHub tools explicitly:
```yaml
tools:
  github:
    allowed:
      - search_pull_requests
      - pull_request_read
      - search_issues
      - issue_read
```

## References

- MCP token limit: 25,000 tokens per response
- GitHub search syntax: https://docs.github.com/en/search-github/searching-on-github
- Analyzed workflow runs:
  - Run 18821735224 (Copilot Agent PR Analysis): 150k token overflow
  - Run 18821713918 (Smoke Claude): 150k and 39k token overflows
