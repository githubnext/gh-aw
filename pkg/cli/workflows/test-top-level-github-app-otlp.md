---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
  id-token: write

github-app:
  app-id: ${{ vars.APP_ID }}
  private-key: ${{ secrets.APP_PRIVATE_KEY }}
observability:
  otlp:
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    github-app:
      type: github-oidc
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# Top-Level GitHub App with OTLP OIDC + GitHub MCP

This workflow exercises `observability.otlp.github-app` token minting together with
top-level `github-app` fallback used by `tools.github`.
