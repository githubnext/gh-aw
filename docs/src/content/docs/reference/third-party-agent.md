---
title: Custom AI Engines
description: Import declarative engine definitions and configure third-party coding agents.
sidebar:
  order: 330
---

Third-party coding agents integrate through a Markdown engine definition, maintained by the engine's publisher. Its `engine.behaviors` frontmatter declares installation, configuration, execution, and log parsing. Importing the definition registers the engine at compile time without changing the gh-aw binary.

For built-in engines and their feature matrix, see [AI engines](/gh-aw/reference/engines/).

## Shared imported engines

Custom engines require object form with `engine.id`. String form accepts only built-in IDs; there is no `engine: custom`.

```yaml wrap
engine:
  id: example-engine
imports:
  - owner/repository/.github/workflows/example-engine.md@v1.2.3
```

The ID must match the imported definition. Prefer a publisher-maintained definition pinned to a reviewed commit SHA or tag. Local definitions can be vendored under `.github/workflows/shared/` and imported by path.

An imported definition is executable configuration: review its installation commands, permissions, network access, secret handling, tool restrictions, and parser. Compatibility and support belong to its publisher; gh-aw does not guarantee native-agent features or restrictions merely because a definition compiles.

Agent Plugins are available only when the definition declares `behaviors.plugins`; see [plugin configuration](/gh-aw/reference/frontmatter/#agent-plugins-plugins).

## Authentication and inference

Credentials, model prefixes, and provider routing depend on the definition. Configure the secrets and network destinations it declares. A custom engine can support multiple inference providers, but that does not imply compatibility with every provider API.

`safe-outputs.threat-detection: true` uses Copilot by default, even when the main agent is custom. Supply Copilot authentication or choose a supported detection engine explicitly; see [Threat detection](/gh-aw/reference/threat-detection/).

## Unsupported engine samples

These definitions are examples for integration authors, **not supported gh-aw engines**. They have no compatibility or maintenance commitment. Production users should obtain a definition maintained by the engine owner.

| Engine | Sample definition |
| --- | --- |
| [OpenCode](https://opencode.ai) | [opencode.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/opencode.md) |
| [Aider](https://aider.chat/docs/) | [aider.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/aider.md) |
| [Crush](https://github.com/charmbracelet/crush) | [crush.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/crush.md) |
| [Cursor](https://cursor.com/docs/cli) | [cursor.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/cursor.md) |
| [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) | [deepseek-harness.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/deepseek-harness.md) |
| [Goose](https://github.com/aaif-goose/goose) | [goose.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/goose.md) |
| [Kiro](https://kiro.dev/) | [kiro.md](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/kiro.md) |
| [Pydantic AI](https://ai.pydantic.dev/) | [Publisher definition](https://github.com/pydantic/pydantic-ai/blob/main/src/pydantic_ai_harness/gh-aw/pydantic.md) |

### Sample-specific constraints

| Sample | Runtime and limitations |
| --- | --- |
| OpenCode | Installs OpenCode 2 from `@opencode/cli`, uses `--standalone`, native MCP with Code Mode disabled, and JSONL events. Copilot requires AWF; direct Anthropic/OpenAI BYOK is supported without it. Repository JSON/JSONC settings are overlaid; updates/sharing are disabled. V2 has no language servers, and V1 plugins require porting. Native npm plugins are not gh-aw Agent Plugins. |
| DeepSeek Harness | Pins `dsh` v0.2.0-rc.2. Uses headless stdin prompts, a private per-run configuration patch, and CLI-proxy tools rather than native MCP. Copilot requires AWF; direct Anthropic/OpenAI credentials apply only without AWF. Per-command native bash restrictions are rejected. No native reasoning, tool, refusal, or token-accounting observations are inferred from stdout. |
| Goose | Pins CLI v1.53.0 with archive checksum verification. Uses OpenAI-compatible inference, native authenticated MCP, file-based prompts, and JSONL tool/usage events. Configuration/session state is temporary; keyring is disabled. No native gh-aw `web-fetch`; use MCP or shell. Copilot can use `copilot-requests: write` without a PAT. |
| Kiro | Sample pins CLI 2.31.1 with the V2 harness; Powers plugins are not supported. |

Read the pinned definition for its exact install and provider contract. Native events and accounting are retained only when observed; process exit zero alone does not prove task completion.

## Defining an engine

A definition contains `engine.id`, metadata, and `engine.behaviors`. Its behaviors install the CLI, configure inference and MCP, invoke it headlessly, and expose logs for session reconstruction. Keep secrets outside the agent sandbox and fail explicitly when required credentials, endpoints, or restrictions cannot be honored.

Start from a sample with a matching transport rather than copying installation or configuration code into every workflow. Publish the definition with versioned changes and document its credentials, models, tool restrictions, supported platforms, and [unified session](/gh-aw/specs/unified-agent-session-specification/) coverage.

See [imports](/gh-aw/reference/imports/) for resolution and pinning rules.
