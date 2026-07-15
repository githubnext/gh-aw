---
mcp-scripts:
  list_workflows:
    description: "List GitHub Actions workflows with per_page pagination support. Returns total_count, per_page, page, and a workflows array. Defaults to per_page=10 to avoid large responses."
    inputs:
      owner:
        type: string
        description: "Repository owner (username or organization)"
        required: true
      repo:
        type: string
        description: "Repository name"
        required: true
      per_page:
        type: number
        description: "Results per page (1–100, default: 10)"
        required: false
      page:
        type: number
        description: "Page number (default: 1)"
        required: false
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e

      OWNER="${INPUT_OWNER:-}"
      REPO="${INPUT_REPO:-}"
      PER_PAGE="${INPUT_PER_PAGE:-10}"
      PAGE="${INPUT_PAGE:-1}"

      if [[ -z "$OWNER" ]]; then
        echo '{"error": "owner is required"}' >&2
        exit 1
      fi

      if [[ -z "$REPO" ]]; then
        echo '{"error": "repo is required"}' >&2
        exit 1
      fi

      if ! [[ "$PER_PAGE" =~ ^[0-9]+$ ]] || [[ "$PER_PAGE" -lt 1 ]] || [[ "$PER_PAGE" -gt 100 ]]; then
        echo '{"error": "per_page must be between 1 and 100"}' >&2
        exit 1
      fi

      if ! [[ "$PAGE" =~ ^[0-9]+$ ]] || [[ "$PAGE" -lt 1 ]]; then
        echo '{"error": "page must be a positive integer"}' >&2
        exit 1
      fi

      RESPONSE=$(gh api "repos/${OWNER}/${REPO}/actions/workflows?per_page=${PER_PAGE}&page=${PAGE}")

      echo "$RESPONSE" | jq \
        --argjson per_page "$PER_PAGE" \
        --argjson page "$PAGE" \
        '{
          total_count: .total_count,
          per_page: $per_page,
          page: $page,
          workflows: [.workflows[] | {id, node_id, name, path, state, created_at, updated_at, url, html_url, badge_url}]
        }'

  list_label:
    description: "List labels in a GitHub repository with per_page pagination support. Returns labels array, item_count, per_page, and page. Defaults to per_page=10 to avoid large responses."
    inputs:
      owner:
        type: string
        description: "Repository owner (username or organization)"
        required: true
      repo:
        type: string
        description: "Repository name"
        required: true
      per_page:
        type: number
        description: "Results per page (1–100, default: 10)"
        required: false
      page:
        type: number
        description: "Page number (default: 1)"
        required: false
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e

      OWNER="${INPUT_OWNER:-}"
      REPO="${INPUT_REPO:-}"
      PER_PAGE="${INPUT_PER_PAGE:-10}"
      PAGE="${INPUT_PAGE:-1}"

      if [[ -z "$OWNER" ]]; then
        echo '{"error": "owner is required"}' >&2
        exit 1
      fi

      if [[ -z "$REPO" ]]; then
        echo '{"error": "repo is required"}' >&2
        exit 1
      fi

      if ! [[ "$PER_PAGE" =~ ^[0-9]+$ ]] || [[ "$PER_PAGE" -lt 1 ]] || [[ "$PER_PAGE" -gt 100 ]]; then
        echo '{"error": "per_page must be between 1 and 100"}' >&2
        exit 1
      fi

      if ! [[ "$PAGE" =~ ^[0-9]+$ ]] || [[ "$PAGE" -lt 1 ]]; then
        echo '{"error": "page must be a positive integer"}' >&2
        exit 1
      fi

      RESPONSE=$(gh api "repos/${OWNER}/${REPO}/labels?per_page=${PER_PAGE}&page=${PAGE}")

      echo "$RESPONSE" | jq \
        --argjson per_page "$PER_PAGE" \
        --argjson page "$PAGE" \
        '{
          labels: [.[] | {id, node_id, url, name, color, default, description}],
          item_count: length,
          per_page: $per_page,
          page: $page
        }'

  list_issues:
    description: "List issues in a repository with per_page pagination, returning only triage metadata (number, title, state, labels, author, created_at, html_url) — body is intentionally omitted to reduce token cost. Defaults to per_page=10 and state=open."
    inputs:
      owner:
        type: string
        description: "Repository owner (username or organization)"
        required: true
      repo:
        type: string
        description: "Repository name"
        required: true
      state:
        type: string
        description: "Issue state: open, closed, or all (default: open)"
        required: false
      labels:
        type: string
        description: "Comma-separated list of label names to filter by (default: none)"
        required: false
      assignee:
        type: string
        description: "Filter by assignee login (default: none)"
        required: false
      per_page:
        type: number
        description: "Results per page (1–100, default: 10)"
        required: false
      page:
        type: number
        description: "Page number (default: 1)"
        required: false
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e

      OWNER="${INPUT_OWNER:-}"
      REPO="${INPUT_REPO:-}"
      STATE="${INPUT_STATE:-open}"
      LABELS="${INPUT_LABELS:-}"
      ASSIGNEE="${INPUT_ASSIGNEE:-}"
      PER_PAGE="${INPUT_PER_PAGE:-10}"
      PAGE="${INPUT_PAGE:-1}"

      if [[ -z "$OWNER" ]]; then
        echo '{"error": "owner is required"}' >&2
        exit 1
      fi

      if [[ -z "$REPO" ]]; then
        echo '{"error": "repo is required"}' >&2
        exit 1
      fi

      if ! [[ "$PER_PAGE" =~ ^[0-9]+$ ]] || [[ "$PER_PAGE" -lt 1 ]] || [[ "$PER_PAGE" -gt 100 ]]; then
        echo '{"error": "per_page must be between 1 and 100"}' >&2
        exit 1
      fi

      if ! [[ "$PAGE" =~ ^[0-9]+$ ]] || [[ "$PAGE" -lt 1 ]]; then
        echo '{"error": "page must be a positive integer"}' >&2
        exit 1
      fi

      QUERY="repos/${OWNER}/${REPO}/issues?state=${STATE}&per_page=${PER_PAGE}&page=${PAGE}"
      if [[ -n "$LABELS" ]]; then
        QUERY="${QUERY}&labels=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$LABELS")"
      fi
      if [[ -n "$ASSIGNEE" ]]; then
        QUERY="${QUERY}&assignee=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$ASSIGNEE")"
      fi

      RESPONSE=$(gh api "$QUERY")

      echo "$RESPONSE" | jq \
        --argjson per_page "$PER_PAGE" \
        --argjson page "$PAGE" \
        '{
          issues: [.[] | {
            number,
            title,
            state,
            html_url,
            created_at,
            updated_at,
            author: .user.login,
            labels: [.labels[].name],
            assignees: [.assignees[].login],
            milestone: (.milestone.title // null)
          }],
          item_count: length,
          per_page: $per_page,
          page: $page
        }'

  search_issues:
    description: "Search issues/PRs across GitHub with per_page pagination, returning only triage metadata (number, title, state, labels, author, created_at, html_url, repository) — body is intentionally omitted to reduce token cost. Defaults to per_page=10."
    inputs:
      query:
        type: string
        description: "GitHub search query (e.g. 'repo:owner/name is:issue is:open no:label')"
        required: true
      per_page:
        type: number
        description: "Results per page (1–100, default: 10)"
        required: false
      page:
        type: number
        description: "Page number (default: 1)"
        required: false
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e

      QUERY="${INPUT_QUERY:-}"
      PER_PAGE="${INPUT_PER_PAGE:-10}"
      PAGE="${INPUT_PAGE:-1}"

      if [[ -z "$QUERY" ]]; then
        echo '{"error": "query is required"}' >&2
        exit 1
      fi

      if ! [[ "$PER_PAGE" =~ ^[0-9]+$ ]] || [[ "$PER_PAGE" -lt 1 ]] || [[ "$PER_PAGE" -gt 100 ]]; then
        echo '{"error": "per_page must be between 1 and 100"}' >&2
        exit 1
      fi

      if ! [[ "$PAGE" =~ ^[0-9]+$ ]] || [[ "$PAGE" -lt 1 ]]; then
        echo '{"error": "page must be a positive integer"}' >&2
        exit 1
      fi

      ENCODED_QUERY=$(python3 -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))" "$QUERY")
      RESPONSE=$(gh api "search/issues?q=${ENCODED_QUERY}&per_page=${PER_PAGE}&page=${PAGE}")

      echo "$RESPONSE" | jq \
        --argjson per_page "$PER_PAGE" \
        --argjson page "$PAGE" \
        '{
          total_count: .total_count,
          incomplete_results: .incomplete_results,
          per_page: $per_page,
          page: $page,
          items: [.items[] | {
            number,
            title,
            state,
            html_url,
            created_at,
            updated_at,
            author: .user.login,
            labels: [.labels[].name],
            assignees: [.assignees[].login],
            milestone: (.milestone.title // null),
            repository: (.repository_url | split("/") | .[-2:] | join("/"))
          }]
        }'
---
<!--
## GitHub MCP Pagination Wrappers

This shared workflow provides mcp-script wrappers for `list_workflows`, `list_label`,
`list_issues`, and `search_issues` that add proper `per_page`/`page` pagination support
and strip large fields (e.g. issue bodies) to reduce token cost in triage agents.

The built-in `list_label` GitHub MCP tool returns up to 100 labels regardless of any
`per_page` argument (it uses a hardcoded GraphQL `labels(first: 100)` query). The
`list_workflows` deprecated alias for `actions_list` may not surface `per_page` in its
schema to callers. The `list_issues` and `search_issues` MCP tools return full issue
bodies even when only titles/numbers are needed for triage. These wrappers call the
GitHub REST API directly so `per_page` is respected and body fields are projected away.

### Available Tools

1. **list_workflows** — List Actions workflows, calls `GET /repos/{owner}/{repo}/actions/workflows`
2. **list_label** — List repository labels, calls `GET /repos/{owner}/{repo}/labels`
3. **list_issues** — List repository issues (triage mode, no body), calls `GET /repos/{owner}/{repo}/issues`
4. **search_issues** — Search issues/PRs (triage mode, no body), calls `GET /search/issues`

### Usage

Import this shared workflow to activate all wrappers:

```yaml
imports:
  - shared/github-mcp-pagination-wrappers.md
```

### Tool Parameters

#### list_workflows

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| owner | string | Yes | - | Repository owner |
| repo | string | Yes | - | Repository name |
| per_page | number | No | 10 | Results per page (1–100) |
| page | number | No | 1 | Page number |

#### list_label

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| owner | string | Yes | - | Repository owner |
| repo | string | Yes | - | Repository name |
| per_page | number | No | 10 | Results per page (1–100) |
| page | number | No | 1 | Page number |

#### list_issues

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| owner | string | Yes | - | Repository owner |
| repo | string | Yes | - | Repository name |
| state | string | No | open | Issue state: open, closed, or all |
| labels | string | No | - | Comma-separated label names to filter by |
| assignee | string | No | - | Filter by assignee login |
| per_page | number | No | 10 | Results per page (1–100) |
| page | number | No | 1 | Page number |

#### search_issues

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| query | string | Yes | - | GitHub search query (e.g. `repo:owner/name is:issue is:open no:label`) |
| per_page | number | No | 10 | Results per page (1–100) |
| page | number | No | 1 | Page number |

### list_workflows Response

```json
{
  "total_count": 42,
  "per_page": 10,
  "page": 1,
  "workflows": [
    {
      "id": 12345,
      "node_id": "W_...",
      "name": "CI",
      "path": ".github/workflows/ci.yml",
      "state": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "url": "https://api.github.com/repos/owner/repo/actions/workflows/12345",
      "html_url": "https://github.com/owner/repo/actions/workflows/ci.yml",
      "badge_url": "https://github.com/owner/repo/actions/workflows/ci.yml/badge.svg"
    }
  ]
}
```

### list_label Response

```json
{
  "labels": [
    {
      "id": 1,
      "node_id": "LA_...",
      "url": "https://api.github.com/repos/owner/repo/labels/bug",
      "name": "bug",
      "color": "d73a4a",
      "default": true,
      "description": "Something isn't working"
    }
  ],
  "item_count": 10,
  "per_page": 10,
  "page": 1
}
```

### list_issues Response

Body is omitted — use `issue_read` to fetch full details for a specific issue when needed.

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

### search_issues Response

Body is omitted — use `issue_read` to fetch full details for a specific issue when needed.

```json
{
  "total_count": 42,
  "incomplete_results": false,
  "per_page": 10,
  "page": 1,
  "items": [
    {
      "number": 123,
      "title": "Fix login bug",
      "state": "open",
      "html_url": "https://github.com/owner/repo/issues/123",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-02T00:00:00Z",
      "author": "octocat",
      "labels": ["bug"],
      "assignees": [],
      "milestone": null,
      "repository": "owner/repo"
    }
  ]
}
```

### Source scripts

- `.github/skills/github-workflows-query/query-workflows.sh`
- `.github/skills/github-labels-query/query-labels.sh`
- `.github/skills/github-issue-query/query-issues-triage.sh`
-->
