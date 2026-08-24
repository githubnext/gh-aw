---
private: true
emoji: "🧪"
description: Smoke test for Claude engine with Copilot model syntax on GitHub Inference that posts a concise PR summary comment
on:
  slash_command:
    name: smoke-claude-on-copilot
    strategy: centralized
    events: [pull_request, pull_request_comment]
  status-comment: true
permissions:
  contents: read
  pull-requests: read
name: Smoke Claude on Copilot
model: copilot/claude-sonnet-4.6
engine:
  id: claude
  bare: true
strict: true
imports:
  - shared/reporting.md
tools:
  github:
    mode: gh-proxy
safe-outputs:
  allowed-domains: [default-safe-outputs]
  add-comment:
    max: 1
    hide-older-comments: true
timeout-minutes: 10
sandbox:
  agent:
    id: awf
---

# Smoke Test: Claude on GitHub Inference PR Summary

Goal: validate that Claude with a `copilot/...` model can read the current pull request and post one concise summary comment.

1. If this run is not in PR context, call `noop` and stop.
2. Read the current PR details for `${{ github.event.pull_request.number }}` from `${{ github.repository }}`.
3. Produce a short summary with:
   - PR title
   - author
   - file count
   - a 2-3 sentence high-level summary of what changed
4. Post exactly one `add_comment` safe output to the current PR with this summary.

Keep the comment compact (max 8 lines).