---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
observability:
  otlp:
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    github-app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  create-issue:
    title-prefix: "[automated] "
engine: copilot
---

# OTLP GitHub App token minting with GitHub MCP

This workflow exercises `observability.otlp.github-app` token minting together with
`tools.github`.
