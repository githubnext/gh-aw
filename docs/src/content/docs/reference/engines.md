---
title: AI Engines (aka Coding Agents)
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Copilot, Claude, Codex, Gemini, Crush, OpenCode, and Pi with their specific configuration options.
sidebar:
  order: 600
---

GitHub Agentic Workflows use [AI Engines](/gh-aw/reference/glossary/#engine) (normally a coding agent) to interpret and execute natural language instructions.

## Available Coding Agents

Set `engine:` in your workflow frontmatter and configure the corresponding secret:

| Engine | `engine:` value | Required Secret |
|--------|-----------------|-----------------|
| [GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) (default) | `copilot` | [COPILOT_GITHUB_TOKEN](/gh-aw/reference/auth/#copilot_github_token) |
| [Claude by Anthropic (Claude Code)](https://www.anthropic.com/index/claude) | `claude` | [ANTHROPIC_API_KEY](/gh-aw/reference/auth/#anthropic_api_key) |
| [OpenAI Codex](https://openai.com/blog/openai-codex) | `codex` | [OPENAI_API_KEY](/gh-aw/reference/auth/#openai_api_key) |
| [Google Gemini CLI](https://github.com/google-gemini/gemini-cli) | `gemini` | [GEMINI_API_KEY](/gh-aw/reference/auth/#gemini_api_key) |
| [Crush](https://github.com/charmbracelet/crush) (experimental) | `crush` | [COPILOT_GITHUB_TOKEN](/gh-aw/reference/auth/#copilot_github_token) |
| [OpenCode](https://opencode.ai) (experimental) | `opencode` | [COPILOT_GITHUB_TOKEN](/gh-aw/reference/auth/#copilot_github_token) |
| [Pi](https://www.npmjs.com/package/@earendil-works/pi-coding-agent) (experimental) | `pi` | [COPILOT_GITHUB_TOKEN](/gh-aw/reference/auth/#copilot_github_token) (default); switches to provider-specific secret when `model:` uses `provider/model` format |

Copilot CLI is the default — `engine:` can be omitted when using Copilot. See the linked authentication docs for secret setup instructions.

## Which engine should I choose?

Choose the engine that best matches your needs and existing AI account: Copilot supports the broadest gh-aw feature set, including custom agents and autopilot-style continuations; Claude offers stronger control over turn limits (`max-turns`) for long reasoning sessions; and Gemini or Codex fit well when those models are already part of existing tooling or budget decisions. You can switch later by changing only `engine:` and the corresponding secret.

## Engine Feature Comparison

Not all features are available across all engines. The table below summarizes per-engine support for commonly used workflow options:

| Feature | Copilot | Claude | Codex | Gemini | Crush | OpenCode | Pi |
|---------|:-------:|:------:|:-----:|:------:|:-----:|:--------:|:--:|
| `max-runs` (AWF invocation cap) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `max-turns` | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `max-continuations` | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `tools.web-fetch` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `tools.web-search` | via MCP | via MCP | ✅ (opt-in) | via MCP | via MCP | via MCP | via MCP |
| `engine.agent` (custom agent file) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `engine.api-target` (custom endpoint) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `engine.bare` (disable context loading) | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| `engine.harness` (custom harness script) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Tools allowlist | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |

`max-runs` (default `500`) and `max-effective-tokens` (default `25M`) are top-level frontmatter fields supported by all engines. `max-turns` limits Claude iterations per run; `max-continuations` enables Copilot autopilot mode. Codex `web-search` is opt-in via `tools: web-search:`; other engines use a third-party MCP server — see [Using Web Search](/gh-aw/reference/web-search/). `engine.agent`, `engine.bare`, and `engine.harness` are described below.

## Extended Coding Agent Configuration

Workflows can specify extended configuration for the coding agent:

```yaml wrap
engine:
  id: copilot
  version: latest                       # defaults to latest
  model: gpt-5                          # example override; omit to use engine default
  command: /usr/local/bin/copilot       # custom executable path
  args: ["--add-dir", "/workspace"]     # custom CLI arguments
  agent: agent-id                       # custom agent file identifier
  api-target: api.acme.ghe.com          # custom API endpoint hostname (GHEC/GHES)
```

### Pinning a Specific Engine Version

By default, workflows install the latest available version of each engine CLI. To pin to a specific version, set `version` to the desired release:

| Engine | `id` | Example `version` |
|--------|------|-------------------|
| GitHub Copilot CLI | `copilot` | `"0.0.422"` |
| Claude Code | `claude` | `"2.1.70"` |
| Codex | `codex` | `"0.111.0"` |
| Gemini CLI | `gemini` | `"0.31.0"` |
| Crush | `crush` | `"1.2.14"` |
| OpenCode | `opencode` | `"0.1.0"` |
| Pi | `pi` | `"0.72.1"` |

```yaml wrap
engine:
  id: copilot
  version: "0.0.422"
```

Pinning is useful when you need reproducible builds or want to avoid breakage from a new CLI release while testing. Remember to update the pinned version periodically to pick up bug fixes and new features.

`version` also accepts a GitHub Actions expression string, enabling `workflow_call` reusable workflows to parameterize the engine version via caller inputs. Expressions are passed injection-safely through an environment variable rather than direct shell interpolation:

```yaml wrap
on:
  workflow_call:
    inputs:
      engine-version:
        type: string
        default: latest

---

engine:
  id: copilot
  version: ${{ inputs.engine-version }}
```

### Copilot Custom Configuration

Use `agent` to reference a custom agent file in `.github/agents/` (omit the `.agent.md` extension):

```yaml wrap
engine:
  id: copilot
  agent: technical-doc-writer  # .github/agents/technical-doc-writer.agent.md
```

See [Copilot Agent Files](/gh-aw/reference/copilot-custom-agents/) for details.

### Engine Environment Variables

All engines support custom environment variables through the `env` field:

```yaml wrap
engine:
  id: copilot
  env:
    DEBUG_MODE: "true"
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
```

Environment variables can also be defined at workflow, job, step, and other scopes. See [Environment Variables](/gh-aw/reference/environment-variables/) for complete documentation on precedence and all 13 env scopes.

### Enterprise API Endpoint (`api-target`)

The `api-target` field specifies a custom API endpoint hostname for the agentic engine. Use this when running workflows against GitHub Enterprise Cloud (GHEC), GitHub Enterprise Server (GHES), or any custom AI endpoint.

For a complete setup and debugging walkthrough for GHE Cloud with data residency, see [Debugging GHE Cloud with Data Residency](/gh-aw/troubleshooting/debug-ghe/).

The value must be a hostname only — no protocol or path (e.g., `api.acme.ghe.com`, not `https://api.acme.ghe.com/v1`). The field works with any engine.

**Example** — specify a GHEC or GHES Copilot endpoint (use `api.enterprise.githubcopilot.com` for GHES):

```yaml wrap
engine:
  id: copilot
  api-target: api.acme.ghe.com
network:
  allowed:
    - defaults
    - acme.ghe.com
    - api.acme.ghe.com
```

The specified hostname must also be listed in `network.allowed` for the firewall to permit outbound requests.

#### Custom API Endpoints via Environment Variables

Set a base URL environment variable in `engine.env` to route API calls to an internal LLM router, Azure OpenAI deployment, or corporate proxy. AWF automatically extracts the hostname and applies it to the API proxy. The target domain must also appear in `network.allowed`.

| Engine | Environment variable |
|--------|---------------------|
| `codex`, `crush` | `OPENAI_BASE_URL` |
| `claude` | `ANTHROPIC_BASE_URL` |
| `copilot` | `GITHUB_COPILOT_BASE_URL` |
| `gemini` | `GEMINI_API_BASE_URL` |

```yaml wrap
engine:
  id: codex
  model: gpt-4o
  env:
    OPENAI_BASE_URL: "https://llm-router.internal.example.com/v1"
    OPENAI_API_KEY: ${{ secrets.LLM_ROUTER_KEY }}

network:
  allowed:
    - github.com
    - llm-router.internal.example.com
```

`GITHUB_COPILOT_BASE_URL` is a fallback — if both it and `engine.api-target` are set, `engine.api-target` takes precedence. Crush uses OpenAI-compatible API format; its `model` field uses `provider/model` format (e.g., `openai/gpt-4o`).

### Copilot Bring Your Own Key (BYOK) Mode

The Copilot engine supports routing requests to an external LLM provider instead of GitHub's default routing. This is useful when you want to use a different model or provider (e.g., OpenAI, Anthropic, Azure OpenAI, or a local Ollama/vLLM instance) while still using the Copilot CLI tooling.

Set `COPILOT_PROVIDER_BASE_URL` in `engine.env` to activate BYOK mode. The credential variables `COPILOT_PROVIDER_BASE_URL`, `COPILOT_PROVIDER_API_KEY`, and `COPILOT_PROVIDER_BEARER_TOKEN` are explicitly allowed to carry `${{ secrets.* }}` references in `engine.env` under strict mode — they are not leaked to the agent container. Other `COPILOT_PROVIDER_*` variables hold non-sensitive configuration and can be set as plain strings.

| Variable | Required | Description |
|---|---|---|
| `COPILOT_PROVIDER_BASE_URL` | ✅ for BYOK | Base URL of the external provider (e.g. `https://api.openai.com/v1`) |
| `COPILOT_MODEL` | ✅ for BYOK | Model to use (e.g. `claude-sonnet-4`, `gpt-4o`); required by most providers |
| `COPILOT_PROVIDER_API_KEY` | Optional | API key for cloud providers (OpenAI, Anthropic, etc.); not needed for local providers |
| `COPILOT_PROVIDER_BEARER_TOKEN` | Optional | Bearer token alternative to `COPILOT_PROVIDER_API_KEY`; takes precedence when set |
| `COPILOT_PROVIDER_TYPE` | Optional | Provider format: `openai` (default), `azure`, or `anthropic` |
| `COPILOT_PROVIDER_WIRE_API` | Optional | Wire API variant: `completions` (default) or `responses` (for GPT-5 series) |
| `COPILOT_PROVIDER_MODEL_ID` | Optional | Model ID sent on the wire when it differs from `COPILOT_MODEL` (e.g. an Azure deployment name) |
| `COPILOT_PROVIDER_WIRE_MODEL` | Optional | Alternative to `COPILOT_PROVIDER_MODEL_ID` for overriding the wire model |
| `COPILOT_PROVIDER_MAX_PROMPT_TOKENS` | Optional | Override the maximum prompt token limit (otherwise resolved from model catalog) |
| `COPILOT_PROVIDER_MAX_OUTPUT_TOKENS` | Optional | Override the maximum output token limit |

**Example:**

```yaml wrap
engine:
  id: copilot
  env:
    COPILOT_PROVIDER_BASE_URL: ${{ secrets.PROVIDER_BASE_URL }}   # REQUIRED — activates BYOK
    COPILOT_MODEL: claude-sonnet-4                                # REQUIRED for most providers
    COPILOT_PROVIDER_API_KEY: ${{ secrets.PROVIDER_API_KEY }}     # OPTIONAL for local providers
    COPILOT_PROVIDER_TYPE: anthropic                              # OPTIONAL — default: openai

network:
  allowed:
    - defaults
    - your-provider-domain.example.com
```

> [!NOTE]
> Credentials are kept out of the agent container — only a dummy API key activating the AWF BYOK detection path is visible to the agent process; the real credential is isolated in the AWF API proxy sidecar. See [AWF sandbox architecture](/gh-aw/reference/sandbox/).

### Engine Command-Line Arguments

All engines support custom command-line arguments through the `args` field, injected before the prompt:

```yaml wrap
engine:
  id: copilot
  args: ["--add-dir", "/workspace", "--verbose"]
```

Arguments are added in order and placed before the `--prompt` flag. Consult the specific engine's CLI documentation for available flags.

### Custom Engine Command

Override the default engine executable using the `command` field. Useful for testing pre-release versions, custom builds, or non-standard installations. Installation steps are automatically skipped.

```yaml wrap
engine:
  id: copilot
  command: /usr/local/bin/copilot-dev  # absolute path
  args: ["--verbose"]
```

### Custom Harness Script (`harness`)

The `harness` field lets you replace the built-in Node.js harness wrapper that the Copilot engine uses to launch the CLI. Use this when you need to customize startup behavior, inject pre/post hooks, or test an alternative harness implementation.

```yaml wrap
engine:
  id: copilot
  harness: custom_copilot_harness.cjs
```

The value must be a bare filename — no directory separators, no `..`, and no shell metacharacters. It must end with `.js`, `.cjs`, or `.mjs`. When `harness` is set, AWF automatically ensures Node 24 is available in the runner environment.

> [!NOTE]
> `engine.harness` is currently only applied during Copilot engine execution. Setting it on other engines has no effect.

**Validation rules:**

| Rule | Valid example | Invalid example |
|------|--------------|-----------------|
| Bare filename only | `my_harness.cjs` | `subdir/harness.cjs` |
| No path traversal | `harness.mjs` | `../harness.cjs` |
| Must start with `[A-Za-z0-9_]` | `harness.js` | `-harness.cjs` |
| Must end with `.js`, `.cjs`, or `.mjs` | `wrapper.cjs` | `harness.sh` |

### Bare Mode (`bare`)

Set `engine.bare: true` to disable automatic loading of context and custom instructions by the engine. Use this when the workflow prompt is fully self-contained and you want to prevent the engine from reading memory files, AGENTS.md, or built-in system prompts that would otherwise be loaded automatically.

```yaml wrap
engine:
  id: claude
  bare: true
```

The underlying mechanism is engine-specific:

| Engine | Effect |
|--------|--------|
| Copilot | Passes `--no-custom-instructions` — suppresses `.github/AGENTS.md` and user-level custom instructions |
| Claude | Passes `--bare` — suppresses CLAUDE.md memory files |
| Codex | Passes `--no-system-prompt` — suppresses the default system prompt |
| Gemini | Sets `GEMINI_SYSTEM_MD=/dev/null` — overrides the built-in system prompt with an empty file |

Defaults to `false`.

### Custom Token Weights (`token-weights`)

Override the built-in token cost multipliers used when computing [Effective Tokens](/gh-aw/reference/effective-tokens-specification/). Useful when your workflow uses a custom model not in the built-in list, or when you want to adjust the relative cost ratios for your use case.

```yaml wrap
engine:
  id: claude
  token-weights:
    multipliers:
      my-custom-model: 2.5      # 2.5x the cost of claude-sonnet-4.5
      experimental-llm: 0.8    # Override an existing model's multiplier
    token-class-weights:
      output: 6.0              # Override output token weight (default: 4.0)
      cached-input: 0.05       # Override cached input weight (default: 0.1)
```

`multipliers` is a map of model names to numeric multipliers relative to `claude-sonnet-4.5` (= 1.0). Keys are case-insensitive and support prefix matching. `token-class-weights` overrides the per-class weights applied before the model multiplier; the defaults are `input: 1.0`, `cached-input: 0.1`, `output: 4.0`, `reasoning: 4.0`, `cache-write: 1.0`.

Custom weights are embedded in the compiled workflow YAML and read by `gh aw logs` and `gh aw audit` when analyzing runs.

## Timeout Configuration

Repositories with long build or test cycles require careful timeout tuning at multiple levels. This section documents the timeout knobs available for each engine.

### Job-Level Timeout (`timeout-minutes`)

`timeout-minutes` sets the maximum wall-clock time for the entire agent job. This is the primary knob for repositories with long build times. The default is 20 minutes.

```yaml wrap
timeout-minutes: 60   # allow up to 60 minutes for the agent job
```

See [Long Build Times](/gh-aw/reference/sandbox/#long-build-times) in the Sandbox reference for recommended values and concrete examples, including a 30-minute C++ workflow.

### Per-Tool-Call Timeout (`tools.timeout`)

`tools.timeout` limits how long any single tool invocation may run, in seconds. Useful when individual `bash` commands (builds, test suites) take longer than an engine's default:

```yaml wrap
tools:
  timeout: 300   # 5 minutes per tool call
```

Defaults: Claude `60s`, Codex `120s`. Other engines (Copilot, Gemini, Crush) are engine-managed and not enforced by gh-aw. See [Tool Timeout Configuration](/gh-aw/reference/tools/#tool-timeout-configuration) for full documentation including `tools.startup-timeout`.

### Per-Engine Timeout Controls

| Knob | Copilot | Claude | Codex/Gemini/Crush/OpenCode | Purpose |
|---|:---:|:---:|:---:|---|
| `timeout-minutes` | ✅ | ✅ | ✅ | Job-level wall clock |
| `tools.timeout` | ✅ | ✅ | ✅ | Per tool-call limit (seconds) |
| `tools.startup-timeout` | ✅ | ✅ | ✅ | MCP server startup limit |
| `max-turns` | ❌ | ✅ | ❌ | Iteration budget |
| `max-continuations` | ✅ | ❌ | ❌ | Autopilot run budget |

Copilot uses `max-continuations` for autopilot runs; Claude uses `max-turns` to cap iterations. Codex, Gemini, Crush, and OpenCode rely solely on `timeout-minutes` and `tools.timeout`.

```yaml wrap
# Claude — combine iteration cap with per-tool timeout
engine:
  id: claude
max-turns: 20
tools:
  timeout: 600
timeout-minutes: 60
```

When `max-turns` is set in frontmatter, gh-aw passes it to Claude automatically — no need to also set the `CLAUDE_CODE_MAX_TURNS` env var.

## Claude Tool Enforcement Security Model

Claude Code accepts a `--permission-mode` flag that determines whether the declared `tools:` allowlist is enforced. Set `engine.permission-mode` to one of `auto`, `acceptEdits`, `plan`, or `bypassPermissions`:

```yaml wrap
engine:
  id: claude
  permission-mode: auto
```

`engine.permission-mode` takes precedence over any `--permission-mode` flag supplied through `engine.args`. When unset, the default is `acceptEdits` (or `auto` when `tools.edit: false`). gh-aw **does not** derive `bypassPermissions` implicitly from unrestricted bash — set it explicitly.

| `engine.permission-mode` | Effective mode | `--allowed-tools` enforced? | Gateway `allowed:` enforced? |
|---|---|:---:|:---:|
| unset (default) | `acceptEdits` | ✅ Yes | ✅ Yes |
| unset, with `tools.edit: false` | `auto` | ✅ Yes | ✅ Yes |
| `auto` | `auto` | ✅ Yes | ✅ Yes |
| `acceptEdits` | `acceptEdits` | ✅ Yes | ✅ Yes |
| `plan` | `plan` | ✅ Yes | ✅ Yes |
| `bypassPermissions` | `bypassPermissions` | ❌ No | ✅ Yes |

### Gateway-side enforcement

The MCP gateway's `allowed:` filter is the sole effective tool boundary in `bypassPermissions` mode (and a second layer of enforcement otherwise). Always specify `allowed:` on each `mcp-servers:` entry to restrict which MCP tools are reachable:

```yaml wrap
mcp-servers:
  notion:
    container: "mcp/notion"
    allowed: ["search_pages", "get_page"]   # enforced at gateway level
```

> [!WARNING]
> Do not rely on `tools:` or `mcp-servers: allowed:` for security guarantees in `bypassPermissions` mode. The agent can already run arbitrary shell commands when unrestricted bash is granted, so `--allowed-tools` provides no meaningful additional boundary.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/introduction/architecture/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
- [Long Build Times](/gh-aw/reference/sandbox/#long-build-times) - Timeout tuning for large repositories
- [Self-Hosted Runners](/gh-aw/reference/self-hosted-runners/) - Fast hardware for long-running workflows
