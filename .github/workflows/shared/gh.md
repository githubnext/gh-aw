---
safe-inputs:
  gh:
    description: "Execute any gh CLI command with access to the repository's GITHUB_TOKEN. Supports all gh subcommands including pr, issue, api, repo, run, etc."
    inputs:
      args:
        type: string
        description: "The gh CLI command arguments (e.g., 'pr list --limit 5', 'issue view 123', 'api repos/{owner}/{repo}')"
        required: true
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Execute the gh CLI command with the provided arguments
      gh $INPUT_ARGS
---
<!--
## gh CLI Safe Input Tool

This shared workflow provides a `gh` safe-input tool that allows agents to execute any GitHub CLI (`gh`) command using the repository's `GITHUB_TOKEN`.

### Usage

Import this shared workflow to get access to the `gh` tool:

```yaml
imports:
  - shared/gh.md
```

The agent can then use the tool to execute any gh CLI command:
- `gh` with `args: "pr list --limit 5"` lists the last 5 PRs
- `gh` with `args: "issue view 123"` views issue #123
- `gh` with `args: "api repos/{owner}/{repo}"` calls the GitHub API
- `gh` with `args: "pr view 456 --json title,body,author"` gets PR details as JSON

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| args | string | yes | The gh CLI command arguments |

### Example Commands

```bash
# List PRs
gh args: "pr list --limit 10"

# Get PR details as JSON
gh args: "pr view 123 --json number,title,body,author,state,createdAt,mergedAt"

# List issues
gh args: "issue list --state open --limit 20"

# Call the GitHub API
gh args: "api repos/{owner}/{repo}/pulls/123"

# Search code
gh args: "search code 'function myFunc' --repo owner/repo"
```

### Security

This tool uses the repository's `GITHUB_TOKEN` which has limited permissions based on the workflow's `permissions` configuration. The agent can only perform actions that the token allows.
-->
