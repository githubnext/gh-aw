---
private: true
emoji: "🔬"
description: Smoke Copilot SDK and Conductor multi-agent orchestration
on:
  schedule: every 2 days
  slash_command:
    name: smoke-copilot-sdk
    strategy: centralized
    events: [issues, issue_comment, pull_request, pull_request_comment]
  workflow_dispatch:
  label_command:
    name: smoke-sdk
    events: [pull_request]
  github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}
permissions:
  contents: read
  copilot-requests: write
name: Smoke Copilot SDK
model: copilot/gpt-5.4
engine:
  id: copilot
  fallback-models:
    - copilot/claude-sonnet-5
    - copilot/gpt-5.4-mini
  copilot-sdk: true
  bare: true
imports:
  - shared/smoke-test-brevity.md
  - shared/reporting.md
runtimes:
  uv: {}
steps:
  - name: Install and validate Conductor
    env:
      UV_PYTHON_INSTALL_DIR: /tmp/gh-aw/python/uv-python
      UV_PYTHON: /tmp/gh-aw/conductor/venv/bin/python
      XDG_CACHE_HOME: /tmp/gh-aw/conductor/cache
    run: |
      set -euo pipefail
      uv venv --python 3.12 --python-preference only-managed /tmp/gh-aw/conductor/venv
      uv pip install --requirements=.github/workflows/requirements-conductor.txt
      mkdir -p /tmp/gh-aw/conductor/workflows /tmp/gh-aw/conductor/runs
      cp .github/conductor/smoke-copilot-sdk.yaml /tmp/gh-aw/conductor/workflows/
      /tmp/gh-aw/conductor/venv/bin/conductor validate /tmp/gh-aw/conductor/workflows/smoke-copilot-sdk.yaml
      /tmp/gh-aw/conductor/venv/bin/python -c 'from copilot import CopilotClient; CopilotClient()'
mcp-servers:
  conductor:
    container: node:24-bookworm-slim@sha256:51b1100cc2a83d370c6a60952e3f2989c8a43159d0e38586e090f3b3326efefd
    entrypoint: /tmp/gh-aw/conductor/venv/bin/conductor
    entrypointArgs:
      - mcp
      - serve
      - --workflow-dir
      - /tmp/gh-aw/conductor/workflows
      - --allow
      - smoke-copilot-sdk
      - --launch-dir
      - /tmp/gh-aw/conductor/runs
      - --max-concurrent-runs
      - "1"
      - --max-wait-seconds
      - "30"
    mounts:
      - /tmp/gh-aw/conductor:/tmp/gh-aw/conductor:rw
      - /tmp/gh-aw/python/uv-python:/tmp/gh-aw/python/uv-python:ro
    env:
      COPILOT_GITHUB_TOKEN: ${{ github.token }}
      CONDUCTOR_NO_UPDATE_CHECK: "1"
      XDG_CACHE_HOME: /tmp/gh-aw/conductor/cache
      PATH: /tmp/gh-aw/conductor/venv/bin:/usr/local/bin:/usr/bin:/bin
    allowed:
      - smoke_copilot_sdk
      - conductor_run_status
      - conductor_await_run
      - conductor_cancel_run
max-tool-denials: 3
tools:
  bash:
    - "*"
  edit:
safe-outputs:
  create-issue:
    expires: 2h
    group: true
    close-older-issues: true
    close-older-key: "smoke-copilot-sdk"
    labels: [automation, testing]
timeout-minutes: 10
features:
  gh-aw-detection: false
sandbox:
  agent:
    id: awf
---

# Smoke Test: Copilot SDK and Conductor Validation

## Tasks

1. **File Writing**: Create a file `/tmp/smoke-copilot-sdk-${{ github.run_id }}.txt` with the content:
   ```
   Copilot SDK smoke test passed at <current date/time>
   ```
   Create the directory if it does not exist.

2. **Verify**: Read the file back with `cat` and confirm it contains the phrase "smoke test passed".

3. **Bash calculation**: Run a bash command to compute `echo $((6 * 7))` and confirm the output is `42`.

4. **Conductor orchestration**: Invoke the Conductor MCP tool `smoke_copilot_sdk`
   exactly once with `marker` set to `GH_AW_CONDUCTOR_SMOKE_OK`. This runs the
   checked-in two-agent workflow in `.github/conductor/smoke-copilot-sdk.yaml`
   using [microsoft/conductor](https://github.com/microsoft/conductor).
   Conductor is already installed and validated; do not install it, rewrite
   the workflow, or substitute a shell command or task agent.
   If the tool returns a run handle, wait with `conductor_await_run` until it
   finishes. PASS requires a completed run with output `status` equal to `PASS`,
   `marker` equal to `GH_AW_CONDUCTOR_SMOKE_OK`, and numeric `answer` equal to `42`.
   Report the actual output and run ID when available. Missing tools, failed or
   timed-out runs, or mismatched output mean FAIL; never fabricate a result.
   Cancel an unfinished run with `conductor_cancel_run` before reporting FAIL.

## Output

Create an issue titled **"Smoke Test: Copilot SDK - ${{ github.run_id }}"** with:
- ✅ or ❌ for each task above
- Overall status: PASS or FAIL
- Conductor output or actual error (do not include credentials)
- Run URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}