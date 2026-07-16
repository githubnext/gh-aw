---
engine:
  id: auggie
  display-name: Auggie CLI
  description: Augment Code Auggie CLI (experimental)
  runtime-id: auggie
  experimental: true
  provider:
    name: augmentcode
  auth:
    - role: session
      secret: AUGMENT_SESSION_AUTH
  behaviors:
    supported-env-var-keys:
      - AUGMENT_SESSION_AUTH
    capabilities:
      web-search: true
    manifest:
      files:
        - AGENTS.md
      path-prefixes:
        - .augment/
    installation:
      package-manager: npm
      package-name: "@augmentcode/auggie"
      version: "0.29.0"
      step-name: Install Auggie CLI
      binary-name: auggie
      include-node-setup: true
      verify-command: auggie --version
      verify-step-name: Verify Auggie CLI installation
      docs-url: https://docs.augmentcode.com/cli/overview
    execution:
      command-name: auggie
      args:
        - --print
        - --quiet
      step-name: Execute Auggie CLI
      model-env-var: GH_AW_MODEL_AGENT_AUGGIE
      detection-model-env-var: GH_AW_MODEL_DETECTION_AUGGIE
      model-flag: --model
      mcp-config-flag: --mcp-config
      write-timestamp: true
    mcp:
      config-path: "${{ runner.temp }}/gh-aw/mcp-config/mcp-servers.json"
---

<!-- # Auggie CLI

Shared engine configuration for Augment Code Auggie CLI. -->
