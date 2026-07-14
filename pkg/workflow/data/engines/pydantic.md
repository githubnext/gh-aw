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

Engine auth and execution configuration for Pydantic AI multi-provider coding agent (BYOK).

Installation is handled by importing shared/pydantic-engine.md into your workflow:

  imports:
    - shared/pydantic-engine.md

This file provides only the auth/execution behaviors. The shared workflow provides
the uv/pydantic-ai installation steps. -->
