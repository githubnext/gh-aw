---
title: AI Engines
description: Overview of AI engines available in GitHub Agentic Workflows, including Claude, Codex, and custom engines with links to detailed configuration guides.
sidebar:
  order: 1
---

GitHub Agentic Workflows support multiple engines to interpret and execute natural language instructions or run custom GitHub Actions steps. Each engine has unique capabilities, configuration options, and use cases.

## Quick Engine Reference

| Engine | Type | Status | Best For | Documentation |
|--------|------|--------|----------|---------------|
| [Claude](/gh-aw/reference/engines/claude/) | AI | Stable ✅ | Reasoning, code analysis, general workflows | [View Guide →](/gh-aw/reference/engines/claude/) |
| [Codex](/gh-aw/reference/engines/codex/) | AI | Experimental ⚠️ | Code generation, specialized integrations | [View Guide →](/gh-aw/reference/engines/codex/) |
| [Custom](/gh-aw/reference/engines/custom/) | Traditional | Stable ✅ | Deterministic steps, hybrid workflows | [View Guide →](/gh-aw/reference/engines/custom/) |

## Engine Types

### AI Engines

AI engines interpret natural language instructions and execute them using various tools and capabilities:

- **Claude**: Uses Anthropic's Claude Code CLI with excellent reasoning capabilities
- **Codex**: Uses OpenAI Codex CLI with specialized code generation features

### Traditional Engines

Traditional engines execute predefined GitHub Actions steps without AI interpretation:

- **Custom**: Executes user-defined GitHub Actions steps for deterministic workflows

## Quick Start Examples

### Claude Engine (Recommended)
```yaml
---
engine: claude
---

# Analyze Code Quality

Review the code in this repository and suggest improvements for performance and maintainability.
```

### Codex Engine
```yaml
---
engine: codex
---

# Generate API Tests

Create comprehensive test cases for the REST API endpoints in this project.
```

### Custom Engine
```yaml
---
engine:
  id: custom
  steps:
    - name: Run tests
      run: npm test
    - name: Deploy
      uses: actions/deploy@v1
---

# Traditional GitHub Actions Steps

This workflow runs predefined steps without AI interpretation.
```

## Available Engines

### [Claude Engine](/gh-aw/reference/engines/claude/) (Default)

The recommended AI engine for most workflows with excellent reasoning and code analysis capabilities.

- **Type**: AI Engine
- **Status**: Stable ✅
- **Network Isolation**: ✅ Python hooks with domain allow-lists
- **Version Control**: ✅ npm package versions
- **Max Turns**: ✅ Cost control support

**Quick Config:**
```yaml
engine:
  id: claude
  version: latest
  max-turns: 5
```

[**→ View Complete Claude Documentation**](/gh-aw/reference/engines/claude/)

### [Codex Engine](/gh-aw/reference/engines/codex/) (Experimental)

AI engine optimized for code generation and specialized development tasks.

- **Type**: AI Engine  
- **Status**: Experimental ⚠️
- **Network Isolation**: ⚠️ Limited to specific tools
- **Version Control**: ✅ npm package versions
- **Custom Config**: ✅ TOML configuration support

**Quick Config:**
```yaml
engine:
  id: codex
  model: gpt-4
  config: |
    [custom]
    setting = "value"
```

[**→ View Complete Codex Documentation**](/gh-aw/reference/engines/codex/)

### [Custom Engine](/gh-aw/reference/engines/custom/)

Execute traditional GitHub Actions steps without AI interpretation.

- **Type**: Traditional Engine
- **Status**: Stable ✅  
- **Network Isolation**: ❌ Manual implementation required
- **Version Control**: ✅ Action version pinning
- **Direct Control**: ✅ Exact step execution

**Quick Config:**
```yaml
engine:
  id: custom
  steps:
    - name: Build
      run: make build
    - uses: actions/deploy@v1
```

[**→ View Complete Custom Documentation**](/gh-aw/reference/engines/custom/)**

## Feature Comparison

| Feature | Claude | Codex | Custom |
|---------|--------|--------|--------|
| **AI Interpretation** | ✅ | ✅ | ❌ |
| **Network Isolation** | ✅ Full | ⚠️ Limited | ❌ Manual |
| **Version Control** | ✅ npm | ✅ npm | ✅ Actions |
| **Max Turns** | ✅ | ❌ | ⚠️ No effect |
| **Tools Whitelist** | ✅ | ✅ | ❌ |
| **HTTP Transport** | ✅ | ❌ | ❌ |
| **Custom Config** | ❌ | ✅ TOML | ❌ |
| **Environment Variables** | ✅ | ✅ | ✅ |
| **Experimental** | ❌ | ✅ | ❌ |

## Common Configuration

### Environment Variables

All engines support custom environment variables through the `env` field:

```yaml
engine:
  id: claude  # or codex, custom
  env:
    DEBUG_MODE: "true"
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
```

### Version Control

Engines support version specification for their underlying tools:

```yaml
# Claude: Controls @anthropic-ai/claude-code version
engine:
  id: claude
  version: latest  # or beta, v1.2.3

# Codex: Controls @openai/codex version  
engine:
  id: codex
  version: latest  # or beta, v2.1.0

# Custom: Control action versions in steps
engine:
  id: custom
  steps:
    - uses: actions/setup-node@v4  # Pin to v4
```

## Engine Selection Guidelines

**Choose [Claude](/gh-aw/reference/engines/claude/) when:**
- You need strong reasoning and analysis capabilities
- Working with complex code review or documentation tasks
- Performing multi-step reasoning workflows
- You want the most stable and well-tested engine
- Network security is important

**Choose [Codex](/gh-aw/reference/engines/codex/) when:**
- You need code-specific AI capabilities
- Working with specialized MCP server configurations
- Requiring custom TOML configuration for advanced scenarios
- You're comfortable with experimental features

**Choose [Custom](/gh-aw/reference/engines/custom/) when:**
- You need deterministic, traditional GitHub Actions behavior
- Building hybrid workflows with some AI and some traditional steps
- You have specific requirements that AI engines can't meet
- Testing or prototyping workflow components
- Maximum control over execution is required

## Migration Between Engines

Switching between engines requires updating the `engine` field and potentially adjusting configuration:

### Claude ↔ Codex Migration

```yaml
# From Claude to Codex
engine: claude                    # Remove
engine: codex                     # Add
# Plus: change ANTHROPIC_API_KEY to OPENAI_API_KEY

# From Codex to Claude  
engine: codex                     # Remove
engine: claude                    # Add
# Plus: change OPENAI_API_KEY to ANTHROPIC_API_KEY
```

### AI → Custom Migration

```yaml
# From AI engine
engine: claude
# Instruction: "Run tests and deploy"

# To Custom engine
engine:
  id: custom
  steps:
    - name: Run tests
      run: npm test
    - name: Deploy
      run: ./deploy.sh
```

### Custom → AI Migration

```yaml
# From Custom engine
engine:
  id: custom
  steps:
    - name: Complex deployment
      run: ./complex-deploy.sh

# To AI engine
engine: claude
# Instruction: "Deploy the application using the deployment script"
```

## Network Security

### Claude Engine Network Isolation

- **Implementation**: Python hooks with domain validation
- **Configuration**: `network.allowed` with domain patterns
- **Ecosystem Bundles**: Predefined domain sets (`bundle:node`, `bundle:python`, etc.)
- **Coverage**: System-wide network interception

### Codex Engine Network Limitations  

- **Implementation**: Tool-specific restrictions
- **Configuration**: Per-tool domain settings
- **Coverage**: Limited to specific tools (e.g., Playwright)
- **Manual Setup**: Requires explicit tool configuration

### Custom Engine Network Control

- **Implementation**: Manual setup in steps
- **Configuration**: User-defined network restrictions
- **Coverage**: Depends on user implementation
- **Examples**: iptables, proxy configuration

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools Configuration](/gh-aw/reference/tools/) - Available tools and MCP servers  
- [Network Configuration](/gh-aw/reference/network/) - Network isolation and domain management
- [Security Guide](/gh-aw/guides/security/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration