---
private: true
emoji: "📊"
name: Smoke Token Telemetry
description: >-
  Smoke CI assertion: validates that the AWF firewall proxy token-usage emitter
  populates token_usage.jsonl for copilot agent LLM runs. Fails fast when
  token telemetry is broken, so regressions are caught in hours not weeks.
on:
  push:
    branches: [main]
    paths:
      - 'actions/setup/js/**'
      - 'cmd/**'
      - 'pkg/**'
      - '*.go'
      - 'go.mod'
  schedule: daily
concurrency:
  group: smoke-token-telemetry-${{ github.ref }}
  cancel-in-progress: true
permissions:
  contents: read
engine:
  id: copilot
strict: true
timeout-minutes: 5
network:
  allowed:
    - defaults
imports:
  - shared/otlp.md
safe-outputs:
  allowed-domains: [default-safe-outputs]
  noop:
features:
  gh-aw-detection: false
jobs:
  check_token_telemetry:
    needs: [agent]
    if: needs.agent.result == 'success'
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - name: Download agent artifact
        id: download-agent
        continue-on-error: true
        uses: actions/download-artifact@v4
        with:
          name: agent
          path: /tmp/gh-aw/
      - name: Assert token_usage.jsonl is non-empty
        if: steps.download-agent.outcome == 'success'
        run: |
          # The AWF firewall proxy writes token_usage.jsonl for every LLM API call.
          # If all token_usage.jsonl files are missing or empty, the emitter is broken.
          TOKEN_FILES=(
            "/tmp/gh-aw/sandbox/firewall-audit-logs/api-proxy-logs/token-usage.jsonl"
            "/tmp/gh-aw/sandbox/firewall/audit/api-proxy-logs/token-usage.jsonl"
            "/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl"
          )

          FOUND_NONEMPTY=false
          for f in "${TOKEN_FILES[@]}"; do
            if [ -s "$f" ]; then
              COUNT=$(grep -c . "$f" 2>/dev/null || echo "0")
              echo "OK: $f — ${COUNT} record(s)"
              FOUND_NONEMPTY=true
            else
              [ -f "$f" ] && echo "EMPTY: $f" || echo "MISSING: $f"
            fi
          done

          if [ "${FOUND_NONEMPTY}" != "true" ]; then
            echo "::error::All token_usage.jsonl files are empty or missing after a successful agent run."
            echo "::error::The AWF firewall proxy token telemetry emitter may be broken."
            echo "::error::See tracking issue: https://github.com/github/gh-aw/issues/42791"
            exit 1
          fi
      - name: Assert agent_usage.json has non-zero token counts
        if: steps.download-agent.outcome == 'success'
        run: |
          USAGE_FILE="/tmp/gh-aw/agent_usage.json"
          if [ ! -f "${USAGE_FILE}" ]; then
            echo "::error::agent_usage.json not found in agent artifact — token summary was not written."
            exit 1
          fi
          INPUT_TOKENS=$(python3 -c "import json; d=json.load(open('${USAGE_FILE}')); print(d.get('input_tokens', 0))")
          if [ "${INPUT_TOKENS}" -le 0 ]; then
            echo "::error::agent_usage.json has zero input_tokens — token telemetry may be broken."
            cat "${USAGE_FILE}"
            exit 1
          fi
          echo "OK: agent_usage.json reports ${INPUT_TOKENS} input tokens"
---

Say the single word "ok" and call noop with a message confirming this run completed successfully.
