---
description: "Shared workflow defining agentic-workflows, serena, and playwright tools for import"
tools:
  agentic-workflows: true
  serena:
    - go
    - typescript
  playwright:
    version: "v1.41.0"
    allowed_domains:
      - "example.com"
      - "github.com"
network:
  allowed:
    - playwright
permissions:
  actions: read
  contents: read
---

# Importable Tools Configuration

This shared workflow provides common tool configurations that can be imported by other workflows.

## Included Tools

- **agentic-workflows**: Workflow introspection and analysis
- **serena**: Code intelligence for Go and TypeScript
- **playwright**: Browser automation with example.com and github.com access
