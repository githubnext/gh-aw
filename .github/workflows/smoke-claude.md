---
description: Smoke test workflow that validates Claude engine functionality by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "heart"
permissions:
  contents: read
  issues: read
  pull-requests: read
  
name: Smoke Claude
engine:
  id: claude
  max-turns: 15
strict: false
imports:
  - shared/mcp-pagination.md
network:
  allowed:
    - defaults
    - github
    - playwright
tools:
  github:
    toolsets: [repos, pull_requests]
  playwright:
    allowed_domains:
      - github.com
  edit:
  bash:
    - "*"
  serena: ["go"]
safe-outputs:
    add-comment:
    create-issue:
    add-labels:
      allowed: [smoke-claude]
timeout-minutes: 10
---

# Smoke Test: Claude Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/smoke-test-claude-${{ github.run_id }}.txt` with content "Smoke test passed for Claude at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"

## Output

**GRAPHIC NOVEL DIALOG STYLE**: Write your output like dramatic comic book speech bubbles with bold action words and visual emphasis.

Add a **very brief** comment (max 5-10 lines) to the current pull request in graphic novel style:
- Use dramatic action words: "**WHOOSH!**", "**KAPOW!**", "**BOOM!**"
- Frame each test like a panel: "*[Panel 1]* The hero queries the GitHub API..."
- Use ✅ or ❌ as visual "impact markers"
- End with a dramatic finale: "**THE END** — MISSION: ACCOMPLISHED!" or "**TO BE CONTINUED...** — FAILURES DETECTED!"

Example tone: "*[Panel 1]* **SWOOSH!** The GitHub MCP springs into action! ✅ Two PRs retrieved in a flash!"

If all tests pass, add the label `smoke-claude` to the pull request.