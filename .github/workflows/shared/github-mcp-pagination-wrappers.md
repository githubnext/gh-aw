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
---
<!--
## GitHub MCP Pagination Wrappers

This shared workflow provides mcp-script wrappers for `list_workflows` and `list_label`
that add proper `per_page` and `page` pagination support.

The built-in `list_label` GitHub MCP tool returns up to 100 labels regardless of any
`per_page` argument (it uses a hardcoded GraphQL `labels(first: 100)` query). The
`list_workflows` deprecated alias for `actions_list` may not surface `per_page` in its
schema to callers. These wrappers call the GitHub REST API directly so `per_page` is
respected on every call.

### Available Tools

1. **list_workflows** — List Actions workflows, calls `GET /repos/{owner}/{repo}/actions/workflows`
2. **list_label** — List repository labels, calls `GET /repos/{owner}/{repo}/labels`

### Usage

Import this shared workflow to activate both wrappers:

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

### Source scripts

- `.github/skills/github-workflows-query/query-workflows.sh`
- `.github/skills/github-labels-query/query-labels.sh`
-->
