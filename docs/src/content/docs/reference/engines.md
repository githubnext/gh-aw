---
title: AI Engines
description: Complete guide to AI engines available in GitHub Agentic Workflows, including Claude, Codex, and custom engines with their specific configuration options.
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

[GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) 

```yaml
engine: copilot
```

#### Secrets

- `GITHUB_COPILOT_CLI_TOKEN` secret is required for authentication.

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

Both Claude and Codex engines support custom error pattern recognition for enhanced log validation:

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

## Engine Selection Guidelines

**Choose Claude when:**
- You need strong reasoning and analysis capabilities
- Working with complex code review or documentation tasks
- Performing multi-step reasoning workflows
- You want the most stable and well-tested engine

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
# From Claude to Codex
engine: claude  # Old
engine: codex   # New

# With configuration preservation
engine:
  id: codex     # Changed from claude
  model: gpt-4  # Add codex-specific options
  config: |     # Codex-only feature
    [custom]
    setting = "value"
```

Note that engine-specific features (like `config` for Codex or `max-turns` for Claude) may not be available when switching engines.

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools Configuration](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/guides/security/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration