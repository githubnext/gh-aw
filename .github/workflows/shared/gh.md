---
safe-inputs:
  mode: stdio
  gh:
    description: "Execute any gh CLI command. Provide the full command after 'gh' (e.g., args: 'pr list --limit 5'). The tool will run: gh <args>. Use single quotes ' for complex args to avoid shell interpretation issues."
    inputs:
      args:
        type: string
        description: "Arguments to pass to gh CLI (without the 'gh' prefix). Examples: 'pr list --limit 5', 'issue view 123', 'api repos/{owner}/{repo}'"
        required: true
    env:
      GH_AW_GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      GH_TOKEN=$GH_AW_GH_TOKEN gh $INPUT_ARGS
---

**IMPORTANT**: Always use the `gh` safe-input tool for GitHub CLI commands instead of running `gh` directly via bash. The safe-input tool has proper authentication configured with `GITHUB_TOKEN`, while bash commands do not have GitHub CLI authentication by default.

**Correct**:
```
Use the gh safe-input tool with args: "pr list --limit 5"
Use the gh safe-input tool with args: "issue view 123"
```

**Incorrect**:
```
Run: gh pr list --limit 5  ❌ (No authentication in bash)
Execute bash: gh issue view 123  ❌ (No authentication in bash)
```

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
