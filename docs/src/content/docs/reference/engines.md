---
title: AI Engines
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Copilot and custom engines with their specific configuration options.
sidebar:
  order: 600
---

GitHub Agentic Workflows support multiple AI [engines](/gh-aw/reference/glossary/#engine) (which AI model/provider to use) to interpret and execute natural language instructions. Each engine has unique capabilities and configuration options.

:::note[Experimental Engines]
Claude and Codex engines are available but marked as experimental. They are not documented here but can still be used by setting `engine: claude` or `engine: codex` in your workflow frontmatter. For production workflows, we recommend using the GitHub Copilot CLI engine.
:::

### GitHub Copilot CLI

GitHub Copilot is the default and recommended AI [engine](/gh-aw/reference/glossary/#engine) (which AI model/provider to use) for most workflows. The [GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) provides [Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP, the standardized protocol for connecting AI agents to tools) server support and is designed for conversational AI workflows.

```yaml wrap
engine: copilot
```

#### Extended Configuration

```yaml wrap
engine:
  id: copilot
  version: latest                       # defaults to latest
  model: gpt-5                          # defaults to claude-sonnet-4
  args: ["--add-dir", "/workspace"]     # custom CLI arguments
```

Configuration options: `model` (gpt-5 or claude-sonnet-4), `version` (CLI version), `args` (command-line arguments). Alternatively set model via `COPILOT_MODEL` environment variable.

#### Required Secrets

**`COPILOT_GITHUB_TOKEN`**: GitHub [Personal Access Token](/gh-aw/reference/glossary/#personal-access-token-pat) (PAT, a token that authenticates you to GitHub's APIs) with "Copilot Requests" permission. **`GH_AW_GITHUB_TOKEN`** (optional): Required for [GitHub Tools Remote Mode](/gh-aw/reference/tools/#modes-and-restrictions).

#### Authenticating with a Personal Access Token (PAT)

Create a fine-grained PAT at <https://github.com/settings/personal-access-tokens/new>. Select your user account (not an organization), choose "Public repositories" access, and enable "Copilot Requests" permissions. Then add it to your repository:

```bash wrap
gh aw secrets set COPILOT_GITHUB_TOKEN --value "<your-github-pat>"
```

:::caution[Legacy Tokens Removed]
The `COPILOT_CLI_TOKEN` and `GH_AW_COPILOT_TOKEN` secret names are **no longer supported** as of v0.26+. If you're using these tokens, please migrate to `COPILOT_GITHUB_TOKEN`.

The legacy secret name `GH_AW_GITHUB_TOKEN` is still supported for backward compatibility, but `COPILOT_GITHUB_TOKEN` is now the recommended name for Copilot operations.
:::

For GitHub Tools Remote Mode, also configure:

```bash wrap
gh aw secrets set GH_AW_GITHUB_TOKEN --value "<your-github-pat>"
```

For more information about GitHub Copilot CLI authentication, see the [official documentation](https://github.com/github/copilot-cli?tab=readme-ov-file#authenticate-with-a-personal-access-token-pat).

:::note
The Copilot engine does not have built-in `web-search` support. You can add web search capabilities using third-party MCP servers. See the [Using Web Search](/gh-aw/guides/web-search/) for available options and setup instructions.
:::

#### Network Permissions

The Copilot engine supports network access control through AWF (Agent Workflow Firewall) from [github.com/githubnext/gh-aw-firewall](https://github.com/githubnext/gh-aw-firewall). Enable it to enforce domain allowlists and log network activity:

```yaml wrap
engine: copilot
network:
  firewall: true           # or configure: { version, log-level, args }
  allowed:
    - defaults             # infrastructure domains
    - python              # ecosystem identifier
    - "api.example.com"   # custom domain
```

Advanced configuration: set `firewall.version` (defaults to latest), `log-level` (debug, info, warn, error), or `args` for additional AWF arguments. Use `firewall: false` or `"disable"` to disable.

### Agent Sandbox

:::note[Mandatory Sandbox]
The agent sandbox is now mandatory for all workflows and defaults to AWF (Agent Workflow Firewall). See [Sandbox Configuration](/gh-aw/reference/sandbox/) for details.
:::

The sandbox configuration is controlled via the `sandbox.agent` field:
- **AWF (default)**: Network egress control through domain-based access controls
- **SRT**: Enhanced isolation using Anthropic's Sandbox Runtime (experimental, Copilot only)

See [Network Permissions](/gh-aw/reference/network/) for domain configuration details.

### Custom Engine

Define custom GitHub Actions steps without AI interpretation for deterministic workflows.

```yaml wrap
engine: custom
```

#### Extended Configuration

```yaml wrap
engine:
  id: custom
  steps:
    - name: Install dependencies
      run: npm ci
```

## Custom Agent Files

All AI engines support custom agent files that provide specialized instructions and behavior. See the [Custom Agent Files](/gh-aw/reference/custom-agents/) reference for complete documentation on creating and using custom agents.

## Engine Environment Variables

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

## Engine Command-Line Arguments

All engines support custom command-line arguments through the `args` field, injected before the prompt:

```yaml wrap
engine:
  id: copilot
  args: ["--add-dir", "/workspace", "--verbose"]
```

Arguments are added in order and placed before the `--prompt` flag. Common uses include adding directories (`--add-dir`), enabling verbose logging (`--verbose`, `--debug`), and passing engine-specific flags. Consult the specific engine's CLI documentation for available flags.

## Engine Error Patterns

All engines (Copilot, Claude, Codex, and Custom) support custom error pattern recognition for enhanced log validation. This allows you to define project-specific error formats that should be detected in agent logs.

### Basic Usage

```yaml wrap
engine:
  id: copilot
  error_patterns:
    - pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)"
      level_group: 2
      message_group: 3
      description: "Custom error format with timestamp"
```

### Multiple Patterns

Define multiple error patterns to catch different error formats:

```yaml wrap
engine:
  id: claude
  error_patterns:
    - pattern: "PROJECT_ERROR:\\s+(.+)"
      message_group: 1
      description: "Project-specific error"
    - pattern: "VALIDATION_FAILED:\\s+(.+)"
      message_group: 1
      description: "Validation error"
```

### Pattern Fields

- **`pattern`** (required): ECMAScript regular expression to match log lines
- **`level_group`** (optional): Capture group index (1-based) containing error level (ERROR, WARNING, etc.). Use 0 to infer from pattern content.
- **`message_group`** (optional): Capture group index (1-based) containing the error message. Use 0 to use the entire match.
- **`description`** (optional): Human-readable description of what this pattern matches

### Shared Error Patterns

Error patterns can be defined in shared workflows and imported:

**`shared/error-patterns.md`:**
```yaml wrap
---
engine:
  error_patterns:
    - pattern: "SHARED_ERROR:\\s+(.+)"
      message_group: 1
      description: "Shared error pattern"
---
```

**Main workflow:**
```yaml wrap
---
imports:
  - ./shared/error-patterns.md
engine: copilot
---
```

Custom error patterns are merged with engine-specific built-in patterns during workflow compilation.

## Migration Between Engines

Switch engines by changing the `engine` field in your frontmatter:

```yaml wrap
# Simple switch
engine: copilot

# With configuration
engine:
  id: copilot
  model: gpt-5              # Optional; defaults to claude-sonnet-4
  version: latest
```

Engine-specific features may not be available when switching engines.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/guides/security/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
