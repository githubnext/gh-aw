---
engine:
  id: cursor
  display-name: Cursor
  description: Cursor CLI with headless mode and multi-provider LLM support
  runtime-id: cursor
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
      package-manager: npm
      package-name: "@cursor/agent"
      version: "0.1.0"
      step-name: Install Cursor
      binary-name: cursor-agent
      include-node-setup: true
      cooldown: true
      verify-command: cursor-agent --version
      verify-step-name: Verify Cursor CLI installation
      docs-url: https://cursor.com/docs/cli/headless
    config-file:
      path: cursor-agent.jsonc
      step-name: Write Cursor Config
      content: |-
        {
          "agent": {
            "build": {
              "permission": {
                "bash": "allow",
                "edit": "allow",
                "read": "allow",
                "glob": "allow",
                "grep": "allow",
                "webfetch": "allow",
                "websearch": "allow",
                "external_directory": "allow"
              }
            }
          },
          "autoupdate": false,
          "disabled_providers": ["openai"],
          "provider": {
            "awf-proxy": {
              "api": "http://172.30.0.30:10002",
              "options": {
                "apiKey": "awf-copilot-proxy"
              },
              "models": {
                "claude-sonnet-4.5": {}
              }
            }
          }
        }
      merge-strategy: json-merge
    execution:
      command-name: cursor-agent
      args:
        - run
        - --print-logs
        - --log-level
        - DEBUG
      step-name: Execute Cursor CLI
      model-env-var: CURSOR_MODEL
      model-env-provider-prefix: awf-proxy
      mcp-config-env-var: GH_AW_MCP_CONFIG
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
      env:
        XDG_DATA_HOME: /tmp/cursor-agent-data
    mcp:
      config-path: cursor-agent.jsonc
---

<!-- # Cursor CLI

Shared engine configuration for Cursor multi-provider AI coding agent (BYOK). -->
