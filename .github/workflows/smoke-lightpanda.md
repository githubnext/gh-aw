---
private: true
emoji: "🐼"
description: Smoke test workflow that validates Lightpanda browser agent functionality
on:
  slash_command:
    name: smoke-lightpanda
    strategy: centralized
    events: [issues, issue_comment, pull_request, pull_request_comment]
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["water"]
  reaction: "rocket"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke Lightpanda
model: copilot/claude-sonnet-4-5
engine:
  id: lightpanda
strict: true
imports:
  - shared/lightpanda.md
network:
  allowed:
    - defaults
    - example.com
    - www.example.com
safe-outputs:
  allowed-domains: [default-safe-outputs]
  messages:
    footer: "> 🐼 *[{workflow_name}]({run_url}) — Powered by Lightpanda*{ai_credits_suffix}{history_link}"
    run-started: "🐼 Lightpanda initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
    run-success: "🐼 [{workflow_name}]({run_url}) Lightpanda delivered."
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. Lightpanda encountered unexpected challenges..."
timeout-minutes: 15
features:
  gh-aw-detection: false
sandbox:
  agent:
    id: awf
    sudo: false
---

# Smoke Test: Lightpanda Browser Agent

**CRITICAL EFFICIENCY REQUIREMENTS:**
- Keep ALL outputs extremely short and concise. Use single-line responses.
- NO verbose explanations or unnecessary context.

Lightpanda is a headless browser agent specialized for web navigation and data
extraction. This smoke test verifies that the Lightpanda nightly binary can be
downloaded, started, and used to navigate a URL with its built-in browser primitives.

## Test Requirements

1. **Web Navigation Test**: Navigate to https://example.com and extract the page title.

Report the extracted page title as your final output (one line is sufficient).
This test is considered passing when Lightpanda exits successfully with exit code 0.
