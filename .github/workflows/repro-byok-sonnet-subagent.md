---
private: true
emoji: "🐛"
description: >
  Repro: sub-agent launched with model "sonnet" fails with 400 when the Copilot CLI
  uses wireApi=responses, because claude-sonnet-4.6 does not support the Responses API.
on:
  workflow_dispatch:
name: Repro - BYOK Sonnet Sub-Agent Responses API Error
permissions:
  contents: read
engine:
  id: copilot
  model: gpt-5-mini
  bare: true
  env:
    # BYOK: point the Copilot CLI directly at the Copilot API so that model
    # routing behaves the same as in the AWF API-proxy environment.
    COPILOT_PROVIDER_BASE_URL: https://api.githubcopilot.com
    COPILOT_PROVIDER_BEARER_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
    # Force the Responses API wire format – this is what the AWF docker-compose
    # injects for all agent containers (COPILOT_PROVIDER_WIRE_API=responses),
    # which triggers the 400 error for Anthropic models.
    COPILOT_PROVIDER_WIRE_API: responses
strict: true
timeout-minutes: 10
safe-outputs:
  noop: false
  create-issue:
    expires: 2h
    group: true
    close-older-issues: true
    close-older-key: "repro-byok-sonnet-subagent"
    labels: [automation, testing]
---

# Repro: BYOK Sonnet Sub-Agent — Responses API 400

This workflow is a **standalone reproduction** of the sub-agent failures
reported in
[issue #39849, comment 4830631969](https://github.com/github/gh-aw/issues/39849#issuecomment-4830631969).

## Bug Summary

In the PR Sous Chef run
[28359605525](https://github.com/github/gh-aw/actions/runs/28359605525),
five `pr-processor` sub-agents were launched and all returned
`skip_reason: sub_agent_error` with no output.

**Root cause** (from `sandbox/agent/logs/process-*.log`):

```
400 model claude-sonnet-4.6 does not support Responses API.
```

**Failure chain:**

1. The AWF docker-compose injects `COPILOT_PROVIDER_WIRE_API=responses` into
   every agent container, so the Copilot CLI always uses the OpenAI Responses
   API wire format.
2. The PR Sous Chef main agent (gpt-5-mini) passes `model: "sonnet"` when
   calling the `task` tool to launch sub-agents (the optional model-override
   parameter in the tool schema).
3. The API proxy resolves the `sonnet` alias → `claude-sonnet-4.6` (Anthropic).
4. The Copilot CLI sends a Responses-API-format request for `claude-sonnet-4.6`.
5. Anthropic rejects: `400 unsupported_api_for_model`.
6. All five sub-agents complete with no output → `skip_reason: sub_agent_error`.

**Note:** The `pr-processor` agent definition declares `model: claude-haiku-4.5`,
but the main agent overrides this via the task-tool `model` parameter.

## Repro Configuration

This workflow reproduces the failure by:

- Using BYOK mode (`COPILOT_PROVIDER_BASE_URL` + `COPILOT_PROVIDER_WIRE_API: responses`)
  so the same wire-format constraint is active.
- Having the main agent call `dummy-sonnet-sub` with `model: "sonnet"` as an
  explicit model override (matching what PR Sous Chef does).
- The sub-agent will attempt a Responses-API request for `claude-sonnet-4.6`
  and receive a `400 unsupported_api_for_model` error.

## Instructions

Use the `task` tool to call the `dummy-sonnet-sub` custom agent with:
- `model: "sonnet"` (explicit override, exactly as PR Sous Chef does)
- `prompt: "Say hello and report your model name."`

After the sub-agent completes (or fails), collect the result. Then:

- If the sub-agent produced output: create an issue titled
  `[repro] Sonnet sub-agent unexpectedly succeeded — run ${{ github.run_id }}`
  and mark it as a regression fix opportunity.
- If the sub-agent returned an error (expected): call `noop` with message
  `repro-confirmed: sub-agent with model=sonnet failed as expected (400 unsupported_api_for_model)`.

Only one safe-output action is needed.

## agent: `dummy-sonnet-sub`
---
description: Minimal sub-agent for the Responses-API repro — says hello only
model: claude-sonnet-4.6
---
You are a test sub-agent. Say "hello from dummy-sonnet-sub" and nothing else.
Do not use any tools.
