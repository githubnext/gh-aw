---
name: github-discussion-query
description: Query GitHub discussions efficiently with jq argument support for filtering
---

# GitHub Discussion Query Skill

This skill provides efficient querying of GitHub discussions with built-in jq filtering support.

## Important: jq Parameter is Optional

The `--jq` parameter is **optional**. When called without `--jq`, this skill returns **schema and data size information** instead of the full data.
This prevents overwhelming responses with large datasets and helps you understand the data structure before querying.

Use `--jq '.'` to get all data, or use a more specific filter for targeted results.

## Usage

Use this skill to query discussions from the current repository or any specified repository.

### Basic Query (Returns Schema Only)

To list discussions from the current repository:

```bash
./query-discussions.sh
# Returns schema and data size, not full data
```

### Get All Data

To get all discussion data:

```bash
./query-discussions.sh --jq '.'
```

### With Repository

To query a specific repository:

```bash
./query-discussions.sh --repo owner/repo
```

### With jq Filtering

Use the `--jq` argument to filter and transform the output:

```bash
# Get discussion numbers and titles
./query-discussions.sh --jq '.[] | {number, title}'

# Get discussions by a specific author
./query-discussions.sh --jq '.[] | select(.author.login == "username")'

# Get discussions in a specific category
./query-discussions.sh --jq '.[] | select(.category.name == "Ideas")'

# Get answered discussions
./query-discussions.sh --jq '.[] | select(.answer != null)'

# Count discussions by category
./query-discussions.sh --jq 'group_by(.category.name) | map({category: .[0].category.name, count: length})'
```

### Common Options

- `--limit`: Maximum number of discussions to fetch. Default: 30
- `--repo`: Repository in owner/repo format. Default: current repo
- `--jq`: (Optional) jq expression for filtering/transforming output. If omitted, returns schema info

### Example Queries

**Find discussions with many comments:**
```bash
./query-discussions.sh --jq '.[] | select(.comments.totalCount > 5) | {number, title, comments: .comments.totalCount}'
```

**Get unanswered discussions:**
```bash
./query-discussions.sh --jq '.[] | select(.answer == null) | {number, title, category: .category.name}'
```

**List discussions with their labels:**
```bash
./query-discussions.sh --jq '.[] | {number, title, labels: [.labels[].name]}'
```

**Find discussions by category:**
```bash
./query-discussions.sh --jq '.[] | select(.category.name == "Q&A") | {number, title, author: .author.login}'
```

**Get recently updated discussions:**
```bash
./query-discussions.sh --jq 'sort_by(.updatedAt) | reverse | .[0:10] | .[] | {number, title, updatedAt}'
```

## Output Format

The script outputs JSON by default, making it easy to pipe through jq for additional processing.

## Requirements

- GitHub CLI (`gh`) authenticated
- `jq` for filtering (installed by default on most systems)
