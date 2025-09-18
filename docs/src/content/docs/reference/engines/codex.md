---
title: Codex Engine
description: Complete guide to the Codex engine, including configuration, network isolation, version control, and features.
sidebar:
  order: 2
---

The Codex engine is an experimental AI engine for GitHub Agentic Workflows that uses OpenAI Codex CLI with MCP server support. It's designed for code-focused tasks and advanced integration scenarios.

:::caution[Experimental Feature]
The Codex engine is experimental and may have breaking changes. Use with caution in production workflows.
:::

## Basic Configuration

### Simple Configuration

```yaml
engine: codex
```

### Extended Configuration

```yaml
engine:
  id: codex
  version: latest                   # Optional: CLI version (default: latest)
  model: gpt-4                      # Optional: specific model
  user-agent: custom-workflow-name  # Optional: GitHub MCP user agent
  env:                              # Optional: environment variables
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY_CI }}
    DEBUG_MODE: "true"
  config: |                         # Optional: custom TOML configuration
    [custom_section]
    key1 = "value1"
    key2 = "value2"
    
    [server_settings]
    timeout = 60
    retries = 3
```

## Frontmatter Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | `codex` | Engine identifier (required) |
| `version` | string | `latest` | Codex CLI version |
| `model` | string | Default model | Specific LLM model to use |
| `user-agent` | string | Workflow name | GitHub MCP server user agent |
| `env` | object | none | Custom environment variables |
| `config` | string | none | Additional TOML configuration |

### Environment Variables

The Codex engine supports custom environment variables:

```yaml
engine:
  id: codex
  env:
    # OpenAI API configuration (required)
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    
    # Debug and development settings
    DEBUG_MODE: "true"
    CODEX_LOG_LEVEL: "debug"
    
    # Custom API endpoints
    OPENAI_BASE_URL: https://api.openai.com/v1
    
    # GitHub integration
    GITHUB_PERSONAL_ACCESS_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Note**: The following environment variables are automatically set:
- `GITHUB_AW_PROMPT`: Path to the prompt file
- `GITHUB_STEP_SUMMARY`: GitHub Actions step summary
- `OPENAI_API_KEY`: Required OpenAI API key

### Custom TOML Configuration

The Codex engine supports additional TOML configuration through the `config` field:

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
    
    # MCP server overrides
    [mcp_servers.custom]
    command = "custom-mcp-server"
    args = ["--mode", "production"]
```

## Version Control

The Codex engine supports version control through the `version` field, which controls the version of the `@openai/codex` npm package.

### Version Specification

```yaml
engine:
  id: codex
  version: latest    # Use latest version (default)
  
engine:
  id: codex
  version: beta      # Use beta version
  
engine:
  id: codex
  version: v2.1.0    # Use specific version
```

### Generated Installation Steps

The version specification controls the npm installation:

```yaml
# With version: latest (default)
- name: Install Codex
  run: npm install -g @openai/codex

# With version: v2.1.0
- name: Install Codex
  run: npm install -g @openai/codex@v2.1.0
```

### Node.js Setup

The Codex engine automatically sets up Node.js 24:

```yaml
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '24'
```

## Network Isolation

The Codex engine implements network isolation through domain restrictions for Playwright and other tools, but does not use the same Python hook system as Claude.

### How Network Isolation Works

1. **Tool-level Restrictions**: Network access is controlled at the tool level
2. **Playwright Domains**: Allowed domains for browser automation
3. **MCP Configuration**: Network settings in config.toml
4. **No Global Hooks**: Unlike Claude, no global network interception

### Configuration

Network isolation is configured through tool-specific settings:

```yaml
# In workflow frontmatter
tools:
  playwright:
    allowed_domains:
      - "example.com"
      - "*.trusted.com"
      - "api.github.com"

# Or using ecosystem bundles
tools:
  playwright:
    allowed_domains:
      - "bundle:node"
      - "bundle:github"
      - "custom.domain.com"
```

### Playwright Network Isolation

For Playwright tools, the Codex engine generates MCP configuration:

```toml
[mcp_servers.playwright]
command = "npx"
args = [
  "@playwright/mcp@latest",
  "--allowed-origins",
  "example.com,*.trusted.com,api.github.com"
]
```

### Limited Network Isolation

The Codex engine has more limited network isolation compared to Claude:

- **Tool-specific**: Only certain tools support network restrictions
- **No Global Hooks**: No system-wide network interception
- **Manual Configuration**: Requires explicit tool configuration

## Features

### Core Capabilities

- **Code-focused AI**: Optimized for code generation and analysis
- **MCP Integration**: Full Model Context Protocol support
- **Custom Configuration**: Flexible TOML configuration system
- **Tool Integration**: Specialized MCP server configurations

### Supported Features

| Feature | Supported | Description |
|---------|-----------|-------------|
| Max Turns | ❌ | Not supported in Codex engine |
| Tools Whitelist | ✅ | MCP tool allow-listing |
| HTTP Transport | ❌ | Only stdio transport supported |
| Network Isolation | ⚠️ | Limited to specific tools |
| Custom Environment | ✅ | Environment variable customization |
| Version Control | ✅ | CLI version specification |
| Custom TOML Config | ✅ | Unique to Codex engine |

### MCP Server Integration

The Codex engine generates `config.toml` for MCP server configuration:

```toml
[history]
persistence = "none"

[mcp_servers.github]
user_agent = "workflow-name"
command = "docker"
args = ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghcr.io/github/github-mcp-server:sha-09deac4"]
env = { "GITHUB_PERSONAL_ACCESS_TOKEN" = "${{ secrets.GITHUB_TOKEN }}" }

[mcp_servers.safe_outputs]
command = "npx"
args = ["@github/safe-outputs-mcp-server@latest"]

# Custom configuration appended here
[custom_section]
key1 = "value1"
key2 = "value2"
```

### User Agent Configuration

The Codex engine supports custom user agent strings for GitHub MCP server:

```yaml
engine:
  id: codex
  user-agent: my-custom-workflow
```

This sets the `user_agent` field in the GitHub MCP server configuration.

## Execution Process

### Installation Phase

1. **Node.js Setup**: Install Node.js 24
2. **Codex Installation**: Install `@openai/codex` package with specified version
3. **Authentication**: Login to Codex with OpenAI API key

### Execution Phase

1. **Config Generation**: Create `config.toml` with MCP servers and custom configuration
2. **Codex Execution**: Run Codex CLI with generated configuration
3. **Log Capture**: Capture output for debugging and analysis

### Generated Execution Steps

```yaml
- name: Execute Codex
  run: |
    # Setup logging
    mkdir -p /tmp/aw-logs
    
    # Check Codex installation
    which codex
    codex --version
    
    # Authenticate with OpenAI
    codex login --api-key "$OPENAI_API_KEY"
    
    # Execute with instruction
    codex --full-auto exec "$INSTRUCTION" 2>&1 | tee /tmp/aw-logs/log.txt
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    GITHUB_STEP_SUMMARY: ${{ env.GITHUB_STEP_SUMMARY }}
```

## Security Considerations

### API Key Management

- Requires `secrets.OPENAI_API_KEY` for authentication
- Can be overridden with custom secret via `env` configuration
- Uses GitHub secrets for secure credential management

### Limited Network Security

The Codex engine has weaker network isolation compared to Claude:

- No global network hooks
- Tool-specific restrictions only
- Manual configuration required

### Configuration Security

- Custom TOML configuration is appended to generated config
- Validate TOML syntax to prevent injection
- Avoid overriding security-critical sections

## Troubleshooting

### Common Issues

**API Key Missing**
```
Error: OPENAI_API_KEY environment variable not set
```
Solution: Configure `secrets.OPENAI_API_KEY` in repository secrets.

**Version Not Found**
```
Error: Package @openai/codex@v999 not found
```
Solution: Verify version exists in npm registry or use `latest`.

**Invalid TOML Configuration**
```
Error: Failed to parse TOML configuration
```
Solution: Validate TOML syntax in the `config` field.

**Network Access Issues**
```
Error: Network request blocked
```
Solution: Configure allowed domains for specific tools like Playwright.

### Debug Configuration

Enable verbose logging:

```yaml
engine:
  id: codex
  env:
    DEBUG_MODE: "true"
    CODEX_LOG_LEVEL: "debug"
  config: |
    [logging]
    level = "debug"
    file = "/tmp/codex-debug.log"
```

### TOML Configuration Best Practices

1. **Validate Syntax**: Use a TOML validator before deployment
2. **Avoid Conflicts**: Don't override `[history]` or `[mcp_servers.*]` sections
3. **Use Comments**: Document custom configuration sections
4. **Test Thoroughly**: Validate configuration in development environment

## Migration from Claude

When migrating from Claude to Codex:

1. **Remove Max Turns**: Codex doesn't support `max-turns`
2. **Update API Key**: Change from `ANTHROPIC_API_KEY` to `OPENAI_API_KEY`
3. **Configure Network**: Set tool-specific network restrictions
4. **Add Custom Config**: Use `config` field for advanced settings

```yaml
# Claude configuration
engine:
  id: claude
  max-turns: 5
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}

# Equivalent Codex configuration
engine:
  id: codex
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  config: |
    [execution]
    mode = "controlled"  # Custom setting to limit iterations
```

## Related Documentation

- [Custom TOML Configuration](/gh-aw/reference/frontmatter/#codex-engine-custom-configuration) - Detailed TOML configuration guide
- [Tools Configuration](/gh-aw/reference/tools/) - MCP servers and tool setup
- [Network Configuration](/gh-aw/reference/network/) - Network isolation principles
- [Engine Migration](/gh-aw/reference/engines/#migration-between-engines) - Switching between engines

## External Links

- [OpenAI Codex Documentation](https://platform.openai.com/docs/guides/code) - Official Codex documentation
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference) - API documentation
- [Model Context Protocol](https://spec.modelcontextprotocol.io/) - MCP specification
- [TOML Specification](https://toml.io/en/) - TOML format documentation