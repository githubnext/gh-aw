---
title: How to configure a third-party agent
description: Use a third-party coding agent with GitHub Agentic Workflows by importing an engine definition file distributed by the agent's publisher.
sidebar:
  order: 330
---

Third-party coding agent CLIs that are not built into gh-aw can integrate through a declarative engine definition file that the agent publisher distributes. This guide uses [OpenCode](https://opencode.ai) as a concrete open-source example.

## How third-party engine integration works

A third-party agent publishes a Markdown engine definition file to their GitHub repository. The file's frontmatter declares the agent's installation, configuration, and execution steps using the `engine.behaviors` format. When a workflow imports that file, gh-aw registers the engine at compile time — no changes to the gh-aw binary are required.

## Example: OpenCode

OpenCode is an open-source, provider-agnostic AI coding agent (BYOK — Bring Your Own Key) that supports 75+ models from Anthropic, OpenAI, Google, Groq, and others via a unified CLI interface.

An agent publisher provides an engine definition file like the following in their repository. The file's `engine.behaviors` block tells gh-aw exactly how to install, configure, and invoke the CLI:

```aw wrap title=".github/workflows/opencode-engine.md (published by the OpenCode project)"
---
engine:
  id: opencode
  display-name: OpenCode
  description: OpenCode CLI with headless mode and multi-provider LLM support
  runtime-id: opencode
  experimental: true
  behaviors:
    secret-strategy: universal-llm-consumer
    capabilities:
      max-turns: true
    manifest:
      files:
        - opencode.jsonc
        - AGENTS.md
      path-prefixes:
        - .opencode/
    installation:
      package-manager: npm
      package-name: opencode-ai
      version: "1.2.14"
      step-name: Install OpenCode
      binary-name: opencode
      include-node-setup: true
      cooldown: true
      verify-command: opencode --version
      verify-step-name: Verify OpenCode CLI installation
      docs-url: https://opencode.ai/docs
    config-file:
      path: opencode.jsonc
      step-name: Write OpenCode Config
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
          "autoupdate": false
        }
      merge-strategy: json-merge
    execution:
      command-name: opencode
      args:
        - run
        - --print-logs
        - --log-level
        - DEBUG
      step-name: Execute OpenCode CLI
      model-env-var: OPENCODE_MODEL
      mcp-config-env-var: GH_AW_MCP_CONFIG
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
    mcp:
      config-path: opencode.jsonc
---
```

## Configure a workflow to use OpenCode

Import the engine definition file and set `engine: opencode` in your workflow:

```aw wrap
on: issues

engine: opencode

imports:
  - sst/opencode/.github/workflows/opencode-engine.md@v1.2.14

network:
  allowed:
    - defaults
    - api.anthropic.com

---

Triage this issue and apply an appropriate label.
```

Pin the import to a specific tag or SHA to control when you pick up new versions of the engine definition.

The `network.allowed` entry should match the provider you are using. OpenCode supports multiple providers — for example, add `api.openai.com` instead of (or in addition to) `api.anthropic.com` when using an OpenAI model.

## Add the API key secret

OpenCode reads provider credentials from environment variables. For the default Anthropic provider, add `ANTHROPIC_API_KEY` to your repository or organization:

1. Go to **Settings → Secrets and variables → Actions**.
2. Create a new secret named `ANTHROPIC_API_KEY` with the value from your Anthropic account.

For other providers, set the corresponding key (for example `OPENAI_API_KEY` for OpenAI models) and reference it in your workflow's `engine.env` block.

## Pin the engine version

The engine definition above declares a default CLI version under `behaviors.installation.version`. Override it with `engine.version` in your workflow to pin or upgrade independently of the engine definition file:

```aw wrap
engine:
  id: opencode
  version: "1.3.0"

imports:
  - sst/opencode/.github/workflows/opencode-engine.md@v1.2.14
```

## Recompile after workflow edits

Engine settings live in workflow frontmatter. Recompile whenever you change the import reference, the engine version, or any other frontmatter field:

```bash
gh aw compile .github/workflows/my-workflow.md --watch
```

## Related documentation

- [AI Engines Reference](/gh-aw/reference/engines/) — built-in engine options and configuration
- [Imports Reference](/gh-aw/reference/imports/) — how imports and frontmatter merging work
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) — configuring outbound network access
