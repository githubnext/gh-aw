---
name: github-issue-query
description: Query GitHub issues efficiently with jq argument support for filtering
---

# GitHub Issue Query Skill

This skill provides efficient querying of GitHub issues with built-in jq filtering support.

## Usage

Use this skill to query issues from the current repository or any specified repository.

### Basic Query

To list issues from the current repository:

```bash
./query-issues.sh
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
- `--jq`: jq expression for filtering/transforming output

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

**Find old issues (created over 30 days ago):**
```bash
./query-issues.sh --jq '.[] | select(.createdAt < (now - 2592000 | strftime("%Y-%m-%dT%H:%M:%SZ")))'
```

## Output Format

The script outputs JSON by default, making it easy to pipe through jq for additional processing.

## Requirements

- GitHub CLI (`gh`) authenticated
- `jq` for filtering (installed by default on most systems)
