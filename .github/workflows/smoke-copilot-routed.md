---
private: true
emoji: "🧭"
description: Smoke Copilot CLI model routing with deterministic runner and proxy evidence checks
on:
  schedule: every 2 days
  workflow_dispatch:
  push:
    branches: [main]
    paths:
      - ".github/workflows/smoke-*-routed*.md"
      - ".github/workflows/smoke-*-routed*.lock.yml"
      - ".github/workflows/shared/reporting.md"
      - ".github/workflows/shared/smoke-test-brevity.md"
      - "pkg/workflow/model_routing.go"
      - "pkg/workflow/awf_config.go"
      - "pkg/workflow/awf_config_build.go"
      - "pkg/workflow/compiler_orchestrator_engine.go"
      - "pkg/workflow/copilot_engine_execution.go"
      - "pkg/workflow/engine.go"
      - "pkg/workflow/engine_fallback_models.go"
      - "pkg/workflow/engine_validation.go"
      - "pkg/workflow/pi_engine.go"
      - "pkg/workflow/pi_engine_config.go"
      - "pkg/workflow/sandbox_agent_images.go"
      - "pkg/workflow/threat_detection_helpers.go"
      - "pkg/workflow/safe_outputs_env.go"
      - "pkg/workflow/compiler_yaml_artifacts.go"
      - "pkg/workflow/notify_comment.go"
      - "pkg/workflow/evals_steps.go"
      - "pkg/workflow/data/engines/*.md"
      - "pkg/workflow/data/model_aliases.json"
      - "pkg/cli/data/models.json"
      - "pkg/constants/version_constants.go"
      - "pkg/constants/constants.go"
      - "pkg/workflow/data/action_pins.json"
      - "pkg/actionpins/data/action_pins.json"
      - ".github/aw/actions-lock.json"
      - "actions/setup/js/copilot_harness.cjs"
      - "actions/setup/js/claude_harness.cjs"
      - "actions/setup/js/codex_harness.cjs"
      - "actions/setup/js/harness_*.cjs"
      - "actions/setup/js/awf_model_routing.cjs"
      - "actions/setup/js/copilot_routing_effort.cjs"
      - "actions/setup/js/copilot_wire_api.cjs"
      - "actions/setup/js/copilot_sdk_*.cjs"
      - "actions/setup/js/pi_*.cjs"
      - "actions/setup/js/model_aliases.cjs"
      - "actions/setup/js/resolve_model_alias.cjs"
      - "actions/setup/js/model_attribution.cjs"
      - "actions/setup/js/model_costs.cjs"
      - "actions/setup/js/model_fallback.cjs"
      - "actions/setup/js/steering_helpers.cjs"
      - "actions/setup/js/generate_aw_info.cjs"
      - "actions/setup/sh/install_copilot_cli.sh"
      - "actions/setup/sh/install_awf_binary.sh"
      - "actions/setup/js/smoke_model_routing_assertions.cjs"
      - "actions/setup/js/unified_session*.cjs"
      - "actions/setup/js/copilot_session.cjs"
      - "actions/setup/js/claude_session.cjs"
      - "actions/setup/js/codex_session.cjs"
      - "actions/setup/js/agent_session.cjs"
      - "actions/setup/js/session_artifact.cjs"
      - "actions/setup/js/parse_token_usage.cjs"
      - "actions/setup/js/awf_reflect*.cjs"
      - "!**/*.test.cjs"
      - "!**/*_test.go"
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
      - ".github/workflows/smoke-*-routed*.md"
      - ".github/workflows/smoke-*-routed*.lock.yml"
      - ".github/workflows/shared/reporting.md"
      - ".github/workflows/shared/smoke-test-brevity.md"
      - "pkg/workflow/model_routing.go"
      - "pkg/workflow/awf_config.go"
      - "pkg/workflow/awf_config_build.go"
      - "pkg/workflow/compiler_orchestrator_engine.go"
      - "pkg/workflow/copilot_engine_execution.go"
      - "pkg/workflow/engine.go"
      - "pkg/workflow/engine_fallback_models.go"
      - "pkg/workflow/engine_validation.go"
      - "pkg/workflow/pi_engine.go"
      - "pkg/workflow/pi_engine_config.go"
      - "pkg/workflow/sandbox_agent_images.go"
      - "pkg/workflow/threat_detection_helpers.go"
      - "pkg/workflow/safe_outputs_env.go"
      - "pkg/workflow/compiler_yaml_artifacts.go"
      - "pkg/workflow/notify_comment.go"
      - "pkg/workflow/evals_steps.go"
      - "pkg/workflow/data/engines/*.md"
      - "pkg/workflow/data/model_aliases.json"
      - "pkg/cli/data/models.json"
      - "pkg/constants/version_constants.go"
      - "pkg/constants/constants.go"
      - "pkg/workflow/data/action_pins.json"
      - "pkg/actionpins/data/action_pins.json"
      - ".github/aw/actions-lock.json"
      - "actions/setup/js/copilot_harness.cjs"
      - "actions/setup/js/claude_harness.cjs"
      - "actions/setup/js/codex_harness.cjs"
      - "actions/setup/js/harness_*.cjs"
      - "actions/setup/js/awf_model_routing.cjs"
      - "actions/setup/js/copilot_routing_effort.cjs"
      - "actions/setup/js/copilot_wire_api.cjs"
      - "actions/setup/js/copilot_sdk_*.cjs"
      - "actions/setup/js/pi_*.cjs"
      - "actions/setup/js/model_aliases.cjs"
      - "actions/setup/js/resolve_model_alias.cjs"
      - "actions/setup/js/model_attribution.cjs"
      - "actions/setup/js/model_costs.cjs"
      - "actions/setup/js/model_fallback.cjs"
      - "actions/setup/js/steering_helpers.cjs"
      - "actions/setup/js/generate_aw_info.cjs"
      - "actions/setup/sh/install_copilot_cli.sh"
      - "actions/setup/sh/install_awf_binary.sh"
      - "actions/setup/js/smoke_model_routing_assertions.cjs"
      - "actions/setup/js/unified_session*.cjs"
      - "actions/setup/js/copilot_session.cjs"
      - "actions/setup/js/claude_session.cjs"
      - "actions/setup/js/codex_session.cjs"
      - "actions/setup/js/agent_session.cjs"
      - "actions/setup/js/session_artifact.cjs"
      - "actions/setup/js/parse_token_usage.cjs"
      - "actions/setup/js/awf_reflect*.cjs"
      - "!**/*.test.cjs"
      - "!**/*_test.go"
  label_command:
    name: smoke-routing
    events: [pull_request]
    strategy: decentralized
  bots: [Copilot]
  status-comment: false
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
permissions:
  contents: read
  copilot-requests: write
name: Smoke Copilot Routed
concurrency:
  group: "smoke-copilot-routed-${{ github.event_name == 'pull_request' && format('pr-{0}', github.event.pull_request.number) || github.run_id }}"
  cancel-in-progress: true
  job-discriminator: ${{ github.run_id }}
strict: true
engine:
  id: copilot
  bare: true
  model-routing:
    goal: cost
    mode: economy
    allowed-models: [gpt-5.4-mini, gpt-5.6-luna, claude-haiku-4.5]
imports:
  - shared/smoke-test-brevity.md
  - shared/reporting.md
tools:
  bash:
    - "echo"
safe-outputs:
  create-issue:
    expires: 2h
    group: true
    close-older-issues: true
    close-older-key: "smoke-copilot-routed"
    labels: [automation, testing]
  threat-detection:
    model: claude-haiku-5.5
    post-steps:
      - name: Assert threat-detection evidence
        if: always()
        uses: actions/github-script@3a2844b7e9c422d3c10d287c895573f7108da1b3 # v9.0.0
        env:
          SMOKE_DETECTION_EXECUTION: ${{ steps.detection_agentic_execution.outcome }}
        with:
          script: |
            const path = require("node:path");
            const { detectionMain } = require(path.join(process.env.RUNNER_TEMP, "gh-aw/actions/smoke_model_routing_assertions.cjs"));
            await detectionMain({
              core,
              executionOutcome: process.env.SMOKE_DETECTION_EXECUTION,
              detectionModel: "claude-haiku-5.5",
              allowedModels: ["gpt-5.4-mini", "gpt-5.6-luna", "claude-haiku-4.5"],
            });
timeout-minutes: 10
sandbox:
  agent:
    id: awf
post-steps:
  - name: Assert model-routing evidence
    if: always()
    uses: actions/github-script@v9.0.0
    env:
      SMOKE_EXECUTION: ${{ steps.agentic_execution.outcome }}
    with:
      script: |
        const path = require("node:path");
        const { main } = require(path.join(process.env.RUNNER_TEMP, "gh-aw/actions/smoke_model_routing_assertions.cjs"));
        await main({
          core,
          engine: "copilot",
          executionOutcome: process.env.SMOKE_EXECUTION,
          allowedModels: ["gpt-5.4-mini", "gpt-5.6-luna", "claude-haiku-4.5"],
        });
---

# Smoke Test: Copilot CLI Model Routing

This run checks model routing, not the task. Keep it trivial and short.

## Task

Run `echo 42` once and confirm the output is `42`. Do not make any other tool calls except the issue below.

## Output

Create an issue titled **"Smoke Test: Copilot Routed - ${{ github.run_id }}"** with:
- ✅ or ❌ for the task above
- Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

The routing verdict comes from the post-step, which checks runner- and proxy-written evidence, not this report.
