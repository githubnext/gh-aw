---
title: Engine Feature Compatibility
description: Comprehensive matrix showing which features are supported by each AI engine in GitHub Agentic Workflows.
sidebar:
  order: 605
---

This page provides a complete feature compatibility matrix for all AI engines supported by GitHub Agentic Workflows. Use this reference to understand which features are available for your chosen engine.

## Supported Engines

GitHub Agentic Workflows supports four engines:

- **Copilot**: GitHub Copilot CLI (default, recommended)
- **Claude**: Anthropic Claude Code CLI
- **Codex**: OpenAI Codex CLI (experimental)
- **Custom**: User-defined GitHub Actions steps

## Transport Types

MCP (Model Context Protocol) servers can use different transport mechanisms for communication:

| Transport | Copilot | Claude | Codex | Custom |
|-----------|---------|--------|-------|--------|
| **stdio** (local commands) | ✅ | ✅ | ✅ | ✅ |
| **HTTP** (remote servers) | ✅ | ✅ | ✅ | ❌ |

**Notes:**
- **stdio**: All engines support local MCP servers via command execution
- **HTTP**: Copilot, Claude, and Codex support HTTP-based remote MCP servers
- **Custom**: Only supports MCP configuration for context; actual MCP server execution depends on custom step implementation

## Engine Configuration Features

Configuration options available in the `engine:` frontmatter field:

| Feature | Copilot | Claude | Codex | Custom | Description |
|---------|---------|--------|-------|--------|-------------|
| **model** | ✅ | ✅ | ✅ | ❌ | AI model selection |
| **version** | ✅ | ✅ | ✅ | ❌ | CLI version to install |
| **args** | ✅ | ✅ | ✅ | ✅ | Custom command-line arguments |
| **max-turns** | ❌ | ✅ | ❌ | ✅ | Maximum chat iterations per run |
| **max-concurrency** | ✅ | ✅ | ✅ | ✅ | Maximum concurrent workflows |
| **env** | ✅ | ✅ | ✅ | ✅ | Custom environment variables |
| **steps** | ❌ | ❌ | ❌ | ✅ | Custom GitHub Actions steps |

**Usage Examples:**

```yaml wrap
# Copilot with model
engine:
  id: copilot
  model: gpt-5

# Claude with max-turns
engine:
  id: claude
  max-turns: 10

# Custom with steps
engine:
  id: custom
  steps:
    - name: Run tests
      run: npm test
```

## Tool Support

### Built-in Tool Capabilities

Support for tools without requiring MCP server configuration:

| Tool | Copilot | Claude | Codex | Custom | Description |
|------|---------|--------|-------|--------|-------------|
| **web-fetch** | ❌ | ✅ | ❌ | ❌ | Fetch content from URLs |
| **web-search** | ❌ | ✅ | ✅ | ❌ | Search the web for information |
| **Tool Allowlisting** | ✅ | ✅ | ✅ | ❌ | Fine-grained MCP tool permissions |

**Notes:**
- Copilot requires third-party MCP servers for web-fetch and web-search (see [Web Search Guide](/gh-aw/guides/web-search/))
- Claude has native WebFetch and WebSearch support
- Codex has native web-search support
- Custom engine inherits tool availability from custom step implementations

### MCP Tool Configuration

All engines support MCP server configuration through the `mcp-servers:` field:

| Feature | Copilot | Claude | Codex | Custom |
|---------|---------|--------|-------|--------|
| **Local MCP Servers** | ✅ | ✅ | ✅ | ⚠️ |
| **Remote MCP Servers** | ✅ | ✅ | ✅ | ❌ |
| **Tool Allowlisting** | ✅ | ✅ | ✅ | ❌ |

**Legend:**
- ⚠️ Custom engine can access MCP config but requires custom steps to interact with servers

### Standard Tools

Standard tools available through the `tools:` field:

| Tool | Copilot | Claude | Codex | Custom | Description |
|------|---------|--------|-------|--------|-------------|
| **github** | ✅ | ✅ | ✅ | ⚠️ | GitHub API operations |
| **edit** | ✅ | ✅ | ✅ | ⚠️ | File editing capabilities |
| **bash** | ✅ | ✅ | ✅ | ⚠️ | Shell command execution |
| **playwright** | ✅ | ✅ | ✅ | ⚠️ | Browser automation |
| **agentic-workflows** | ✅ | ✅ | ✅ | ⚠️ | Workflow introspection |
| **cache-memory** | ✅ | ✅ | ❌ | ❌ | Persistent memory storage |

**Notes:**
- Custom engine can configure tools but requires custom steps to utilize them
- cache-memory tool uses MCP server configuration and requires MCP support

## Network Features

Network access control and security features:

| Feature | Copilot | Claude | Codex | Custom | Description |
|---------|---------|--------|-------|--------|-------------|
| **Network Permissions** | ✅ | ✅ | ⚠️ | ❌ | Domain-based access control |
| **Firewall (AWF)** | ✅ | ❌ | ❌ | ❌ | Active firewall enforcement |
| **Network Hooks** | ❌ | ✅ | ❌ | ❌ | Hook-based enforcement |

**Network Configuration Example:**

```yaml wrap
# Copilot with firewall
engine: copilot
network:
  firewall: true
  allowed:
    - defaults
    - python

# Claude with network hooks
engine: claude
network:
  allowed:
    - defaults
    - "api.example.com"
```

**Enforcement Mechanisms:**
- **Copilot**: Uses AWF (Agent Workflow Firewall) for active network filtering
- **Claude**: Uses network hooks for domain validation
- **Codex**: Supports network configuration but enforcement is limited
- **Custom**: No built-in network enforcement

## Safe Outputs

Safe output features for creating GitHub resources:

| Feature | Copilot | Claude | Codex | Custom |
|---------|---------|--------|-------|--------|
| **create-issue** | ✅ | ✅ | ✅ | ✅ |
| **create-discussion** | ✅ | ✅ | ✅ | ✅ |
| **add-comment** | ✅ | ✅ | ✅ | ✅ |
| **create-pull-request** | ✅ | ✅ | ✅ | ✅ |
| **create-pull-request-review-comment** | ✅ | ✅ | ✅ | ✅ |
| **update-issue** | ✅ | ✅ | ✅ | ✅ |
| **create-agent-task** | ✅ | ✅ | ✅ | ✅ |
| **add-labels** | ✅ | ✅ | ✅ | ✅ |
| **create-code-scanning-alert** | ✅ | ✅ | ✅ | ✅ |

**Note**: All engines support safe outputs as they are processed in separate workflow jobs after the main engine execution completes.

## Custom Agent Support

Support for custom agent files (imported via `imports:` field):

| Feature | Copilot | Claude | Codex | Custom |
|---------|---------|--------|-------|--------|
| **Agent File Import** | ✅ | ✅ | ✅ | ❌ |
| **Agent Frontmatter** | ✅ | ⚠️ | ⚠️ | ❌ |
| **Agent Markdown Content** | ✅ | ✅ | ✅ | ❌ |

**Implementation:**
- **Copilot**: Uses `--agent` flag with agent identifier
- **Claude/Codex**: Prepends agent markdown content to workflow prompt
- **Custom**: No built-in agent support (can implement in custom steps)

**Legend:**
- ⚠️ Only markdown content is used; frontmatter is parsed but tool configurations are not applied

## Authentication Requirements

Required secrets for each engine:

| Engine | Primary Secret | Alternative Secret | Notes |
|--------|---------------|-------------------|-------|
| **Copilot** | `COPILOT_GITHUB_TOKEN` | `COPILOT_CLI_TOKEN` | Fine-grained PAT with "Copilot Requests" permission |
| **Claude** | `CLAUDE_CODE_OAUTH_TOKEN` | `ANTHROPIC_API_KEY` | Claude Code OAuth or Anthropic API key |
| **Codex** | `CODEX_API_KEY` | `OPENAI_API_KEY` | OpenAI API key |
| **Custom** | None | - | Uses `GITHUB_TOKEN` unless custom steps require specific secrets |

## Experimental Features

Features marked as experimental may change in future versions:

| Feature | Copilot | Claude | Codex | Custom |
|---------|---------|--------|-------|--------|
| **Engine Status** | Stable | Stable | ⚠️ Experimental | Stable |

## Feature Selection Guide

### Choose Copilot if you need:
- Default recommended engine
- AWS firewall support
- GitHub Copilot ecosystem integration
- Active network access control

### Choose Claude if you need:
- Built-in web-fetch and web-search
- max-turns control
- Advanced tool allowlisting
- Network hooks for enforcement

### Choose Codex if you need:
- OpenAI ecosystem integration
- Built-in web-search
- Experimental features

### Choose Custom if you need:
- Complete control over execution steps
- Custom tool integration
- Non-AI workflow automation
- GitHub Actions-based workflows

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Detailed engine configuration
- [Tools Reference](/gh-aw/reference/tools/) - Tool configuration options
- [Network Access](/gh-aw/reference/network/) - Network permission configuration
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output configuration
- [MCP Servers Guide](/gh-aw/guides/mcps/) - Using MCP servers

## Version Information

This compatibility matrix is accurate as of the current version of GitHub Agentic Workflows. Features may be added or changed in future releases. Check the [status page](/gh-aw/status/) for the latest information.

:::tip
When choosing an engine, consider your specific requirements for tools, network access, and configuration options. The default Copilot engine is recommended for most use cases.
:::
