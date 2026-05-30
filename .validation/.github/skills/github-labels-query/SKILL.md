---
name: github-labels-query
description: List GitHub repository labels with per_page pagination support.
---

# GitHub Labels Query Skill

List GitHub repository labels with efficient pagination using the `--per-page` flag.

## Usage

Use this script to list labels from any repository with controlled page sizes.

### Basic Usage

```bash
./query-labels.sh --owner github --repo gh-aw
# Returns 10 labels (default per_page=10)
```

### Pagination

```bash
# Get the first label only
./query-labels.sh --owner github --repo gh-aw --per-page 1

# Get 25 labels starting from page 2
./query-labels.sh --owner github --repo gh-aw --per-page 25 --page 2
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
  "labels": [
    {
      "id": 12345,
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

## Source

Calls the GitHub REST API:
`GET /repos/{owner}/{repo}/labels?per_page={n}&page={n}`
