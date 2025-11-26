---
title: AI Engines
description: Complete guide to AI engines (coding agents) usable with GitHub Agentic Workflows, including Copilot and custom engines with their specific configuration options.
sidebar:
  order: 600
---

GitHub Agentic Workflows support multiple AI engines (coding agents) to interpret and execute natural language instructions. Each engine has unique capabilities and configuration options.

:::note[Experimental Engines]
Claude and Codex engines are available but marked as experimental. They are not documented here but can still be used by setting `engine: claude` or `engine: codex` in your workflow frontmatter. For production workflows, we recommend using the GitHub Copilot CLI engine.
:::

### GitHub Copilot CLI

GitHub Copilot is the default and recommended AI engine for most workflows. The [GitHub Copilot CLI](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli) provides MCP server support and is designed for conversational AI workflows.

```yaml wrap
engine: copilot
```

#### Extended Configuration

```yaml wrap
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

- **`COPILOT_GITHUB_TOKEN`**: GitHub Personal Access Token (PAT) with "Copilot Requests" permission
- **`GH_AW_GITHUB_TOKEN`** (optional): Required for [GitHub Tools Remote Mode](/gh-aw/reference/tools/#modes-and-restrictions)

#### Authenticating with a Personal Access Token (PAT)

To use the Copilot engine, you need a fine-grained Personal Access Token with the "Copilot Requests" permission enabled:

1. Visit <https://github.com/settings/personal-access-tokens/new>
2. Under "Resource owner", select your user account (not an organization, see note below).
3. Under "Repository access," select "Public repositories"
4. Under "Permissions," click "Add permissions" and select "Copilot Requests". If you are not finding this option, review steps 2 and 3.
5. Generate your token
6. Add the token to your repository secrets as `COPILOT_GITHUB_TOKEN`:

```bash wrap
gh secret set COPILOT_GITHUB_TOKEN -a actions --body "<your-github-pat>"
```

:::note[Backward Compatibility]
The legacy secret names `COPILOT_CLI_TOKEN`, `GH_AW_COPILOT_TOKEN`, and `GH_AW_GITHUB_TOKEN` are still supported for backward compatibility, but `COPILOT_GITHUB_TOKEN` is now the recommended name for Copilot operations.
:::

For GitHub Tools Remote Mode, also configure:

```bash wrap
gh secret set GH_AW_GITHUB_TOKEN -a actions --body "<your-github-pat>"
```

For more information about GitHub Copilot CLI authentication, see the [official documentation](https://github.com/github/copilot-cli?tab=readme-ov-file#authenticate-with-a-personal-access-token-pat).

:::note
The Copilot engine does not have built-in `web-search` support. You can add web search capabilities using third-party MCP servers. See the [Using Web Search](/gh-aw/guides/web-search/) for available options and setup instructions.
:::

#### Network Permissions

The Copilot engine supports network access control through the `network:` configuration at the workflow level. When network permissions are configured, you can enable AWF (Agent Workflow Firewall) to enforce domain-based access controls. AWF is sourced from [github.com/githubnext/gh-aw-firewall](https://github.com/githubnext/gh-aw-firewall).

Enable network permissions and firewall in your workflow:

```yaml wrap
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

```yaml wrap
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

```yaml wrap
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

# Disable firewall using boolean
network:
  firewall: false

# Disable firewall using string (equivalent to false)
network:
  firewall: "disable"

# Custom configuration with version and arguments
network:
  firewall:
    version: "v0.1.0"
    log-level: debug
    args: ["--verbose"]
```

### Disabling the Firewall

To disable the firewall for any engine that supports it, set `firewall: false` in the `network` configuration. When disabling the firewall while also specifying `network.allowed` domains, you must set `strict: false` to avoid compilation errors:

```yaml wrap
strict: false
network:
  allowed:
    - defaults
    - python
    - "api.example.com"
  firewall: false
```

:::caution
When `network.allowed` domains are specified, disabling the firewall triggers:
- A **warning** in normal mode (compilation succeeds)
- An **error** in strict mode (compilation fails)

Set `strict: false` explicitly if you need to disable the firewall while using domain allowlists.
:::

See the [Network Permissions](/gh-aw/reference/network/) documentation for details on configuring allowed domains and ecosystem identifiers.

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

## Engine Command-Line Arguments

All engines support custom command-line arguments through the `args` field, injected before the prompt:

```yaml wrap
engine:
  id: copilot
  args: ["--add-dir", "/workspace", "--verbose"]
```

Arguments are added in order and placed before the `--prompt` flag. Common uses include adding directories (`--add-dir`), enabling verbose logging (`--verbose`, `--debug`), and passing engine-specific flags. Consult the specific engine's CLI documentation for available flags.

## Engine Error Patterns

All engines support custom error pattern recognition for enhanced log validation:

```yaml wrap
engine:
  id: copilot
  error_patterns:
    - pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)"
      level_group: 2
      message_group: 3
      description: "Custom error format with timestamp"
```

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
