---
title: AI Engines
description: Complete guide to AI engines available in GitHub Agentic Workflows, including Claude, Copilot, Codex, and custom engines with their specific configuration options.
sidebar:
  order: 1
---

GitHub Agentic Workflows support multiple AI engines to interpret and execute natural language instructions. Each engine has unique capabilities and configuration options.

## Agentic Engines

### Anthropic Claude Code (Default)

Claude Code is the default and recommended AI engine for most workflows. It excels at reasoning, code analysis, and understanding complex contexts.

```yaml
engine: claude
```

**Extended configuration:**
```yaml
engine:
  id: claude
  version: beta
  model: claude-3-5-sonnet-20241022
  max-turns: 5
  env:
    AWS_REGION: us-west-2
    DEBUG_MODE: "true"
```

#### Secrets

- `ANTHROPIC_API_KEY` secret is required for authentication.

### GitHub Copilot (Experimental)

[GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) with MCP server support. Designed for conversational AI workflows with access to GitHub repositories and development tools.

```yaml
engine: copilot
```

**Extended configuration:**
```yaml
engine:
  id: copilot
  version: latest
  model: gpt-5                          # Optional: uses claude-sonnet-4 by default
  env:
    GITHUB_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN }}
    DEBUG_MODE: "true"
```

**Features:**
- Conversational AI engine powered by GitHub Copilot
- Uses GitHub Copilot CLI (`@github/copilot`) for natural language processing
- Supports MCP servers for tool integration
- Works with file directories and project contexts
- Integrates with GitHub API and repositories

**Copilot-specific fields:**
- **`model`** (optional): AI model to use (`gpt-5` or defaults to `claude-sonnet-4`)
- **`version`** (optional): Version of the GitHub Copilot CLI to install (defaults to `latest`)

**Environment Variables:**
- **`COPILOT_MODEL`**: Alternative way to set the model (e.g., `gpt-5`)

#### Secrets

- `COPILOT_CLI_TOKEN` secret is required for authentication.

> **Important**: The standard GitHub Actions `GITHUB_TOKEN` is **not compatible** with GitHub Copilot CLI. You must use a Personal Access Token (PAT) with appropriate permissions.

**To obtain a compatible token:**

1. **For GitHub.com users**: Generate a Personal Access Token at [github.com/settings/tokens](https://github.com/settings/tokens)
   - Select **Classic** token or **Fine-grained** token
   - Required scopes: `repo`, `read:org` (for repository access)
   - Optional scopes: `copilot` (if available for enhanced Copilot features)

2. **For GitHub Enterprise users**: Contact your GitHub Enterprise administrator for Copilot CLI access tokens

3. **Add the token to your repository**:
   - Go to your repository → Settings → Secrets and variables → Actions
   - Create a new secret named `COPILOT_CLI_TOKEN`
   - Paste your Personal Access Token as the value

**Token Requirements:**
- Must have repository access permissions (`repo` scope)
- Must be associated with a GitHub account that has Copilot access
- Cannot be the default GitHub Actions `GITHUB_TOKEN`

### OpenAI Codex (Experimental)

OpenAI Codex CLI with MCP server support. Designed for code-focused tasks and integration scenarios.

```yaml
engine: codex
```

**Extended configuration:**
```yaml
engine:
  id: codex
  model: gpt-4
  user-agent: custom-workflow-name
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY_CI }}
  config: |
    [custom_section]
    key1 = "value1"
    key2 = "value2"
    
    [server_settings]
    timeout = 60
    retries = 3
```

**Features:**
- Code-focused AI engine
- Generates `config.toml` for MCP server configuration
- Supports custom TOML configuration via `config` field
- Configurable user agent for GitHub MCP server
- Requires `OPENAI_API_KEY` secret

**Codex-specific fields:**
- **`user-agent`** (optional): Custom user agent string for GitHub MCP server configuration
- **`config`** (optional): Additional TOML configuration text appended to generated config.toml

#### Secrets

- `OPENAI_API_KEY` secret is required for authentication.

### Custom Engine

For advanced users who want to define completely custom GitHub Actions steps instead of using AI interpretation.

```yaml
engine: custom
```

**Extended configuration:**
```yaml
engine:
  id: custom
  steps:
    - name: Custom step
      run: echo "Custom logic here"
    - uses: actions/setup-node@v4
      with:
        node-version: '18'
```

**Features:**
- Execute user-defined GitHub Actions steps
- No AI interpretation - direct step execution
- Useful for deterministic workflows or hybrid approaches

## Engine-Specific Configuration

### Environment Variables

All engines support custom environment variables through the `env` field:

```yaml
engine:
  id: claude
  env:
    DEBUG_MODE: "true"
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
```

**Common use cases:**
- Override default API keys (e.g., `OPENAI_API_KEY` for Codex)
- Set region-specific configuration
- Enable debug modes
- Configure custom endpoints

### Error Patterns

Claude, Copilot, and Codex engines support custom error pattern recognition for enhanced log validation:

```yaml
engine:
  id: codex
  error_patterns:
    - pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)"
      level_group: 2
      message_group: 3
      description: "Custom error format with timestamp"
```

## Codex Engine Advanced Configuration

The Codex engine supports additional customization through the `config` field, which allows you to append raw TOML configuration to the generated `config.toml` file.

### Custom Configuration Example

```yaml
engine:
  id: codex
  config: |
    # Custom logging configuration
    [logging]
    level = "debug"
    file = "/tmp/codex-debug.log"
    
    # Server timeout settings
    [server]
    timeout = 120
    max_connections = 10
    
    # Custom tool configurations
    [tools.custom_analyzer]
    enabled = true
    mode = "strict"
```

### Generated Output

This configuration generates a `config.toml` file with the structure:

```toml
[history]
persistence = "none"

[mcp_servers.github]
user_agent = "workflow-name"
command = "docker"
args = ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server:sha-09deac4"]
env = { "GITHUB_PERSONAL_ACCESS_TOKEN" = "${{ secrets.GITHUB_TOKEN }}" }

# Custom configuration
[logging]
level = "debug"
file = "/tmp/codex-debug.log"

[server]
timeout = 120
max_connections = 10

[tools.custom_analyzer]
enabled = true
mode = "strict"
```

### Best Practices for Custom Config

1. **Validate TOML**: Ensure your configuration is valid TOML syntax
2. **Avoid conflicts**: Don't override standard sections like `[history]` or `[mcp_servers.*]`
3. **Use descriptive sections**: Name your configuration sections clearly
4. **Document purpose**: Include comments in your TOML to explain custom settings
5. **Test thoroughly**: Validate that your custom configuration works as expected

## Copilot CLI Configuration

The Copilot engine provides comprehensive configuration options for the GitHub Copilot CLI integration, including model selection, directory context, and MCP server management.

### Available Models

GitHub Copilot CLI supports the following models:

- **Default**: `claude-sonnet-4` - Claude Sonnet 4 model (used when no model is specified)
- **Alternative**: `gpt-5` - GPT-5 model (set via `COPILOT_MODEL` environment variable or `--model` argument)

> **Important**: 
> - The default model used by GitHub Copilot CLI is Claude Sonnet 4. GitHub reserves the right to change this model.
> - Each workflow execution counts as one premium request against your monthly quota. For information about premium requests, see [Requests in GitHub Copilot](https://docs.github.com/en/copilot/managing-copilot/monitoring-usage-and-entitlements/about-premium-requests).
> - Model availability may vary based on your GitHub Copilot subscription and regional settings. 
> - Refer to the [GitHub Copilot CLI documentation](https://docs.github.com/en/copilot/concepts/agents/about-copilot-cli#model-usage) for the most current model availability.

### Advanced Configuration Example

```yaml
engine:
  id: copilot
  version: latest
  model: gpt-5                          # Optional: defaults to claude-sonnet-4
  env:
    GITHUB_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN }}
    COPILOT_MODEL: gpt-5                # Alternative way to set model
    XDG_CONFIG_HOME: /tmp/.copilot
    XDG_STATE_HOME: /tmp/.copilot
    DEBUG_MODE: "true"
```

### CLI Arguments and Options

The Copilot engine automatically configures the GitHub Copilot CLI with optimal settings:

- `--add-dir /tmp/` - Adds project directory context
- `--log-level debug` - Enables detailed logging
- `--log-dir /tmp/.copilot/logs/` - Configures log output directory
- `--model <model>` - Specifies the AI model (e.g., `gpt-5`; defaults to `claude-sonnet-4`)

### MCP Server Integration

Copilot works seamlessly with MCP servers for tool integration. The engine automatically:
- Generates MCP configuration at `/tmp/.copilot/mcp-config.json`
- Uses "local" type for stdio-based MCP servers (Copilot CLI convention)
- Supports HTTP-based MCP servers for distributed tool access
- Provides built-in GitHub tools without additional MCP configuration

### Installation and Setup

The Copilot engine handles installation automatically:
1. Sets up Node.js 22 environment
2. Installs `@github/copilot` CLI globally via npm
3. Configures authentication using `COPILOT_CLI_TOKEN`
4. Sets up MCP server configurations
5. Creates necessary directory structures

## Engine Selection Guidelines

**Choose Claude when:**
- You need strong reasoning and analysis capabilities
- Working with complex code review or documentation tasks
- Performing multi-step reasoning workflows
- You want the most stable and well-tested engine

**Choose Copilot when:**
- You want conversational AI with GitHub integration
- Working with repository analysis and development workflows  
- Need access to Claude Sonnet 4 (default) or GPT-5 models
- Prefer GitHub's native AI tooling and ecosystem
- You're comfortable with experimental features

**Choose Codex when:**
- You need code-specific AI capabilities
- Working with specialized MCP server configurations
- Requiring custom TOML configuration for advanced scenarios
- You're comfortable with experimental features

**Choose Custom when:**
- You need deterministic, traditional GitHub Actions behavior
- Building hybrid workflows with some AI and some traditional steps
- You have specific requirements that AI engines can't meet
- Testing or prototyping workflow components

## Migration Between Engines

Switching between engines is straightforward - just change the `engine` field in your frontmatter:

```yaml
# From Claude to Copilot
engine: claude  # Old
engine: copilot # New

# From Codex to Copilot
engine: codex   # Old  
engine: copilot # New

# With configuration preservation
engine:
  id: copilot   # Changed from claude/codex
  model: gpt-5  # Add copilot-specific options (optional; defaults to claude-sonnet-4)
  version: latest
```

Note that engine-specific features (like `config` for Codex, `max-turns` for Claude, or `model` for Copilot) may not be available when switching engines.

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools Configuration](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/guides/security/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration