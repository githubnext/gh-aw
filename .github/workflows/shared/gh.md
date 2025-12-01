---
safe-inputs:
  gh:
    description: "Execute any gh CLI command. Provide the full command after 'gh' (e.g., args: 'pr list --limit 5'). The tool will run: gh <args>"
    inputs:
      args:
        type: string
        description: "Arguments to pass to gh CLI (without the 'gh' prefix). Examples: 'pr list --limit 5', 'issue view 123', 'api repos/{owner}/{repo}'"
        required: true
    env:
      GH_TOKEN: ${{ github.token }}
    run: |
      gh $INPUT_ARGS
---
<!--
## gh CLI Safe Input Tool

A simple safe-input tool that wraps the GitHub CLI (`gh`).

### Usage

```yaml
imports:
  - shared/gh.md
```

### Invocation

Provide gh CLI arguments via the `args` parameter:

```
gh with args: "pr list --limit 5"
gh with args: "issue view 123"
gh with args: "api repos/{owner}/{repo}"
gh with args: "pr view 456 --json title,body,author"
```

The tool executes: `gh <args>`

### Authentication

Uses `GH_TOKEN` from `github.token` which has permissions based on the workflow's `permissions` configuration.
-->
