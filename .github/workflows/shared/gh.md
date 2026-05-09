---
tools:
  github:
    mode: gh-proxy
---

**IMPORTANT**: The `gh` CLI is pre-authenticated. Always use `gh` commands directly for GitHub operations (listing issues, pull requests, reading repository contents, etc.). No GitHub MCP server is available; `gh` is your only way to interact with GitHub.

**Correct**:
```
Run: gh pr list --limit 5
Run: gh issue view 123
Run: gh api repos/{owner}/{repo}
```

**Incorrect**:
```
Use the mcpscripts-gh tool  ❌ (Tool no longer available - use gh CLI directly)
```

<!--
## gh-proxy Mode

This shared workflow enables `tools.github.mode: gh-proxy`, which provides a pre-authenticated `gh` CLI binary for all GitHub interactions.

### Usage

```yaml
imports:
  - shared/gh.md
```

### Authentication

The `gh` binary is pre-authenticated using `GITHUB_TOKEN` with permissions based on the workflow's `permissions` configuration.
-->
