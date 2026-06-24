---
private: true
emoji: "🧪"
description: Smoke test for Claude engine on GitHub Inference that posts a concise PR summary comment
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
engine:
  id: claude
  model-provider: github
  model: claude-haiku-4.5
  bare: true
strict: true
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
    sudo: false
---

# Smoke Test: Claude on GitHub Inference PR Summary

Goal: validate that Claude with `model-provider: github` can read the current pull request and post one concise summary comment.

1. If this run is not in PR context, call `noop` and stop.
2. Read the current PR details for `${{ github.event.pull_request.number }}` from `${{ github.repository }}`.
3. Produce a short summary with:
   - PR title
   - author
   - file count
   - a 2-3 sentence high-level summary of what changed
4. Post exactly one `add_comment` safe output to the current PR with this summary.

Keep the comment compact (max 8 lines).

{{#runtime-import shared/noop-reminder.md}}
