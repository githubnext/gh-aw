---
safe-inputs:
  gh:
    description: "Execute any gh CLI command using the GH_TOKEN environment variable (set from GITHUB_TOKEN). Supports all gh subcommands including pr, issue, api, repo, run, etc."
    inputs:
      args:
        type: string
        description: "The gh CLI command arguments (e.g., 'pr list --limit 5', 'issue view 123', 'api repos/{owner}/{repo}')"
        required: true
    env:
      GH_TOKEN: ${{ github.token }}
    run: |
      # Execute the gh CLI command with the provided arguments
      # Note: INPUT_ARGS is validated as a string by the safe-inputs system
      # The gh CLI handles its own argument parsing and security
      gh $INPUT_ARGS
---
<!--
## gh CLI Safe Input Tool

This shared workflow provides a `gh` safe-input tool that allows agents to execute any GitHub CLI (`gh`) command using the `GH_TOKEN` environment variable (populated from `GITHUB_TOKEN`).

### Usage

Import this shared workflow to get access to the `gh` tool:

```yaml
imports:
  - shared/gh.md
```

The agent can then use the tool to execute any gh CLI command by calling the `gh` safe-input with args parameter:
- Call `gh` with `args: "pr list --limit 5"` to list the last 5 PRs
- Call `gh` with `args: "issue view 123"` to view issue #123  
- Call `gh` with `args: "api repos/{owner}/{repo}"` to call the GitHub API
- Call `gh` with `args: "pr view 456 --json title,body,author"` to get PR details as JSON

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| args | string | yes | The gh CLI command arguments |

### Example Tool Invocations

When using the safe-input tool, provide the args parameter:

```
gh with args: "pr list --limit 10"
gh with args: "pr view 123 --json number,title,body,author,state,createdAt,mergedAt"
gh with args: "issue list --state open --limit 20"
gh with args: "api repos/{owner}/{repo}/pulls/123"
gh with args: "search code 'function myFunc' --repo owner/repo"
```

### Security

This tool uses the `GH_TOKEN` environment variable (populated from `GITHUB_TOKEN`) which has limited permissions based on the workflow's `permissions` configuration. The agent can only perform actions that the token allows.
-->
