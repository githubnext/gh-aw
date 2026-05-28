---
emoji: "🦙"
description: Daily test of the Copilot BYOK endpoint using a local Ollama instance with a small model
on:
  schedule: daily on weekdays
permissions:
  contents: read
  issues: read
name: Daily BYOK Ollama Test
engine:
  id: copilot
  bare: true
  env:
    COPILOT_PROVIDER_BASE_URL: "http://host.docker.internal:11434/v1"
    COPILOT_PROVIDER_API_KEY: "${{ env.OLLAMA_API_KEY }}"
    COPILOT_MODEL: "qwen2.5:0.5b"
strict: true
timeout-minutes: 20
steps:
  - name: Install Ollama
    run: |
      curl -fsSL https://ollama.com/install.sh | sh
  - name: Generate Ollama API key
    run: |
      OLLAMA_API_KEY="$(openssl rand -hex 16)"
      echo "OLLAMA_API_KEY=$OLLAMA_API_KEY" >> "$GITHUB_ENV"
  - name: Start Ollama service
    run: |
      ollama serve &
      echo "Waiting for Ollama service..."
      for i in $(seq 1 30); do
        if curl -sf http://localhost:11434/api/version > /dev/null 2>&1; then
          echo "Ollama is ready"
          break
        fi
        sleep 1
      done
  - name: Pull small model
    run: |
      ollama pull qwen2.5:0.5b
network:
  allowed:
    - defaults
    - host.docker.internal
safe-outputs:
  create-issue:
    expires: 24h
    close-older-issues: true
    close-older-key: "daily-byok-ollama-test"
    labels: [automation, testing]
  messages:
    footer: "> 🦙 *BYOK test via [{workflow_name}]({run_url})*{effective_tokens_suffix}"
    run-started: "🦙 BYOK Ollama test starting... [{workflow_name}]({run_url})"
    run-success: "✅ [{workflow_name}]({run_url}) — BYOK endpoint responded."
    run-failure: "❌ [{workflow_name}]({run_url}) — BYOK endpoint test failed: {status}"
---

# Daily BYOK Endpoint Test

You are a BYOK connectivity test. Your only task is to compose a haiku and report the result.

Write a haiku (5-7-5 syllable pattern) about code, automation, or workflows.

Then create an issue with:
- Title: `BYOK Ollama Test — ${{ github.run_id }}`
- Body:
  ```
  ## 🦙 Daily BYOK Ollama Test

  **Status:** ✅ PASS — Ollama responded via BYOK
  **Model:** qwen2.5:0.5b
  **Run:** ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}

  ### Haiku

  <your haiku here>
  ```

{{#runtime-import shared/noop-reminder.md}}
