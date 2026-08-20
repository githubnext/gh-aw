---
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
---

<!--
## cli-proxy + gh-proxy Mode

This shared workflow bundles `tools.cli-proxy: true` together with `tools.github.mode: gh-proxy`.
These two settings are almost always enabled together: `cli-proxy` provides a pre-authenticated
`gh` CLI binary via `bash`, and `github.mode: gh-proxy` configures the GitHub MCP server to route
through that same proxy. Bundling them here prevents the two settings from drifting independently.

### Usage

```yaml
imports:
  - shared/gh-cli-proxy.md
```

### Authentication

The `gh` binary is pre-authenticated using `GITHUB_TOKEN` with permissions based on the workflow's `permissions` configuration.

Note: this is kept separate from `shared/gh.md` (which only sets `tools.github.mode: gh-proxy`) so that
`shared/gh.md`'s existing consumers are unaffected.
-->
