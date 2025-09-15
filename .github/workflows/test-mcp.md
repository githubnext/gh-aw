---
on:
  workflow_dispatch: {}

permissions: read-all

tools:
  github:
    docker_image_version: "latest"
  playwright:
    docker_image_version: "v1.41.0"
    allowed_domains: ["example.com", "*.github.com"]

safe-outputs:
  create-issue:
    title-prefix: "[Test] "
  add-issue-comment:

engine: claude
---

# Test MCP Configuration

This is a test workflow to demonstrate MCP configuration generation and server launching.