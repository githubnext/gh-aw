---
description: Smoke test that validates OTLP telemetry export to all supported vendors (OTLP endpoint + JSONL mirror) with a simple bash script.
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  reaction: "eyes"
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke OTEL
engine:
  id: copilot
  max-continuations: 1
  bare: true
imports:
  - shared/observability-otlp.md
network:
  allowed:
    - defaults
    - github
tools:
  bash:
    - "*"
  edit:
safe-outputs:
    allowed-domains: [default-safe-outputs]
    add-comment:
      hide-older-comments: true
      max: 1
    create-issue:
      expires: 2h
      close-older-issues: true
      close-older-key: "smoke-otel"
      labels: [automation, testing]
    messages:
      footer: "> 📡 *[{workflow_name}]({run_url}) — OTEL smoke test*"
      run-started: "📡 OTEL smoke test initializing... [{workflow_name}]({run_url})"
      run-success: "✅ [{workflow_name}]({run_url}) — OTEL telemetry verified"
      run-failure: "⚠️ [{workflow_name}]({run_url}) — OTEL telemetry check failed"
timeout-minutes: 5

---

# Smoke Test: OTEL Telemetry Export

Verify that OTLP telemetry environment variables are correctly injected and that spans can be exported.

## Steps

1. **Verify OTLP env vars are set**: Run the following bash commands and report the results:
   ```bash
   echo "=== OTLP Environment Check ==="
   echo "OTEL_EXPORTER_OTLP_ENDPOINT: ${OTEL_EXPORTER_OTLP_ENDPOINT:+(set)}"
   echo "OTEL_EXPORTER_OTLP_HEADERS: ${OTEL_EXPORTER_OTLP_HEADERS:+(set)}"
   echo "OTEL_SERVICE_NAME: ${OTEL_SERVICE_NAME}"
   echo "GH_AW_OTLP_ENDPOINTS: ${GH_AW_OTLP_ENDPOINTS:+(set)}"
   echo "COPILOT_OTEL_FILE_EXPORTER_PATH: ${COPILOT_OTEL_FILE_EXPORTER_PATH}"
   ```

2. **Verify OTLP JSONL mirror path exists**: Run:
   ```bash
   mkdir -p /tmp/gh-aw
   ls -la /tmp/gh-aw/otel.jsonl 2>/dev/null || echo "otel.jsonl not yet created (expected before agent spans)"
   ```

3. **Check span export after setup**: Verify that setup spans were emitted by checking the JSONL mirror:
   ```bash
   if [ -f /tmp/gh-aw/otel.jsonl ]; then
     echo "=== OTLP Spans Found ==="
     SPAN_COUNT=$(wc -l < /tmp/gh-aw/otel.jsonl)
     echo "Total spans: $SPAN_COUNT"
     jq -r '.resourceSpans[]?.scopeSpans[]?.spans[]? | "\(.name) (traceId: \(.traceId[0:16])...)"' /tmp/gh-aw/otel.jsonl 2>/dev/null | head -10
   else
     echo "No OTLP spans found yet"
   fi
   if [ -f /tmp/gh-aw/otlp-export-errors.count ]; then
     OTLP_EXPORT_ERRORS="$(cat /tmp/gh-aw/otlp-export-errors.count)"
     echo "OTLP export errors: $OTLP_EXPORT_ERRORS"
     if [ "${OTLP_EXPORT_ERRORS:-0}" -gt 0 ]; then
       echo "❌ OTLP HTTP export failures detected"
       exit 1
     fi
   else
     echo "OTLP export errors: 0 (counter file not present)"
   fi
   ```

4. **Create a test file**: Write a marker file to confirm the agent ran:
   ```bash
   mkdir -p /tmp/gh-aw/agent
   echo "OTEL smoke test passed at $(date) - run ${{ github.run_id }}" > /tmp/gh-aw/agent/smoke-test-otel-${{ github.run_id }}.txt
   cat /tmp/gh-aw/agent/smoke-test-otel-${{ github.run_id }}.txt
   ```

## Output

Create an issue summarizing the results:
- Title: "Smoke Test: OTEL - ${{ github.run_id }}"
- Body: test results (✅ or ❌) for each check, including whether OTLP env vars were set, spans were found, and OTLP export errors were zero.
- Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
