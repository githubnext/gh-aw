---
title: AI Engines
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Claude, Copilot, Codex, and custom engines with their specific configuration options.
sidebar:
  order: 350
---

GitHub Agentic Workflows support multiple AI engines (coding agents) to interpret and execute natural language instructions. Each engine has unique capabilities and configuration options.

### GitHub Copilot (Default)

GitHub Copilot is the default and recommended AI engine for most workflows. The [GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) provides MCP server support and is designed for conversational AI workflows with access to GitHub repositories and development tools.

```yaml
engine: copilot
```

**Extended configuration:**
```yaml
engine:
  id: copilot
  version: latest
  model: gpt-5                          # Optional: uses claude-sonnet-4 by default
```

**Copilot-specific fields:**
- **`model`** (optional): AI model to use (`gpt-5` or defaults to `claude-sonnet-4`)
- **`version`** (optional): Version of the GitHub Copilot CLI to install (defaults to `latest`)

:::note
The Copilot engine does not have built-in `web-search` support. You can add web search capabilities using third-party MCP servers. See the [Web Search with MCP guide](/gh-aw/guides/web-search/) for available options and setup instructions.
:::

**Environment Variables:**
- **`COPILOT_MODEL`**: Alternative way to set the model (e.g., `gpt-5`)

**Secrets:**

- **`COPILOT_CLI_TOKEN`** secret is required for authentication.

Please [create a GitHub Personal Access Token (PAT) for an account with a GitHub Copilot subscription](https://github.com/settings/tokens) and add this as a repository secret:

```bash
gh secret set COPILOT_CLI_TOKEN -a actions --body "<your-github-pat>"
```

- **`GITHUB_MCP_TOKEN`** secret (optional) is required when using remote mode for GitHub tools.

If you use `mode: remote` for GitHub tools (for faster startup without Docker), you'll need a separate GitHub Personal Access Token:

```bash
gh secret set GITHUB_MCP_TOKEN -a actions --body "<your-github-pat>"
```

See [GitHub Tools - Remote Mode](/gh-aw/reference/tools/#github-remote-mode) for more details.

### Anthropic Claude Code

Claude Code excels at reasoning, code analysis, and understanding complex contexts. It provides robust capabilities for agentic workflows.

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

**Secrets:**

- **`ANTHROPIC_API_KEY`** secret is required for authentication.

Use this to set the secret for your repo:

```bash
gh secret set ANTHROPIC_API_KEY -a actions --body "<your-anthropic-api-key>"
```

- **`GITHUB_MCP_TOKEN`** secret (optional) is required when using remote mode for GitHub tools.

If you use `mode: remote` for GitHub tools (for faster startup without Docker), you'll need a GitHub Personal Access Token:

```bash
gh secret set GITHUB_MCP_TOKEN -a actions --body "<your-github-pat>"
```

See [GitHub Tools - Remote Mode](/gh-aw/reference/tools/#github-remote-mode) for more details.

### OpenAI Codex

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
    CODEX_API_KEY: ${{ secrets.CODEX_API_KEY_CI }}
  config: |
    [custom_section]
    key1 = "value1"
    key2 = "value2"
    
    [server_settings]
    timeout = 60
    retries = 3
```

**Codex-specific fields:**
- **`user-agent`** (optional): Custom user agent string for GitHub MCP server configuration
- **`config`** (optional): Additional TOML configuration text appended to generated config.toml

**Secrets:**

- **`OPENAI_API_KEY`** secret is required for authentication.

Use this to set the secret for your repo:

```bash
gh secret set OPENAI_API_KEY -a actions --body "<your-openai-api-key>"
```

The Codex engine supports additional customization through the `config` field, which allows you to append raw TOML configuration to the generated `config.toml` file.

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
    - name: Install dependencies
      run: npm ci
```

**Features:**
- Execute user-defined GitHub Actions steps
- No AI interpretation - direct step execution
- Useful for deterministic workflows or hybrid approaches

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

Note that engine-specific features may not be available when switching engines.

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete configuration reference
- [Tools Configuration](/gh-aw/reference/tools/) - Available tools and MCP servers
- [Security Guide](/gh-aw/guides/security/) - Security considerations for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration