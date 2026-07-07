---
private: true
emoji: "🦙"
description: Daily test of the Copilot BYOK endpoint using a local Ollama instance with a small model
on:
  schedule: daily on weekdays
max-daily-ai-credits: 10000
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
    env:
      OLLAMA_VERSION: "0.31.1"
      # SHA256 of install.sh from https://github.com/ollama/ollama/releases/download/v${OLLAMA_VERSION}/install.sh
      # To update: curl -fsSL https://github.com/ollama/ollama/releases/download/vNEW_VERSION/install.sh | sha256sum
      OLLAMA_INSTALL_SHA256: "25f64b810b947145095956533e1bdf56eacea2673c55a7e586be4515fc882c9f"
    run: |
      echo "Downloading Ollama v${OLLAMA_VERSION} install script..."
      mkdir -p /tmp/gh-aw
      curl -fsSL "https://github.com/ollama/ollama/releases/download/v${OLLAMA_VERSION}/install.sh" -o /tmp/gh-aw/ollama-install.sh
      echo "${OLLAMA_INSTALL_SHA256}  /tmp/gh-aw/ollama-install.sh" | sha256sum -c -
      bash /tmp/gh-aw/ollama-install.sh
  - name: Generate Ollama API key
    run: |
      OLLAMA_API_KEY="$(openssl rand -hex 16)"
      echo "OLLAMA_API_KEY=$OLLAMA_API_KEY" >> "$GITHUB_ENV"
  - name: Start Ollama service
    env:
      OLLAMA_HOST: "0.0.0.0:11434"
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
  - name: Verify Ollama BYOK readiness
    env:
      OLLAMA_MODEL: "qwen2.5:0.5b"
    run: |
      echo "Checking Ollama model availability..."
      if ! ollama list | grep -Fq "$OLLAMA_MODEL"; then
        echo "::error::Required model '$OLLAMA_MODEL' is not available in Ollama."
        exit 1
      fi

      echo "Waiting for Ollama OpenAI-compatible endpoint..."
      MAX_WAIT_SECONDS=30
      for i in $(seq 1 "$MAX_WAIT_SECONDS"); do
        if curl -sf http://localhost:11434/v1/models > /dev/null 2>&1; then
          echo "Ollama /v1/models is ready"
          exit 0
        fi
        sleep 1
      done

      echo "::error::Ollama /v1/models did not become ready in ${MAX_WAIT_SECONDS}s."
      exit 1
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
    footer: "> 🦙 *BYOK test via [{workflow_name}]({run_url})*{ai_credits_suffix}"
    run-started: "🦙 BYOK Ollama test starting... [{workflow_name}]({run_url})"
    run-success: "✅ [{workflow_name}]({run_url}) — BYOK endpoint responded."
    run-failure: "❌ [{workflow_name}]({run_url}) — BYOK endpoint test failed: {status}"
features:
  gh-aw-detection: true
sandbox:
  agent:
    sudo: false
---

### Daily BYOK Endpoint Test

**Report Formatting**: Use h3 (###) or lower for all headers in your report
to maintain proper document hierarchy. Wrap long sections in
`<details><summary>View Full Details</summary>` tags to improve readability.


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