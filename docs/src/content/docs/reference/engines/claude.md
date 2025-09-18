---
title: Claude Engine
description: Complete guide to the Claude engine, including configuration, network isolation, version control, and features.
sidebar:
  order: 1
---

The Claude engine is the default and recommended AI engine for GitHub Agentic Workflows. It uses Anthropic's Claude Code CLI to interpret and execute natural language instructions with excellent reasoning and code analysis capabilities.

## Basic Configuration

### Simple Configuration

```yaml
engine: claude
```

### Extended Configuration

```yaml
engine:
  id: claude
  version: beta                     # Optional: CLI version (default: latest)
  model: claude-3-5-sonnet-20241022 # Optional: specific model
  max-turns: 5                      # Optional: max chat iterations per run
  env:                              # Optional: environment variables
    AWS_REGION: us-west-2
    DEBUG_MODE: "true"
    CUSTOM_API_ENDPOINT: https://api.example.com
```

## Frontmatter Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | `claude` | Engine identifier (required) |
| `version` | string | `latest` | Claude Code CLI version |
| `model` | string | `claude-3-5-sonnet-20241022` | Specific LLM model |
| `max-turns` | number | none | Maximum chat iterations per run |
| `env` | object | none | Custom environment variables |

### Environment Variables

The Claude engine supports custom environment variables that are passed to the execution environment:

```yaml
engine:
  id: claude
  env:
    # Override API key (uses secrets.ANTHROPIC_API_KEY by default)
    ANTHROPIC_API_KEY: ${{ secrets.CUSTOM_ANTHROPIC_KEY }}
    
    # Debug and development settings
    DEBUG_MODE: "true"
    DISABLE_TELEMETRY: "1"
    DISABLE_ERROR_REPORTING: "1"
    
    # Regional configuration
    AWS_REGION: us-west-2
    
    # Custom API endpoints
    CUSTOM_API_ENDPOINT: https://api.example.com
```

**Note**: The following environment variables are automatically set and should not be overridden:
- `GITHUB_AW_PROMPT`: Path to the prompt file
- `GITHUB_AW_SAFE_OUTPUTS`: Safe outputs configuration
- `GITHUB_AW_MAX_TURNS`: Max turns setting
- `DISABLE_BUG_COMMAND`: Security setting

## Version Control

The Claude engine supports version control through the `version` field, which controls the version of the `@anthropic-ai/claude-code` npm package used.

### Version Specification

```yaml
engine:
  id: claude
  version: latest    # Use latest version (default)
  
engine:
  id: claude
  version: beta      # Use beta version
  
engine:
  id: claude
  version: v1.2.3    # Use specific version
```

### Generated GitHub Actions Step

The version specification controls the npm package installation:

```yaml
# With version: latest (default)
run: npx @anthropic-ai/claude-code@latest --debug --verbose ...

# With version: beta
run: npx @anthropic-ai/claude-code@beta --debug --verbose ...

# With version: v1.2.3
run: npx @anthropic-ai/claude-code@v1.2.3 --debug --verbose ...
```

## Network Isolation

The Claude engine implements network isolation through Python-based hooks that restrict network access to allowed domains only.

### How Network Isolation Works

1. **Domain Allow-lists**: Configure allowed domains in workflow frontmatter
2. **Python Hook Script**: Generated hook intercepts network requests
3. **Settings File**: Claude settings.json enforces network restrictions
4. **Ecosystem Bundles**: Predefined domain sets for common tools

### Configuration

```yaml
# In workflow frontmatter
network:
  allowed:
    - "api.github.com"
    - "*.example.com"
    - "trusted.org"
```

### Ecosystem Domain Bundles

Use predefined domain bundles for common development ecosystems:

```yaml
network:
  allowed:
    - "bundle:node"      # npm, yarn, node.js domains
    - "bundle:python"    # pip, pypi, conda domains  
    - "bundle:github"    # GitHub-related domains
    - "bundle:containers" # Docker, container registries
    - "api.custom.com"   # Custom domain
```

Available bundles:
- `defaults`: SSL certificates, package managers
- `containers`: Docker Hub, GitHub Container Registry, Quay
- `dotnet`: NuGet, .NET domains
- `github`: GitHub CDN, raw content, LFS
- `go`: Go modules, proxy, sum database
- `java`: Maven, Gradle, Oracle JDK
- `node`: npm, yarn, Node.js, pnpm, bun, deno
- `python`: PyPI, conda, pip
- `ruby`: RubyGems, bundler
- `rust`: crates.io, rustup
- And more...

### Generated Network Hook

When network isolation is configured, the Claude engine generates:

1. **Settings Generation Step**: Creates Claude settings.json
```yaml
- name: Generate Claude Settings
  run: |
    mkdir -p /tmp/.claude
    cat > /tmp/.claude/settings.json << 'EOF'
    {
      "network": {
        "allowedDomains": ["api.github.com", "*.example.com"]
      }
    }
    EOF
```

2. **Network Hook Step**: Creates Python validation script
```yaml
- name: Setup Network Permissions Hook
  run: |
    cat > /tmp/network_hook.py << 'EOF'
    #!/usr/bin/env python3
    # Network permissions validator for Claude Code engine
    ALLOWED_DOMAINS = ["api.github.com", "*.example.com"]
    # ... validation logic ...
    EOF
    chmod +x /tmp/network_hook.py
```

3. **Execution with Settings**: Claude Code is invoked with settings
```bash
npx @anthropic-ai/claude-code@latest --settings /tmp/.claude/settings.json ...
```

### Network Hook Implementation

The network hook is a Python script that:

- **Domain Extraction**: Parses URLs and search queries to extract domains
- **Wildcard Matching**: Supports patterns like `*.example.com`
- **Validation Logic**: Allows/blocks requests based on domain allow-list
- **Logging**: Records network access attempts for debugging

## Features

### Core Capabilities

- **Excellent Reasoning**: Strong analytical and problem-solving capabilities
- **Code Analysis**: Deep understanding of code structure and patterns
- **MCP Tool Support**: Full Model Context Protocol integration
- **Tool Allow-listing**: Precise control over available tools

### Supported Features

| Feature | Supported | Description |
|---------|-----------|-------------|
| Max Turns | ✅ | Cost control through turn limiting |
| Tools Whitelist | ✅ | MCP tool allow-listing |
| HTTP Transport | ✅ | Both stdio and HTTP MCP transport |
| Network Isolation | ✅ | Domain-based access control |
| Custom Environment | ✅ | Environment variable customization |
| Version Control | ✅ | CLI version specification |

### MCP Server Integration

The Claude engine generates `mcp-servers.json` configuration for tool integration:

```json
{
  "mcpServers": {
    "github": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "--network=host", "..."],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "..."
      }
    },
    "safe-outputs": {
      "command": "npx",
      "args": ["@github/safe-outputs-mcp-server@latest"],
      "transport": "stdio"
    }
  }
}
```

## Security Considerations

### Default Security Settings

The Claude engine automatically sets secure defaults:

```yaml
env:
  DISABLE_TELEMETRY: "1"           # Disable usage telemetry
  DISABLE_ERROR_REPORTING: "1"    # Disable error reporting
  DISABLE_BUG_COMMAND: "1"        # Disable bug reporting command
```

### API Key Management

- Uses `secrets.ANTHROPIC_API_KEY` by default
- Can be overridden with custom secret via `env` configuration
- Never logged or exposed in workflow outputs

### Network Security

- Network isolation prevents unauthorized external requests
- Domain allow-lists provide fine-grained access control
- Ecosystem bundles reduce configuration while maintaining security

## Troubleshooting

### Common Issues

**Network Access Denied**
```
Error: Network request to unauthorized domain blocked
```
Solution: Add required domains to network allow-list or use appropriate ecosystem bundle.

**Version Not Found**
```
Error: Package @anthropic-ai/claude-code@v999 not found
```
Solution: Verify the version exists in npm registry or use `latest` or `beta`.

**API Key Missing**
```
Error: ANTHROPIC_API_KEY environment variable not set
```
Solution: Ensure `secrets.ANTHROPIC_API_KEY` is configured in repository secrets.

### Debug Configuration

Enable verbose logging for troubleshooting:

```yaml
engine:
  id: claude
  env:
    DEBUG_MODE: "true"
```

The Claude engine automatically includes:
- `--debug` flag for detailed execution logging
- `--verbose` flag for enhanced output
- `--output-format json` for structured responses

## Related Documentation

- [Network Configuration](/gh-aw/reference/network/) - Network isolation and domain management
- [Tools Configuration](/gh-aw/reference/tools/) - MCP servers and tool configuration
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output validation and security
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Complete configuration options

## External Links

- [Anthropic Claude Code CLI](https://github.com/anthropics/claude-code) - Official Claude Code repository
- [Model Context Protocol](https://spec.modelcontextprotocol.io/) - MCP specification
- [Anthropic API Documentation](https://docs.anthropic.com/) - Claude API reference