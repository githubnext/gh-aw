---
name: github-issue-query
description: Query GitHub issues with jq filtering and reusable selectors.
---

# GitHub Issue Query Skill

Query GitHub issues efficiently with built-in jq filtering.

## Triage Mode (Title-Only, No Body)

For triage-style tasks that only need issue metadata, use `query-issues-triage.sh`
instead of `query-issues.sh`. It omits the issue body to dramatically reduce token cost.

```bash
./query-issues-triage.sh --owner github --repo gh-aw
# Returns 10 open issues (default per_page=10), no body field
```

### Triage Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `--owner` | Yes | - | Repository owner (username or organization) |
| `--repo` | Yes | - | Repository name |
| `--state` | No | open | Issue state: open, closed, or all |
| `--labels` | No | - | Comma-separated label names to filter by |
| `--assignee` | No | - | Filter by assignee login |
| `--per-page` | No | 10 | Results per page (1–100) |
| `--page` | No | 1 | Page number |

### Triage Output

```json
{
  "issues": [
    {
      "number": 123,
      "title": "Fix login bug",
      "state": "open",
      "html_url": "https://github.com/owner/repo/issues/123",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-02T00:00:00Z",
      "author": "octocat",
      "labels": ["bug", "priority:high"],
      "assignees": ["monalisa"],
      "milestone": null
    }
  ],
  "item_count": 10,
  "per_page": 10,
  "page": 1
}
```

Use `issue_read` to fetch the full body for a specific issue when needed.

## Full-Data Mode (jq Filtering)

The `--jq` parameter is **optional**. Without `--jq`, this skill returns **schema and data size information** instead of full data.
Use this to avoid oversized responses and inspect structure before targeted queries.

Use `--jq '.'` to get all data, or use a more specific filter for targeted results.

## Usage

Use this skill to query issues from the current repository or any specified repository.

### Basic Query (Returns Schema Only)

To list issues from the current repository:

```bash
./query-issues.sh
# Returns schema and data size, not full data
```

### Get All Data

To get all issue data:

```bash
./query-issues.sh --jq '.'
```

### With Repository

To query a specific repository:

```bash
./query-issues.sh --repo owner/repo
```

### With jq Filtering

Use the `--jq` argument to filter and transform the output:

```bash
# Get only open issues
./query-issues.sh --jq '.[] | select(.state == "OPEN")'

# Get issue numbers and titles
./query-issues.sh --jq '.[] | {number, title}'

# Get issues by a specific author
./query-issues.sh --jq '.[] | select(.author.login == "username")'

# Get issues with specific label
./query-issues.sh --jq '.[] | select(.labels | map(.name) | index("bug"))'

# Count issues by state
./query-issues.sh --jq 'group_by(.state) | map({state: .[0].state, count: length})'
```

### Common Options

- `--state`: Filter by state (open, closed, all). Default: open
- `--limit`: Maximum number of issues to fetch. Default: 30
- `--repo`: Repository in owner/repo format. Default: current repo
- `--jq`: (Optional) jq expression for filtering/transforming output. If omitted, returns schema info

### Example Queries

**Find issues with many comments:**
```bash
./query-issues.sh --jq '.[] | select(.comments.totalCount > 5) | {number, title, comments: .comments.totalCount}'
```

**Get issues assigned to someone:**
```bash
./query-issues.sh --jq '.[] | select(.assignees | length > 0) | {number, title, assignees: [.assignees[].login]}'
```

**List issues with their labels:**
```bash
./query-issues.sh --jq '.[] | {number, title, labels: [.labels[].name]}'
```

**Get project board assignments:**
```bash
./query-issues.sh --jq '.[] | {number, title, projects: [.projectItems.nodes[]? | .project?.url]}'
```

**Find old issues (created over 30 days ago):**
```bash
./query-issues.sh --jq '.[] | select(.createdAt < (now - 2592000 | strftime("%Y-%m-%dT%H:%M:%SZ")))'
```

## Output Format

The script outputs JSON by default, making it easy to pipe through jq for additional processing.

## Requirements

- GitHub CLI (`gh`) authenticated
- `jq` for filtering (installed by default on most systems)
