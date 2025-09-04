---
on:
  pull_request:
    branches:
      - main
    forks: []
  workflow_dispatch:

permissions:
  contents: read

engine:
  id: claude

network:
  allowed:
    - "docs.github.com"

tools:
  claude:
    allowed:
      WebFetch:
      WebSearch:
---

# Secure Web Research Task

Please research the GitHub API documentation or Stack Overflow and find information about repository topics. Summarize them in a brief report.
