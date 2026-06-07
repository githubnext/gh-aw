---
emoji: "🧪"
description: "⚠️ INTENTIONALLY FAILS — Tests that max-ai-credits: 1 is enforced by the AWF firewall and that the per-run budget guardrail cuts off the agent."
on:
  schedule: daily around 10:30
  workflow_dispatch:
max-ai-credits: 1
max-daily-ai-credits: -1
permissions:
  contents: read
  issues: read
engine:
  id: copilot
strict: true
timeout-minutes: 5
network: {}
safe-outputs:
  noop:
  create-issue:
    expires: 24h
    close-older-issues: true
    close-older-key: "daily-max-ai-credits-test"
    labels: [automation, testing]
    max: 1
  messages:
    run-started: "🧪 [{workflow_name}]({run_url}) — per-run AI credit limit test running (intentionally fails, limit: 1 AI credit/run)."
    run-success: "⚠️ [{workflow_name}]({run_url}) completed without hitting the per-run limit of 1 AI credit — verify that max-ai-credits enforcement is working."
    run-failure: "🚫 [{workflow_name}]({run_url}) {status} — expected: the per-run AI credit limit of 1 was reached and the AWF firewall cut off the agent."
---

# Daily Max AI Credits Test (Intentionally Fails)

> ⚠️ **This workflow is intentionally broken.** It exists solely to verify that
> `max-ai-credits: 1` is enforced by the AWF firewall and that the agent is
> cut off when the per-run budget is exhausted.
> **Do not fix this workflow.**

## What This Tests

1. The AWF firewall enforces the `max-ai-credits` per-run budget.
2. Once the agent consumes more than 1 AI credit in a single run, the firewall cuts off the LLM API.
3. Because any real model invocation exceeds 1 AI credit, this workflow will always fail immediately.

## Task (broken by design)

Call `noop` with the message: "Starting max-ai-credits guardrail test."

This invocation consumes AI credits. Since the per-run budget is 1, the AWF firewall will
cut off the agent before or immediately after this call, causing the run to fail.

That failure is the expected and correct outcome.

If the workflow somehow completes without hitting the per-run limit, call `noop` with the message:
"Per-run credit limit not exceeded — verify that max-ai-credits: 1 is enforced by the AWF firewall."
