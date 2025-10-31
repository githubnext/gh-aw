---
title: AI Engines
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Claude, Copilot, Codex, and custom engines with their specific configuration options.
sidebar:
  order: 600
---

GitHub Agentic Workflows support multiple AI engines (coding agents) to interpret and execute natural language instructions. Each engine has unique capabilities and configuration options.

### GitHub Copilot (Default)

GitHub Copilot is the default and recommended AI engine for most workflows. The [GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) provides MCP server support and is designed for conversational AI workflows.

```yaml
engine: copilot
```

#### Extended Configuration

```yaml
engine:
  id: copilot
  version: latest                       # Optional: defaults to latest
  model: gpt-5                          # Optional: defaults to claude-sonnet-4
  args: ["--add-dir", "/workspace"]     # Optional: custom CLI arguments
```

#### Configuration Options

- **`model`**: AI model (`gpt-5` or `claude-sonnet-4`)
- **`version`**: CLI version to install
- **`args`**: Custom command-line arguments (supported by all engines)

#### Environment Variables

- **`COPILOT_MODEL`**: Alternative way to set the model

#### Required Secrets

- **`COPILOT_CLI_TOKEN`**: GitHub Personal Access Token (PAT) with "Copilot Requests" permission
- **`GH_AW_GITHUB_TOKEN`** (optional): Required for [GitHub Tools Remote Mode](/gh-aw/reference/tools/#github-remote-mode)

#### Authenticating with a Personal Access Token (PAT)

To use the Copilot engine, you need a fine-grained Personal Access Token with the "Copilot Requests" permission enabled:

1. Visit https://github.com/settings/personal-access-tokens/new
2. Under "Permissions," click "add permissions" and select "Copilot Requests"
3. Generate your token
4. Add the token to your repository secrets as `COPILOT_CLI_TOKEN`:

```bash
gh secret set COPILOT_CLI_TOKEN -a actions --body "<your-github-pat>"
```

For GitHub Tools Remote Mode, also configure:
```bash
gh secret set GH_AW_GITHUB_TOKEN -a actions --body "<your-github-pat>"
```

For more information about GitHub Copilot CLI authentication, see the [official documentation](https://github.com/github/copilot-cli?tab=readme-ov-file#authenticate-with-a-personal-access-token-pat).

:::note
The Copilot engine does not have built-in `web-search` support. You can add web search capabilities using third-party MCP servers. See the [Using Web Search](/gh-aw/guides/web-search/) for available options and setup instructions.
:::

#### Network Permissions

The Copilot engine supports network access control through the `network:` configuration at the workflow level. When network permissions are configured, you can enable AWF (Agent Workflow Firewall) to enforce domain-based access controls. AWF is sourced from [github.com/githubnext/gh-aw-firewall](https://github.com/githubnext/gh-aw-firewall).

Enable network permissions and firewall in your workflow:

```yaml
engine: copilot

network:
  firewall: true           # Enable AWF enforcement
  allowed:
    - defaults             # Basic infrastructure domains
    - python              # Python ecosystem
    - "api.example.com"   # Custom domain
```

When enabled, AWF wraps the Copilot CLI execution and enforces the configured domain allowlist, logging all network activity for audit purposes. This provides network egress control and an additional layer of security for workflows that need strict network access control.

**Advanced Firewall Configuration:**

Additional AWF settings can be configured through the network configuration:

```yaml
network:
  allowed:
    - defaults
    - python
  firewall:
    version: "v1.0.0"                    # Optional: AWF version (defaults to latest)
    log-level: debug                     # Optional: debug, info (default), warn, error
    args: ["--custom-arg", "value"]      # Optional: additional AWF arguments
```

**Firewall Configuration Formats:**

The `firewall` field supports multiple formats:

```yaml
# Enable with defaults
network:
  firewall: true

# Enable with empty object (same as true)
network:
  firewall:

# Configure log level
network:
  firewall:
    log-level: info    # Options: debug, info (default), warn, error

# Disable firewall (triggers warning if allowed domains are specified)
network:
  allowed: ["example.com"]
  firewall: "disable"

# Custom configuration with version and arguments
network:
  firewall:
    version: "v0.1.0"
    log-level: debug
    args: ["--verbose"]
```

:::caution
Using `firewall: "disable"` with `network.allowed` domains will emit a warning in normal mode and an error in strict mode, as the network may not be properly sandboxed.
:::

See the [Network Permissions](/gh-aw/reference/network/) documentation for details on configuring allowed domains and ecosystem identifiers.

### Anthropic Claude Code

Claude Code excels at reasoning, code analysis, and understanding complex contexts.

```yaml
engine: claude
```

#### Extended Configuration

```yaml
engine:
  id: claude
  version: beta
  model: claude-3-5-sonnet-20241022
  max-turns: 5
  args: ["--custom-flag", "value"]      # Optional: custom CLI arguments
  env:
    AWS_REGION: us-west-2
    DEBUG_MODE: "true"
```

#### Required Secrets

- **`ANTHROPIC_API_KEY`**: Anthropic API key
- **`GH_AW_GITHUB_TOKEN`** (optional): Required for [GitHub Tools Remote Mode](/gh-aw/reference/tools/#github-remote-mode)

Set secrets using:
```bash
gh secret set ANTHROPIC_API_KEY -a actions --body "<your-anthropic-api-key>"
gh secret set GH_AW_GITHUB_TOKEN -a actions --body "<your-github-pat>"
```

### OpenAI Codex

OpenAI Codex CLI with MCP server support. Designed for code-focused tasks.

```yaml
engine: codex
```

#### Extended Configuration

```yaml
engine:
  id: codex
  model: gpt-4
  args: ["--custom-flag", "value"]      # Optional: custom CLI arguments
  user-agent: custom-workflow-name      # Optional: custom user agent for GitHub MCP
  env:
    CODEX_API_KEY: ${{ secrets.CODEX_API_KEY_CI }}
  config: |
    [custom_section]
    key1 = "value1"
    [server_settings]
    timeout = 60
```

#### Codex-Specific Fields

- **`user-agent`**: Custom user agent string for GitHub MCP server
- **`config`**: Additional TOML configuration appended to generated config.toml
- **`args`**: Custom command-line arguments (supported by all engines)

#### Required Secrets

- **`OPENAI_API_KEY`**: OpenAI API key

Set secrets using:
```bash
gh secret set OPENAI_API_KEY -a actions --body "<your-openai-api-key>"
```

### Custom Engine

Define custom GitHub Actions steps without AI interpretation for deterministic workflows.

```yaml
engine: custom
```

#### Extended Configuration

```yaml
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

```yaml
engine:
  id: claude
  env:
    DEBUG_MODE: "true"
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
```

## Engine Command-Line Arguments

All engines support custom command-line arguments through the `args` field, injected before the prompt:

```yaml
engine:
  id: copilot
  args: ["--add-dir", "/workspace", "--verbose"]
```

Arguments are added in order and placed before the `--prompt` flag. Common uses include adding directories (`--add-dir`), enabling verbose logging (`--verbose`, `--debug`), and passing engine-specific flags. Consult the specific engine's CLI documentation for available flags.

## Engine Error Patterns

All engines support custom error pattern recognition for enhanced log validation:

```yaml
engine:
  id: codex
  error_patterns:
    - pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)"
      level_group: 2
      message_group: 3
      description: "Custom error format with timestamp"
```

## Migration Between Engines

Switch engines by changing the `engine` field in your frontmatter:

```yaml
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