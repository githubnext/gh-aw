---
description: Pagination guidance for GitHub MCP tools to stay within token limits while retrieving complete result sets.
---

# GitHub MCP Server — Pagination

See [github-mcp-server.md](github-mcp-server.md) for toolset and tool reference.

MCP tool responses have a **25,000 token limit**. Fetching large result sets without pagination causes the response to be truncated or rejected, forcing costly retry turns.

## `perPage` Defaults by Item Type

| Item type | Recommended `perPage` |
|-----------|----------------------|
| PRs with diffs / issues with comments (detailed) | 10–20 |
| Simple list operations (commits, branches, labels) | 50–100 |
| Exploratory / schema-discovery queries | 1–5 |

Always pass an explicit `perPage` value. Do **not** rely on server defaults.

## Tool-Specific Guidance

**Pull Requests**
- `list_pull_requests` — use `perPage: 10`, `sort: updated`, `direction: desc`
- `pull_request_read` with `method: get_files` — use `perPage: 30`
- Fetch diff and comments **separately** when full detail is needed

**Issues**
- `list_issues` — `perPage: 20`
- `issue_read` with `method: get_comments` — `perPage: 20`

**Search**
- `search_issues`, `search_pull_requests`, `search_code` — `perPage: 10`
- `search_repositories` exploratory calls — `perPage: 3–5`; increase only after narrowing the query

## Pagination Loop (when all pages are needed)

```
page 1 → check total_count or has_next_page → fetch page 2, 3, … until done
```

Process results incrementally rather than accumulating all pages in memory.

## Known Tool Quirks

Two built-in GitHub MCP tools ignore standard pagination parameters:

- **`list_label`** — uses a hardcoded GraphQL `labels(first: 100)` query; `perPage` is silently ignored. Use the `shared/github-mcp-pagination-wrappers.md` wrapper instead.
- **`list_workflows`** — uses snake_case `per_page` (inconsistent with every other list tool). Use the `shared/github-mcp-pagination-wrappers.md` wrapper for consistent camelCase `perPage` support.

One built-in GitHub MCP tool ignores a search qualifier:

- **`search_repositories` with `repo:`** — the `repo:owner/name` qualifier is silently ignored; results are ranked by star count and will return unrelated high-star repositories. Use `org:`, `user:`, `topic:`, or `stars:` to scope repository searches. To resolve a specific repository, use a `repos`-toolset call with explicit `owner` and `repo` parameters (e.g. `get_file_contents`) instead.

## Oversized-Response Errors

If you encounter errors like:

- `MCP tool "list_pull_requests" response (75897 tokens) exceeds maximum allowed tokens (25000)`
- `Response too large for tool [tool_name]`

add `perPage: 10` (or smaller) and retry.
