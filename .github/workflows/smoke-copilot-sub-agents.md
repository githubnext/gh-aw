---
private: true
emoji: "🧪"
description: Smoke Copilot SDK inline sub-agents
on:
  schedule:
    - cron: daily
  workflow_dispatch:
permissions:
  contents: read
  copilot-requests: write
name: Smoke Copilot Sub Agents
strict: true
engine:
  id: copilot
  copilot-sdk: true
  model: gpt-5.3-codex
  bare: true
safe-outputs:
  create-issue:
    expires: 2h
    group: true
    close-older-issues: true
    close-older-key: "smoke-copilot-sub-agents"
    labels: [automation, testing]
timeout-minutes: 10
features:
  gh-aw-detection: false
---

# Smoke Test: Copilot SDK Inline Sub-Agents

**IMPORTANT: Keep all outputs extremely short.**

## Tasks

1. Call `haiku-whoami`, `mini-whoami`, and `nano-whoami` exactly once each.
2. Ask each sub-agent exactly this question: `who am i?`
3. Check the exact responses:
   - `haiku-whoami` → `claude-haiku-4.5`
   - `mini-whoami` → `gpt-5-mini`
   - `nano-whoami` → `gpt-5-nano`
4. Do not use any other agent or any unnecessary tool calls.

## Output

Always create an issue titled **"Smoke Test: Copilot Sub Agents - ${{ github.run_id }}"** with:
- One line per sub-agent showing expected value, actual value, and ✅/❌
- Overall status: PASS only if all three exact matches succeed, otherwise FAIL
- Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

## agent: `haiku-whoami`
---
description: Returns the Haiku model identity for smoke testing
model: claude-haiku-4.5
---
When asked `who am i?`, reply with exactly:

`claude-haiku-4.5`

No extra words, punctuation, or formatting.

## agent: `mini-whoami`
---
description: Returns the GPT-5 mini model identity for smoke testing
model: gpt-5-mini
---
When asked `who am i?`, reply with exactly:

`gpt-5-mini`

No extra words, punctuation, or formatting.

## agent: `nano-whoami`
---
description: Returns the GPT-5 nano model identity for smoke testing
model: gpt-5-nano
---
When asked `who am i?`, reply with exactly:

`gpt-5-nano`

No extra words, punctuation, or formatting.
