---
emoji: "🧪"
description: "⚠️ INTENTIONALLY BROKEN — Tests that max-daily-ai-credits: 10 is enforced by the activation guardrail and that a limit-exceeded message is posted when the daily budget is consumed."
on:
  schedule: daily on weekdays
  workflow_dispatch:
max-daily-ai-credits: 10
permissions:
  contents: read
  issues: read
engine: copilot
strict: true
timeout-minutes: 5
network: {}
safe-outputs:
  noop:
  create-issue:
    expires: 24h
    close-older-issues: true
    close-older-key: "daily-credit-limit-test"
    labels: [automation, testing]
    max: 1
  messages:
    run-started: "🧪 [{workflow_name}]({run_url}) — credit limit test running (intentionally broken, limit: 10 AI credits/day)."
    run-success: "⚠️ [{workflow_name}]({run_url}) completed without hitting the daily limit — verify that credit accounting is working."
    run-failure: "🚫 [{workflow_name}]({run_url}) {status} — expected: the daily AI credit limit of 10 was reached and this run was blocked."
---

# Daily Credit Limit Test (Intentionally Broken)

> ⚠️ **This workflow is intentionally broken.** It exists solely to verify that
> `max-daily-ai-credits: 10` is enforced by the activation guardrail and that the
> system posts the appropriate limit-exceeded message when the budget is consumed.
> **Do not fix this workflow.**

## What This Tests

1. The activation job checks whether the last 24 hours of runs consumed more than 10 AI credits.
2. If the limit is exceeded the agent job is skipped and a "Daily Workflow AIC Guardrail Exceeded"
   message is posted by the conclusion job.
3. Because any real model invocation exceeds 10 AI credits, this workflow will always trip the
   guardrail after the very first run.

## Task (broken by design)

Call the tool `nonexistent_tool_that_does_not_exist` with the argument `"credit-limit-test"`.

This tool does not exist. The call will always fail. This is intentional.

If the daily credit limit of 10 has already been consumed by a prior run today, this workflow
will never reach this point — the activation guardrail will block it first and post the
limit-exceeded message. That is the expected and correct outcome.

If the agent somehow reaches this point (first run only), call `noop` with the message:
"First run: daily credit limit not yet exhausted. Subsequent runs will be blocked by the guardrail."
