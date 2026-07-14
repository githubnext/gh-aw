---
engine:
  id: pydantic
  display-name: Pydantic AI
  description: Pydantic AI CLI with multi-provider LLM support via universal gateway
  runtime-id: pydantic
  experimental: true
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    capabilities:
      max-turns: true
    manifest:
      files:
        - AGENTS.md
    installation:
      package-manager: uv
      package-name: pydantic-ai[cli]
      version: "0.0.67"
      step-name: Install Pydantic AI
      binary-name: pydantic-ai
      verify-command: pydantic-ai --version
      verify-step-name: Verify Pydantic AI installation
      docs-url: https://ai.pydantic.dev/install/
    execution:
      command-name: pydantic-ai
      args:
        - run
      step-name: Execute Pydantic AI
      model-env-var: PYDANTIC_AI_MODEL
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
---

<!-- # Pydantic AI

Shared engine configuration for Pydantic AI multi-provider coding agent (BYOK). -->
