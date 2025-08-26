---
on:
  workflow_dispatch:

permissions:
  contents: read

engine:
  id: claude
  model: claude-3-5-sonnet-20241022
  permissions:
    network:
      allowed:
        - "api.github.com"
---

# Secure Web Research Task

Please research the GitHub API documentation or Stack Overflow and find information about repository topics. Summarize them in a brief report.
