# ADR-25830: Add OpenCode as a Provider-Agnostic BYOK Agentic Engine

**Date**: 2026-04-11
**Status**: Superseded by [PR #26819](https://github.com/github/gh-aw/pull/26819) — OpenCode was archived; the engine was replaced by Crush (`engine: id: crush`) on 2026-04-17.
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

gh-aw supports several first-party agentic engines (Copilot, Claude, Codex, Gemini) that each bind to a single AI provider and require a corresponding vendor API key. Users who want to run models from multiple providers — or who prefer open-source tooling — have no path today without writing a fully custom engine. OpenCode is a provider-agnostic, open-source AI coding agent (BYOK — Bring Your Own Key) that supports 75+ models via a unified CLI interface using a `provider/model` format (e.g., `anthropic/claude-sonnet-4-20250514`). Because each provider's API endpoint is different, adding OpenCode also introduces a new challenge: the network firewall allowlist cannot be a static list and must be computed dynamically from the selected model provider at compile time.

### Decision

We will integrate OpenCode as a fifth built-in agentic engine (`id: "opencode"`) following the existing `BaseEngine` pattern used by Claude, Codex, and Gemini. The engine is installed from npm (`opencode-ai@1.2.14`), runs in headless mode via `opencode run`, and communicates with the LLM gateway proxy on a dedicated port (10004). Provider-specific API domains for the firewall allowlist are resolved at compile time by parsing the `provider/model` string prefix; the default provider is Anthropic. All tool permissions inside the OpenCode sandbox are pre-set to `allow` via an `opencode.jsonc` config file written before execution, which prevents the CI runner from hanging on interactive permission prompts.

### Alternatives Considered

#### Alternative 1: Custom engine wrapper via `engine.command`

Users can already specify `engine.command: opencode run` as a custom command override in the workflow frontmatter, which lets them invoke OpenCode without any first-class engine support. This avoids adding engine code but forces every user to manually specify the install steps, configure the `opencode.jsonc` permissions file, and manage firewall domains themselves. For a community-maintained open-source tool with growing adoption, first-class support provides substantially better UX with correct defaults out of the box.

#### Alternative 2: Extend an existing engine (e.g., Claude) with multi-provider model routing

Rather than adding a new engine, the Claude engine could be extended to accept `openai/` or `google/` model prefixes and route them to alternative providers through the LLM gateway. This avoids maintaining a separate engine abstraction but conflates two distinct CLIs (Claude Code CLI vs. OpenCode CLI) under the same engine ID, creating confusion for end users and making the firewall and installation logic more complex. OpenCode has its own installation artifact, config format (`opencode.jsonc`), and binary — they are genuinely different engines, not model variants.

#### Alternative 3: Static multi-provider domain allowlist

Instead of parsing the model string to derive the firewall domain at compile time, include all known provider API endpoints in `OpenCodeDefaultDomains` statically. This is simpler but violates the principle of least privilege: a workflow using only the Anthropic provider would unnecessarily have `api.openai.com` and `generativelanguage.googleapis.com` in its allowlist. The current implementation includes only the three most common providers in the static default (`OpenCodeDefaultDomains`) as a broad fallback, while `GetOpenCodeDefaultDomains(model)` provides a narrower per-provider list when a model is explicitly configured.

### Consequences

#### Positive
- Users can run any of 75+ models from multiple providers (Anthropic, OpenAI, Google, Groq, Mistral, DeepSeek, xAI) through a single engine selector.
- The BYOK model removes dependency on GitHub Copilot entitlements; any user with a direct provider API key can run agentic workflows.
- Dynamic per-provider domain resolution keeps firewall allowlists as narrow as possible given the selected model.
- The existing `BaseEngine` and engine registry patterns are reused without modification, keeping the diff small and coherent.

#### Negative
- The engine is marked `experimental: true` until smoke tests pass consistently; production readiness is deferred.
- OpenCode does not yet support `--max-turns` or gh-aw's neutral web-search tool abstraction (`supportsMaxTurns: false`, `supportsWebSearch: false`), limiting parity with other engines.
- The `openCodeProviderDomains` map in `domains.go` must be manually kept in sync as OpenCode adds or removes supported providers; there is no automated drift detection.
- Pre-setting all permissions to `allow` in `opencode.jsonc` disables OpenCode's interactive safety guardrails in CI. This is intentional (CI can't answer prompts) but means the agent runs with elevated tool permissions inside the sandbox.

#### Neutral
- A separate LLM gateway port (10004) is allocated for OpenCode, distinct from other engines. This adds one more well-known port constant to `pkg/constants/version_constants.go`.
- The MCP config integration follows the same `renderStandardJSONMCPConfig` path as other JSON-based engines; no new MCP config format is introduced.
- 22 unit tests cover the new engine (identity, capabilities, secrets, installation, execution, firewall, and provider extraction). These are co-located with other engine tests in `pkg/workflow/`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Engine Registration

1. The OpenCode engine **MUST** be registered in `NewEngineRegistry()` under the identifier `"opencode"`.
2. The OpenCode engine **MUST** implement the `AgenticEngine` interface via `BaseEngine` embedding, consistent with all other built-in engines.
3. The OpenCode engine **MUST** be included in `AgenticEngines` and `EngineOptions` so that tooling that enumerates built-in engines discovers it automatically.

### Installation

1. The engine **MUST** install the OpenCode CLI from npm using the pinned package version defined by `DefaultOpenCodeVersion` in `pkg/constants/version_constants.go`.
2. The engine **MUST** skip installation steps when `engine.command` is explicitly overridden in the workflow configuration.
3. The engine **SHOULD** use `BuildStandardNpmEngineInstallSteps` to generate installation steps so that any future changes to the standard npm install pattern apply automatically.

### Execution

1. The engine **MUST** write an `opencode.jsonc` configuration file to `$GITHUB_WORKSPACE` before executing the agent, with all tool permissions (`bash`, `edit`, `read`, `glob`, `grep`, `write`, `webfetch`, `websearch`) set to `"allow"`.
2. The engine **MUST** merge the permissions config with any existing `opencode.jsonc` found in the workspace (using `jq` deep merge), rather than unconditionally overwriting it.
3. The engine **MUST** invoke OpenCode via `opencode run <prompt>` in headless mode, passing `--print-logs` and `--log-level DEBUG` for CI observability.
4. The engine **MUST** route LLM API calls through the local gateway proxy at port `OpenCodeLLMGatewayPort` (10004) by setting `ANTHROPIC_BASE_URL` when the firewall is enabled.
5. The engine **MUST NOT** pass `--max-turns` to the OpenCode CLI, as that flag is not supported.

### Firewall Domain Allowlisting

1. When a model is explicitly configured in `engine.model`, the compiler **MUST** call `GetOpenCodeDefaultDomains(model)` to resolve provider-specific API domains from the `provider/model` prefix.
2. The `extractProviderFromModel` function **MUST** parse the model string by splitting on the first `/` character and returning the left-hand token, lowercased.
3. When no `/` separator is found in the model string, `extractProviderFromModel` **MUST** return `"anthropic"` as the default provider.
4. The `openCodeProviderDomains` map **MUST** be the single source of truth for mapping provider names to their API hostnames; callers **MUST NOT** hardcode provider domain strings outside this map.
5. The `engineDefaultDomains` map in `domains.go` **MUST** include an entry for `constants.OpenCodeEngine` to ensure `GetAllowedDomainsForEngine` works correctly for the OpenCode engine.

### Secret Collection

1. The engine **MUST** include `ANTHROPIC_API_KEY` in the required secret list as the default provider secret.
2. The engine **MUST** include additional secrets from `engine.env` whose key names end in `_API_KEY` or `_KEY`, to support non-default provider configurations.
3. The engine **MUST** collect common MCP secrets via `collectCommonMCPSecrets` and HTTP MCP header secrets via `collectHTTPMCPHeaderSecrets`, consistent with other engines.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the OpenCode engine **MUST** be registered, install via npm at a pinned version, write a complete permissions config before execution, invoke `opencode run` in headless mode, and resolve firewall domains dynamically from the model provider prefix. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
