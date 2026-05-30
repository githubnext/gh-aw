---
tools:
  github:
    mode: gh-proxy
---

<!--
## gh-proxy Mode

This shared workflow enables `tools.github.mode: gh-proxy`, which provides a pre-authenticated `gh` CLI binary for all GitHub interactions. The agent prompt is auto-injected.

### Usage

```yaml
imports:
  - shared/gh.md
```

### Authentication

The `gh` binary is pre-authenticated using `GITHUB_TOKEN` with permissions based on the workflow's `permissions` configuration.
-->
