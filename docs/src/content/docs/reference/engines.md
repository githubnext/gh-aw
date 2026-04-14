---
title: AI Engines (aka Coding Agents)
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Copilot, Claude, Codex, and Gemini with their specific configuration options.
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

Copilot CLI is the default — `engine:` can be omitted when using Copilot. See the linked authentication docs for secret setup instructions.

## Engine Feature Comparison

Not all features are available across all engines. The table below summarizes per-engine support for commonly used workflow options:

| Feature | Copilot | Claude | Codex | Gemini |
|---------|:-------:|:------:|:-----:|:------:|
| `max-turns` | ❌ | ✅ | ❌ | ❌ |
| `max-continuations` | ✅ | ❌ | ❌ | ❌ |
| `tools.web-fetch` | ✅ | ✅ | ✅ | ✅ |
| `tools.web-search` | via MCP | via MCP | ✅ (opt-in) | via MCP |
| `engine.agent` (custom agent file) | ✅ | ❌ | ❌ | ❌ |
| `engine.api-target` (custom endpoint) | ✅ | ✅ | ✅ | ✅ |
| Tools allowlist | ✅ | ✅ | ✅ | ✅ |

**Notes:**
- `max-turns` limits the number of AI chat iterations per run (Claude only).
- `max-continuations` enables autopilot mode with multiple consecutive runs (Copilot only).
- `web-search` for Codex is disabled by default; add `tools: web-search:` to enable it. Other engines use a third-party MCP server — see [Using Web Search](/gh-aw/guides/web-search/).
- `engine.agent` references a `.github/agents/` file for custom Copilot agent behavior. See [Copilot Custom Configuration](#copilot-custom-configuration).

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

**GHEC example** — specify your tenant-specific Copilot endpoint:

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

**GHES example** — use the enterprise Copilot endpoint:

```yaml wrap
engine:
  id: copilot
  api-target: api.enterprise.githubcopilot.com
network:
  allowed:
    - defaults
    - github.company.com
    - api.enterprise.githubcopilot.com
```

The specified hostname must also be listed in `network.allowed` for the firewall to permit outbound requests.

#### Custom API Endpoints via Environment Variables

Three environment variables receive special treatment when set in `engine.env`: `OPENAI_BASE_URL` (for `codex`), `ANTHROPIC_BASE_URL` (for `claude`), `GITHUB_COPILOT_BASE_URL` (for `copilot`), and `GEMINI_API_BASE_URL` (for `gemini`). When any of these is present, the API proxy automatically routes API calls to the specified host instead of the default endpoint. Firewall enforcement remains active, but this routing layer is not a separate authentication boundary for arbitrary code already running inside the agent container.

This enables workflows to use internal LLM routers, Azure OpenAI deployments, corporate Copilot proxies, or other compatible endpoints without bypassing AWF's security model.

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
    - llm-router.internal.example.com   # must be listed here for the firewall to permit outbound requests
```

For Claude workflows routed through a custom Anthropic-compatible endpoint:

```yaml wrap
engine:
  id: claude
  env:
    ANTHROPIC_BASE_URL: "https://anthropic-proxy.internal.example.com"
    ANTHROPIC_API_KEY: ${{ secrets.PROXY_API_KEY }}

network:
  allowed:
    - github.com
    - anthropic-proxy.internal.example.com
```

For Copilot workflows routed through a custom Copilot-compatible endpoint (e.g., a corporate proxy or a GHE Cloud data residency instance):

```yaml wrap
engine:
  id: copilot
  env:
    GITHUB_COPILOT_BASE_URL: "https://copilot-proxy.corp.example.com"

network:
  allowed:
    - github.com
    - copilot-proxy.corp.example.com
```

`GITHUB_COPILOT_BASE_URL` is used as a fallback when `engine.api-target` is not explicitly set. If both are configured, `engine.api-target` takes precedence.

For Gemini workflows routed through a custom Gemini-compatible endpoint:

```yaml wrap
engine:
  id: gemini
  env:
    GEMINI_API_BASE_URL: "https://gemini-proxy.internal.example.com"
    GEMINI_API_KEY: ${{ secrets.PROXY_API_KEY }}

network:
  allowed:
    - github.com
    - gemini-proxy.internal.example.com
```

The custom hostname is extracted from the URL and passed to the AWF `--openai-api-target`, `--anthropic-api-target`, `--copilot-api-target`, or `--gemini-api-target` flag automatically at compile time. No additional configuration is required.

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

| Engine | Default tool timeout |
|--------|----------------------|
| Copilot | not enforced by gh-aw (engine-managed) |
| Claude | 60 s |
| Codex | 120 s |
| Gemini | not enforced by gh-aw (engine-managed) |

See [Tool Timeout Configuration](/gh-aw/reference/tools/#tool-timeout-configuration) for full documentation including `tools.startup-timeout`.

### Per-Engine Timeout Controls

#### Copilot

Copilot does not expose a per-turn wall-clock time limit directly. Use `max-continuations` to control how many sequential agent runs are allowed in autopilot mode, and `timeout-minutes` for the overall job budget:

```yaml wrap
engine:
  id: copilot
max-continuations: 3   # up to 3 consecutive autopilot runs
timeout-minutes: 60
```

#### Claude

Claude supports `max-turns` to cap the number of AI iterations per run. Set it together with `tools.timeout` to control both breadth (number of turns) and depth (time per tool call):

```yaml wrap
engine:
  id: claude
max-turns: 20          # maximum number of agentic iterations
tools:
  timeout: 600         # 10 minutes per bash/tool call
timeout-minutes: 60
```

The `CLAUDE_CODE_MAX_TURNS` environment variable is a Claude Code CLI equivalent of `max-turns`. When `max-turns` is set in frontmatter, gh-aw passes it to the Claude CLI automatically — you do not need to set this env var separately.

#### Codex

Codex does not support `max-turns`. Use `tools.timeout` and `timeout-minutes` to control execution budgets:

```yaml wrap
engine:
  id: codex
tools:
  timeout: 300         # 5 minutes per tool call
timeout-minutes: 60
```

#### Gemini

Gemini does not support `max-turns` or `max-continuations`. Use `timeout-minutes` and `tools.timeout` to bound execution:

```yaml wrap
engine:
  id: gemini
tools:
  timeout: 300
timeout-minutes: 60
```

### Summary Table

| Timeout knob | Copilot | Claude | Codex | Gemini | Notes |
|---|:---:|:---:|:---:|:---:|---|
| `timeout-minutes` | ✅ | ✅ | ✅ | ✅ | Job-level wall clock |
| `tools.timeout` | ✅ | ✅ | ✅ | ✅ | Per tool-call limit (seconds) |
| `tools.startup-timeout` | ✅ | ✅ | ✅ | ✅ | MCP server startup limit |
| `max-turns` | ❌ | ✅ | ❌ | ❌ | Iteration budget (Claude only) |
| `max-continuations` | ✅ | ❌ | ❌ | ❌ | Autopilot run budget (Copilot only) |

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/introduction/architecture/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
- [Long Build Times](/gh-aw/reference/sandbox/#long-build-times) - Timeout tuning for large repositories
- [Self-Hosted Runners](/gh-aw/guides/self-hosted-runners/) - Fast hardware for long-running workflows
