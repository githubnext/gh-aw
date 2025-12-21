---
name: copilot-cli
description: GitHub Copilot CLI Integration
---


# GitHub Copilot CLI Integration

This file integrates with the GitHub Copilot CLI (`@github/copilot`) for agentic workflow execution. The Copilot CLI provides natural language processing capabilities and MCP (Model Context Protocol) server support.

## GitHub Copilot CLI Overview

The GitHub Copilot CLI is an experimental AI-powered command-line interface that can:
- Execute natural language prompts via `copilot --prompt "your instruction"`
- Support MCP servers for tool integration
- Generate code, documentation, and provide explanations
- Work with file directories and project contexts
- Integrate with GitHub API and repositories

## Installation and Setup

The CLI is installed via npm and requires authentication:

```bash
npm install -g @github/copilot
```

**Environment Variables:**
- `GITHUB_TOKEN` or `COPILOT_GITHUB_TOKEN`: GitHub token for authentication
- `XDG_CONFIG_HOME`: Configuration directory (defaults to `/tmp/gh-aw/.copilot/`)
- `XDG_STATE_HOME`: State/cache directory (defaults to `/tmp/gh-aw/.copilot/`)

**Note**: The `COPILOT_CLI_TOKEN` environment variable is no longer supported as of v0.26+. Use `COPILOT_GITHUB_TOKEN` instead.

## Core Command Structure

### Basic Usage
```bash
copilot --prompt "your natural language instruction"
```

### Advanced Options
```bash
copilot --add-dir /path/to/project \
        --log-level debug \
        --log-dir /tmp/gh-aw/logs \
        --model gpt-5 \
        --prompt "instruction"
```

**Key Parameters:**
- `--add-dir`: Add directory context to the prompt
- `--log-level`: Set logging verbosity (debug, info, warn, error)
- `--log-dir`: Directory for log output
- `--model`: Specify AI model (if supported)
- `--prompt`: Natural language instruction (required to avoid interactive mode)

## Tool Permission and Availability Control (v0.0.370+)

Copilot CLI v0.0.370 introduces a distinction between **tool availability** (what the model can see) and **tool permissions** (what requires approval):

### Tool Availability Flags

**`--available-tools [tools...]`** - Restricts which tools the model can see
- Only specified tools will be available to the model
- Disables all other tools
- Acts as an allowlist filter

**`--excluded-tools [tools...]`** - Hides specific tools from the model
- Specified tools will not be available to the model
- Other tools remain available
- Acts as a denylist filter

**Use cases:**
- Limit model to safe read-only operations
- Remove dangerous tools from model visibility
- Create specialized agents with restricted toolsets

### Tool Permission Flags

**`--allow-tool [tools...]`** - Pre-approves tools to run without confirmation
- Tools will execute without user prompts
- Required for non-interactive mode
- Does not expose tools filtered by availability flags

**`--deny-tool [tools...]`** - Denies permission for specific tools
- Tools will always be denied, even if allowed by other flags
- Takes precedence over `--allow-tool` and `--allow-all-tools`
- Useful for blocking dangerous operations while allowing others

**`--allow-all-tools`** - Pre-approves all available tools
- Required for non-interactive execution
- Applies only to tools not filtered by availability flags

### Flag Precedence and Interaction

1. **Availability filters are applied first**: `--available-tools` and `--excluded-tools` control what the model can see
2. **Permission checks are applied second**: `--allow-tool`, `--deny-tool`, and `--allow-all-tools` control approval prompts
3. **Denial takes precedence**: `--deny-tool` overrides any allow rules

**Example combinations:**

```bash
# Safe read-only agent: Model can only see read tools, all pre-approved
copilot --available-tools 'github(get_file_contents)' 'github(list_commits)' \
        --allow-all-tools \
        --prompt "Analyze the repository"

# Flexible agent with safety guardrails: Model sees all tools except dangerous ones
copilot --excluded-tools 'shell(rm:*)' 'shell(git push)' \
        --allow-all-tools \
        --prompt "Help me with the codebase"

# Granular control: Limit visibility and pre-approve specific operations
copilot --available-tools 'github' 'shell(git:*)' 'write' \
        --deny-tool 'shell(git push)' \
        --allow-tool 'github' 'shell(git:*)' 'write' \
        --prompt "Create a pull request"
```

### Tool Permission Patterns

Tool patterns follow the format `kind(argument)`:

- `shell(command:*?)` - Shell commands
  - `shell(echo)` - Specific command
  - `shell(git:*)` - All git commands
  - `shell` - All shell commands

- `write` - File creation and modification tools

- `<mcp-server-name>(tool-name?)` - MCP server tools
  - `github(get_file_contents)` - Specific tool
  - `github` - All tools from server

**Wildcard matching:**
- Use `:*` suffix for prefix matching
- `shell(git:*)` matches `git push`, `git commit`, etc.
- Wildcard matching applies to command stems, so `shell(git:*)` won't match `gitea`

### Migration from Pre-v0.0.370

**Before v0.0.370:**
- Only `--allow-tool` and `--deny-tool` were available
- These flags controlled both visibility and permissions

**After v0.0.370:**
- `--allow-tool`/`--deny-tool` now only control permissions (approval prompts)
- Use `--available-tools`/`--excluded-tools` to control model visibility
- **Backward compatible**: Existing workflows using `--allow-tool`/`--deny-tool` continue to work

**No action required** for existing workflows, but consider:
- Using availability flags for additional security (defense in depth)
- Restricting model visibility to only necessary tools
- Combining both approaches for maximum control

## MCP Server Configuration

Copilot CLI supports MCP servers via JSON configuration at `/tmp/gh-aw/.copilot/mcp-config.json`:

```json
{
  "mcpServers": {
    "github": {
      "type": "local",
      "command": "npx",
      "args": ["@github/github-mcp-server"]
    },
    "playwright": {
      "type": "local", 
      "command": "npx",
      "args": ["@playwright/mcp@latest", "--allowed-hosts", "example.com"]
    },
    "custom-server": {
      "type": "local",
      "command": "python",
      "args": ["-m", "my_server"],
      "env": {
        "API_KEY": "secret"
      }
    }
  }
}
```

**Server Types:**
- `local`: Local command execution (equivalent to `stdio` in other MCP configs)
- `http`: HTTP-based MCP server
- Built-in servers like GitHub are automatically available

## Log Parsing and Output

### Expected Log Format
Copilot CLI logs contain:
- Command execution traces
- Tool call information
- Code blocks with language annotations (```language)
- Error and warning messages
- Suggestions and responses

### Log Parsing Patterns
When parsing logs in `parse_copilot_log.cjs`:
- Look for command patterns: `copilot -p`, `github copilot`
- Extract code blocks between ``` markers
- Capture responses with `Suggestion:` or `Response:` prefixes
- Identify errors with `error:` and warnings with `warning:`
- Filter out timestamps and shell prompts

## Error Handling

### Common Error Patterns
- Authentication failures: Missing or invalid `GITHUB_TOKEN`
- MCP server connection issues
- Tool execution timeouts
- Invalid prompt formatting
- Directory permission issues

### Best Practices
- Always use `--prompt` parameter to avoid interactive blocking
- Set appropriate timeouts for long-running operations
- Validate MCP server configurations before execution
- Handle authentication errors gracefully
- Log detailed error information for debugging

## Integration with GitHub Agentic Workflows

### Engine Configuration
```yaml
engine: copilot
# or
engine:
  id: copilot
  version: latest
  model: gpt-5  # defaults to claude-sonnet-4 if not specified
```

### Tool Integration
- GitHub tools are built-in (don't add to MCP config)
- Playwright uses npx launcher instead of Docker
- Safe outputs use dedicated MCP server
- Custom tools require proper MCP server configuration

### Authentication
- Use `COPILOT_GITHUB_TOKEN` secret for GitHub token
- GitHub Actions default token is incompatible with Copilot CLI
- Must use Personal Access Token (PAT)
- Ensure token has appropriate permissions for repository access
- Token is passed via environment variables to CLI

**Note**: The `COPILOT_CLI_TOKEN` secret name is no longer supported as of v0.26+.

## Development Guidelines

### When Working with Copilot Engine Code
- Follow MCP server configuration patterns in `copilot_engine.go`
- Use "local" type instead of "stdio" for MCP servers
- Handle built-in tools (like GitHub) by skipping MCP configuration
- Ensure proper environment variable setup
- Test with various tool combinations

### Log Parser Development
- Parse both structured and unstructured log output
- Handle multi-line code blocks correctly
- Extract meaningful error and warning information
- Generate proper markdown for step summaries
- Account for CLI-specific output formats

### Testing Considerations
- Mock CLI responses for unit tests
- Test MCP configuration generation
- Validate log parsing with various output formats
- Ensure timeout handling works correctly
- Test authentication scenarios

## Command Examples

### Basic Code Generation
```bash
copilot --prompt "Generate a Python function to calculate fibonacci numbers"
```

### File Analysis
```bash
copilot --add-dir /project --prompt "Analyze the code structure and suggest improvements"
```

### GitHub Integration
```bash
copilot --add-dir /repo --prompt "Create an issue summarizing the recent changes"
```

### With Logging
```bash
copilot --add-dir /tmp/gh-aw \
        --log-level debug \
        --log-dir /tmp/gh-aw/logs \
        --prompt "Review the code and suggest optimizations"
```

## Security Considerations

- Validate all prompts before execution to prevent injection
- Restrict directory access using `--add-dir` carefully
- Ensure MCP servers are from trusted sources
- Log sensitive operations for audit trails
- Use least-privilege tokens for authentication
- Sanitize log output before displaying to users

## Troubleshooting

### CLI Not Found
- Verify npm global installation: `npm list -g @github/copilot`
- Check PATH includes npm global bin directory
- Try reinstalling: `npm uninstall -g @github/copilot && npm install -g @github/copilot`

### Authentication Issues
- **GitHub Actions Token Incompatibility**: The default `GITHUB_TOKEN` does NOT work with Copilot CLI
- Verify you're using a Personal Access Token in `COPILOT_GITHUB_TOKEN` secret
- Verify the token is associated with a Copilot-enabled GitHub account
- For GitHub Enterprise, contact admin for Copilot CLI token access

### MCP Server Issues
- Validate JSON configuration syntax
- Check server command availability
- Verify network connectivity for HTTP servers
- Review server logs for connection errors

### Performance Issues
- Reduce directory scope with targeted `--add-dir`
- Lower log level to reduce I/O overhead
- Set appropriate timeouts for operations
- Monitor token usage and rate limits
