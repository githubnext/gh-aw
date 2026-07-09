---
engine:
  id: crush
  display-name: Crush
  description: Crush CLI with headless mode and multi-provider LLM support
  runtime-id: crush
  experimental: true
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    capabilities:
      max-turns: true
    manifest:
      files:
        - .crush.json
        - AGENTS.md
      path-prefixes:
        - .crush/
    installation:
      package-manager: npm
      package-name: "@charmland/crush"
      version: "0.59.0"
      step-name: Install Crush
      binary-name: crush
      include-node-setup: true
      post-install-scripts: true
      cooldown: true
      verify-command: crush --version
      verify-step-name: Verify Crush CLI installation
      docs-url: https://github.com/charmbracelet/crush
    config-file:
      path: .crush.json
      step-name: Write Crush Config
      content: |-
        {
          "permission": {
            "edit": "allow",
            "bash": "allow",
            "external_directory": "allow"
          }
        }
      merge-strategy: json-merge
    execution:
      command-name: crush
      args:
        - run
        - --verbose
      step-name: Execute Crush CLI
      model-env-var: CRUSH_MODEL
      mcp-config-env-var: GH_AW_MCP_CONFIG
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
    mcp:
      config-path: .crush.json
---

<!-- # Crush CLI

Shared engine configuration for Crush multi-provider AI coding agent (BYOK). -->
