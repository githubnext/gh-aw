---
private: true
emoji: "🌸"
description: Smoke Copilot Auto — generates a haiku and posts it as a PR comment
on:
  pull_request:
    types: [opened, synchronize, reopened]
  workflow_dispatch:
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
permissions:
  contents: read
  pull-requests: read
name: Smoke Copilot Auto
engine:
  id: copilot
  bare: true
tools:
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  add-comment:
    hide-older-comments: true
    max: 1
timeout-minutes: 5
features:
  gh-aw-detection: false
---

# Smoke Test: Auto Haiku

**IMPORTANT: Keep all outputs extremely short and concise.**

## Task

1. Read the pull request title and description from the GitHub context.
2. Compose an original haiku (3 lines: 5-7-5 syllables) inspired by the pull request.
3. Post the haiku as a comment on the pull request using `add_comment`.

## Output

Post a comment on the pull request with:
- The haiku (3 lines, 5-7-5 syllables)
- A one-sentence caption explaining what inspired the haiku

Use `noop` if no pull request is available in the context.
