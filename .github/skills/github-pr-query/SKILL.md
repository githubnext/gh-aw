---
name: github-pr-query
description: Query GitHub pull requests efficiently with jq argument support for filtering
---

# GitHub Pull Request Query Skill

This skill provides efficient querying of GitHub pull requests with built-in jq filtering support.

## Important: jq Parameter is Optional

The `--jq` parameter is **optional**. When called without `--jq`, this skill returns **schema and data size information** instead of the full data.
This prevents overwhelming responses with large datasets and helps you understand the data structure before querying.

Use `--jq '.'` to get all data, or use a more specific filter for targeted results.

## Usage

Use this skill to query pull requests from the current repository or any specified repository.

### Basic Query (Returns Schema Only)

To list pull requests from the current repository:

```bash
./query-prs.sh
# Returns schema and data size, not full data
```

### Get All Data

To get all PR data:

```bash
./query-prs.sh --jq '.'
```

### With Repository

To query a specific repository:

```bash
./query-prs.sh --repo owner/repo
```

### With jq Filtering

Use the `--jq` argument to filter and transform the output:

```bash
# Get only open PRs
./query-prs.sh --jq '.[] | select(.state == "open")'

# Get PR numbers and titles
./query-prs.sh --jq '.[] | {number, title}'

# Get PRs by a specific author
./query-prs.sh --jq '.[] | select(.author.login == "username")'

# Get merged PRs from last week
./query-prs.sh --jq '.[] | select(.mergedAt != null)'

# Count PRs by state
./query-prs.sh --jq 'group_by(.state) | map({state: .[0].state, count: length})'
```

### Common Options

- `--state`: Filter by state (open, closed, merged, all). Default: open
- `--limit`: Maximum number of PRs to fetch. Default: 30
- `--repo`: Repository in owner/repo format. Default: current repo
- `--jq`: (Optional) jq expression for filtering/transforming output. If omitted, returns schema info

### Example Queries

**Find large PRs (many changed files):**
```bash
./query-prs.sh --jq '.[] | select(.changedFiles > 10) | {number, title, changedFiles}'
```

**Get PRs awaiting review:**
```bash
./query-prs.sh --jq '.[] | select(.reviewDecision == "REVIEW_REQUIRED") | {number, title, author: .author.login}'
```

**List PRs with their labels:**
```bash
./query-prs.sh --jq '.[] | {number, title, labels: [.labels[].name]}'
```

## Output Format

The script outputs JSON by default, making it easy to pipe through jq for additional processing.

## Requirements

- GitHub CLI (`gh`) authenticated
- `jq` for filtering (installed by default on most systems)
