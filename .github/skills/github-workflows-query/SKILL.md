---
name: github-workflows-query
description: List GitHub Actions workflows with per_page pagination support.
---

# GitHub Workflows Query Skill

List GitHub Actions workflows with efficient pagination using the `--per-page` flag.

## Usage

Use this script to list workflows from any repository with controlled page sizes.

### Basic Usage

```bash
./query-workflows.sh --owner github --repo gh-aw
# Returns 10 workflows (default per_page=10)
```

### Pagination

```bash
# Get the first workflow only
./query-workflows.sh --owner github --repo gh-aw --per-page 1

# Get 50 workflows starting from page 2
./query-workflows.sh --owner github --repo gh-aw --per-page 50 --page 2
```

## Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `--owner` | Yes | - | Repository owner (username or organization) |
| `--repo` | Yes | - | Repository name |
| `--per-page` | No | 10 | Results per page (1–100) |
| `--page` | No | 1 | Page number |

## Output

Returns JSON with the following fields:

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

## Source

Calls the GitHub REST API:
`GET /repos/{owner}/{repo}/actions/workflows?per_page={n}&page={n}`
