---
# Pydantic AI Engine Setup
# Installs pydantic-ai[cli] via uv so workflows can invoke it as a custom engine.
#
# Usage:
#   imports:
#     - shared/pydantic-engine.md
#
# Then configure the engine in the importing workflow:
#   engine:
#     command: pydantic-ai
#     args: [run]
#     env:
#       PYDANTIC_AI_MODEL: "openai:gpt-4o"
#       OPENAI_API_KEY: "${{ secrets.COPILOT_GITHUB_TOKEN }}"
#       OPENAI_BASE_URL: "${{ env.GITHUB_COPILOT_BASE_URL }}"
#
# Supported model providers (set PYDANTIC_AI_MODEL):
#   - Copilot routing:   openai:gpt-4o  +  OPENAI_API_KEY=${{ secrets.COPILOT_GITHUB_TOKEN }}
#                        OPENAI_BASE_URL=${{ env.GITHUB_COPILOT_BASE_URL }}
#   - Anthropic:         anthropic:claude-sonnet-4-5  +  ANTHROPIC_API_KEY=${{ secrets.ANTHROPIC_API_KEY }}
#   - OpenAI:            openai:gpt-4o  +  OPENAI_API_KEY=${{ secrets.OPENAI_API_KEY }}

network:
  allowed:
    - python

steps:
  - name: Setup uv
    uses: astral-sh/setup-uv@v8.2.0
  - name: Install Pydantic AI
    run: uv tool install 'pydantic-ai[cli]==0.0.67'
  - name: Verify Pydantic AI installation
    run: pydantic-ai --version
---

<!--
# Pydantic AI Engine

Shared setup for [Pydantic AI](https://ai.pydantic.dev/) multi-provider coding agent.

Installs `pydantic-ai[cli]` via `uv tool install` and adds the tool binary to `PATH`.
Import this shared component to make `pydantic-ai run` available in any agentic workflow.

## Auth

Pydantic AI uses standard provider env vars. For GitHub Copilot routing:

```yaml
engine:
  command: pydantic-ai
  args: [run]
  env:
    PYDANTIC_AI_MODEL: "openai:gpt-4o"
    OPENAI_API_KEY: "${{ secrets.COPILOT_GITHUB_TOKEN }}"
    OPENAI_BASE_URL: "${{ env.GITHUB_COPILOT_BASE_URL }}"
```

For Anthropic models:

```yaml
engine:
  command: pydantic-ai
  args: [run]
  env:
    PYDANTIC_AI_MODEL: "anthropic:claude-sonnet-4-5"
    ANTHROPIC_API_KEY: "${{ secrets.ANTHROPIC_API_KEY }}"
```
-->
