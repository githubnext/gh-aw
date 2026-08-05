---
mcp-scripts:
  list_workflows:
    description: "List GitHub Actions workflows with perPage pagination support. Returns total_count, per_page, page, and a workflows array. Defaults to perPage=10 to avoid large responses."
    inputs:
      owner:
        type: string
        description: "Repository owner (username or organization)"
        required: true
      repo:
        type: string
        description: "Repository name"
        required: true
      perPage:
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
      PER_PAGE="${INPUT_PERPAGE:-10}"
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
        echo '{"error": "perPage must be between 1 and 100"}' >&2
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
    description: "List labels in a GitHub repository with perPage pagination support. Returns labels array, item_count, per_page, and page. Defaults to perPage=10 to avoid large responses."
    inputs:
      owner:
        type: string
        description: "Repository owner (username or organization)"
        required: true
      repo:
        type: string
        description: "Repository name"
        required: true
      perPage:
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
      PER_PAGE="${INPUT_PERPAGE:-10}"
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
        echo '{"error": "perPage must be between 1 and 100"}' >&2
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

  get_file_contents_excerpt:
    description: "Read a bounded excerpt from a repository file without returning the whole file. Supports byteOffset/maxBytes and optional startLine/endLine filtering within the fetched byte window."
    inputs:
      owner:
        type: string
        description: "Repository owner (username or organization)"
        required: true
      repo:
        type: string
        description: "Repository name"
        required: true
      path:
        type: string
        description: "Path to the file in the repository"
        required: true
      ref:
        type: string
        description: "Git ref to read from (defaults to GITHUB_SHA, or the repository default branch when unavailable)"
        required: false
      byteOffset:
        type: number
        description: "Zero-based byte offset to start reading from (default: 0)"
        required: false
      maxBytes:
        type: number
        description: "Maximum bytes to fetch before line filtering (1-200000, default: 20000)"
        required: false
      startLine:
        type: number
        description: "Optional one-based line number to start returning within the fetched byte window"
        required: false
      endLine:
        type: number
        description: "Optional one-based line number to stop returning within the fetched byte window"
        required: false
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail

      OWNER="${INPUT_OWNER:-}"
      REPO="${INPUT_REPO:-}"
      PATH_IN_REPO="${INPUT_PATH:-}"
      REF="${INPUT_REF:-${GITHUB_SHA:-}}"
      BYTE_OFFSET="${INPUT_BYTEOFFSET:-0}"
      MAX_BYTES="${INPUT_MAXBYTES:-20000}"
      START_LINE="${INPUT_STARTLINE:-}"
      END_LINE="${INPUT_ENDLINE:-}"

      if [[ -z "$OWNER" ]]; then
        echo '{"error": "owner is required"}' >&2
        exit 1
      fi

      if [[ -z "$REPO" ]]; then
        echo '{"error": "repo is required"}' >&2
        exit 1
      fi

      if [[ -z "$PATH_IN_REPO" ]]; then
        echo '{"error": "path is required"}' >&2
        exit 1
      fi

      if ! [[ "$BYTE_OFFSET" =~ ^[0-9]+$ ]]; then
        echo '{"error": "byteOffset must be a non-negative integer"}' >&2
        exit 1
      fi

      if ! [[ "$MAX_BYTES" =~ ^[0-9]+$ ]] || [[ "$MAX_BYTES" -lt 1 ]] || [[ "$MAX_BYTES" -gt 200000 ]]; then
        echo '{"error": "maxBytes must be between 1 and 200000"}' >&2
        exit 1
      fi

      if [[ -n "$START_LINE" ]] && { ! [[ "$START_LINE" =~ ^[0-9]+$ ]] || [[ "$START_LINE" -lt 1 ]]; }; then
        echo '{"error": "startLine must be a positive integer"}' >&2
        exit 1
      fi

      if [[ -n "$END_LINE" ]] && { ! [[ "$END_LINE" =~ ^[0-9]+$ ]] || [[ "$END_LINE" -lt 1 ]]; }; then
        echo '{"error": "endLine must be a positive integer"}' >&2
        exit 1
      fi

      if [[ -n "$START_LINE" && -n "$END_LINE" && "$END_LINE" -lt "$START_LINE" ]]; then
        echo '{"error": "endLine must be greater than or equal to startLine"}' >&2
        exit 1
      fi

      if [[ -z "$REF" ]]; then
        REF=$(gh repo view "${OWNER}/${REPO}" --json defaultBranchRef --jq '.defaultBranchRef.name')
      fi

      RAW_FILE=$(mktemp)
      trap 'rm -f "$RAW_FILE"' EXIT

      BYTE_END=$((BYTE_OFFSET + MAX_BYTES))
      export OWNER REPO PATH_IN_REPO REF BYTE_OFFSET MAX_BYTES START_LINE END_LINE
      gh api \
        --method GET \
        -H "Accept: application/vnd.github.raw" \
        -H "Range: bytes=${BYTE_OFFSET}-${BYTE_END}" \
        "repos/${OWNER}/${REPO}/contents/${PATH_IN_REPO}" \
        -f "ref=${REF}" > "$RAW_FILE"

      python3 - "$RAW_FILE" <<'PY'
      import json
      import os
      import sys

      raw_path = sys.argv[1]
      max_bytes = int(os.environ["MAX_BYTES"])
      byte_offset = int(os.environ["BYTE_OFFSET"])
      start_line = os.environ.get("START_LINE") or ""
      end_line = os.environ.get("END_LINE") or ""

      data = open(raw_path, "rb").read()
      truncated = len(data) > max_bytes
      data = data[:max_bytes]
      text = data.decode("utf-8", errors="replace")

      line_start = None
      line_end = None
      if start_line:
        line_start = int(start_line)
        line_end = int(end_line) if end_line else line_start + 99
        lines = text.splitlines(keepends=True)
        text = "".join(lines[line_start - 1:line_end])

      print(json.dumps({
        "owner": os.environ["OWNER"],
        "repo": os.environ["REPO"],
        "path": os.environ["PATH_IN_REPO"],
        "ref": os.environ["REF"],
        "byte_offset": byte_offset,
        "max_bytes": max_bytes,
        "start_line": line_start,
        "end_line": line_end,
        "content": text,
        "content_bytes": len(text.encode("utf-8")),
        "truncated_by_max_bytes": truncated,
      }))
      PY
---
<!--
## GitHub MCP Pagination Wrappers

This shared workflow provides mcp-script wrappers for `list_workflows`, `list_label`,
and `get_file_contents_excerpt` that add bounded response controls missing from the
corresponding GitHub MCP calls.

The built-in `list_label` GitHub MCP tool returns up to 100 labels regardless of any
`perPage` argument (it uses a hardcoded GraphQL `labels(first: 100)` query). The
`list_workflows` built-in GitHub MCP tool uses a non-standard `per_page` parameter
(snake_case), inconsistent with every other list-style MCP tool which uses camelCase
`perPage`, and the limit was silently ignored. These wrappers call the GitHub REST API
directly so `perPage` is respected on every call, using the camelCase convention.

### Available Tools

1. **list_workflows** — List Actions workflows, calls `GET /repos/{owner}/{repo}/actions/workflows`
2. **list_label** — List repository labels, calls `GET /repos/{owner}/{repo}/labels`
3. **get_file_contents_excerpt** — Read a bounded file excerpt, calls `GET /repos/{owner}/{repo}/contents/{path}` with a byte `Range` header and optional line filtering

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
| perPage | number | No | 10 | Results per page (1–100) |
| page | number | No | 1 | Page number |

#### list_label

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| owner | string | Yes | - | Repository owner |
| repo | string | Yes | - | Repository name |
| perPage | number | No | 10 | Results per page (1–100) |
| page | number | No | 1 | Page number |

#### get_file_contents_excerpt

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| owner | string | Yes | - | Repository owner |
| repo | string | Yes | - | Repository name |
| path | string | Yes | - | File path in the repository |
| ref | string | No | `GITHUB_SHA` or default branch | Git ref to read |
| byteOffset | number | No | 0 | Zero-based byte offset |
| maxBytes | number | No | 20000 | Maximum bytes to fetch before line filtering (1–200000) |
| startLine | number | No | - | One-based line number to start returning within the fetched byte window |
| endLine | number | No | `startLine + 99` | One-based line number to stop returning within the fetched byte window |

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

### get_file_contents_excerpt Response

```json
{
  "owner": "owner",
  "repo": "repo",
  "path": "README.md",
  "ref": "main",
  "byte_offset": 0,
  "max_bytes": 20000,
  "start_line": 1,
  "end_line": 40,
  "content": "# Project\n\n...",
  "content_bytes": 1234,
  "truncated_by_max_bytes": false
}
```

### Source scripts

- `.github/skills/github-workflows-query/query-workflows.sh`
- `.github/skills/github-labels-query/query-labels.sh`
-->
