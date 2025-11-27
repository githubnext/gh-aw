---
description: Smoke test workflow that validates Codex engine functionality by reviewing recent PRs every 6 hours
on: 
  schedule:
    - cron: "0 0,6,12,18 * * *"  # Every 6 hours
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "hooray"
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Codex
engine: codex
strict: false
network:
  allowed:
    - defaults
    - github
    - playwright
tools:
  github:
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
      allowed: [smoke-codex]
timeout-minutes: 10
---

# Smoke Test: Codex Engine Validation

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

## Test Requirements

1. **GitHub MCP Testing**: Review the last 2 merged pull requests in ${{ github.repository }}
2. **File Writing Testing**: Create a test file `/tmp/smoke-test-codex-${{ github.run_id }}.txt` with content "Smoke test passed for Codex at $(date)"
3. **Bash Tool Testing**: Execute bash commands to verify file creation was successful (use `cat` to read the file back)
4. **Playwright MCP Testing**: Use playwright to navigate to https://github.com and verify the page title contains "GitHub"

## Output

**ENIGMATIC STYLE**: Write your output like cryptic riddles and mysterious prophecies. Be poetic and mystical.

Add a **very brief** comment (max 5-10 lines) to the current pull request in enigmatic style:
- Speak in riddles: "The oracle has spoken...", "As the ancients foretold..."
- Use cryptic metaphors for each test result
- Use ✅ or ❌ as "sacred symbols"
- End with a mysterious verdict: "The prophecy is fulfilled... ✨" or "The shadows whisper of failures..."

Example tone: "The GitHub spirit was summoned... ✅ Two scrolls from the PR archives were retrieved. The ancient ritual proceeds..."

If all tests pass, add the label `smoke-codex` to the pull request.