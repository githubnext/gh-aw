---
title: How to configure a third-party agent
description: Use a third-party coding agent with GitHub Agentic Workflows by importing an engine definition file distributed by the agent's publisher.
sidebar:
  order: 330
---

Third-party coding agent CLIs that are not built into gh-aw can integrate through a declarative engine definition file that the agent publisher distributes. This guide uses [Gentek](https://github.com/gentekai/gentek) as a concrete open-source example.

## How third-party engine integration works

A third-party agent publishes a Markdown engine definition file to their GitHub repository. The file's frontmatter declares the agent's installation, configuration, and execution steps using the `engine.behaviors` format. When a workflow imports that file, gh-aw registers the engine at compile time — no changes to the gh-aw binary are required.

## Example: Gentek

Gentek publishes the following engine definition file at
`.github/workflows/gentek-engine.md` in their open-source repository:

```aw wrap title=".github/workflows/gentek-engine.md (published by the Gentek project)"
---
engine:
  id: gentek
  display-name: Gentek CLI
  description: Gentek CLI with headless mode and multi-provider LLM support
  runtime-id: crush
  experimental: true
  provider:
    name: openai
    auth:
      secret: GENTEK_API_KEY
  behaviors:
    secret-strategy: universal-llm-consumer
    capabilities:
      max-turns: true
    manifest:
      files:
        - gentek.json
        - AGENTS.md
      path-prefixes:
        - .gentek/
    installation:
      package-manager: npm
      package-name: "@gentekai/gentek"
      version: "1.0.0"
      step-name: Install Gentek CLI
      binary-name: gentek
      include-node-setup: true
      cooldown: true
      verify-command: gentek --version
      verify-step-name: Verify Gentek CLI installation
      docs-url: https://github.com/gentekai/gentek
    config-file:
      path: gentek.json
      step-name: Write Gentek Config
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
      command-name: gentek
      args:
        - run
        - --headless
      step-name: Execute Gentek CLI
      model-env-var: GENTEK_MODEL
      mcp-config-env-var: GH_AW_MCP_CONFIG
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
    mcp:
      config-path: gentek.json
---
```

## Configure a workflow to use Gentek

Import the engine definition file and set `engine: gentek` in your workflow:

```aw wrap
on: issues

engine: gentek

imports:
  - gentekai/gentek/.github/workflows/gentek-engine.md@v1.0.0

network:
  allowed:
    - defaults
    - api.openai.com

---

Triage this issue and apply an appropriate label.
```

Pin the import to a specific tag or SHA to control when you pick up new versions of the engine definition.

## Add the API key secret

Gentek reads its API key from `GENTEK_API_KEY`. Add the secret to your repository or organization:

1. Go to **Settings → Secrets and variables → Actions**.
2. Create a new secret named `GENTEK_API_KEY` with the value from your Gentek account.

## Pin the engine version

The engine definition above declares a default CLI version under `behaviors.installation.version`. Override it with `engine.version` in your workflow to pin or upgrade independently of the engine definition file:

```aw wrap
engine:
  id: gentek
  version: "1.2.0"

imports:
  - gentekai/gentek/.github/workflows/gentek-engine.md@v1.0.0
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
