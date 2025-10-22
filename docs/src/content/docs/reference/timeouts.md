---
title: Timeout Configuration
description: Complete guide to timeout settings for workflows, tools, and MCP servers in GitHub Agentic Workflows.
sidebar:
  order: 750
---

GitHub Agentic Workflows provides multiple timeout configurations to control execution time limits at different levels: workflow execution, tool operations, and MCP server initialization.

## Timeout Types

### Workflow Timeout (`timeout_minutes`)

Controls the maximum execution time for the entire workflow job. This is a standard GitHub Actions property.

```yaml
---
timeout_minutes: 30  # Workflow fails after 30 minutes
---
```

**Default**: 20 minutes for agentic workflows

**Range**: 1-360 minutes (6 hours max for GitHub-hosted runners)

**Use cases**:
- Prevent runaway workflows from consuming excessive runner time
- Enforce SLA requirements for workflow completion
- Control costs by limiting maximum execution duration

### Tool Timeout (`engine.timeout`)

Controls the maximum execution time for individual tool or MCP server operations during workflow execution.

```yaml
---
engine:
  id: claude
  timeout: 120  # Each tool call times out after 120 seconds
---
```

**Defaults by engine**:
- **Claude**: 60 seconds
- **Codex**: 120 seconds
- **Copilot**: Uses engine defaults

**Applies to**:
- GitHub API calls
- Bash command execution
- MCP server tool invocations
- File operations
- Web fetch operations

**Engine support**: Claude Code and Codex support this setting. Check engine-specific documentation for availability.

### MCP Server Startup Timeout (`engine.startup-timeout`)

Controls the maximum time allowed for MCP servers to initialize and become ready.

```yaml
---
engine:
  id: claude
  startup-timeout: 180  # MCP servers must start within 180 seconds
---
```

**Default**: 120 seconds (2 minutes)

**Applies to**: All MCP server initialization, including Docker container startup and stdio process initialization.

**Use cases**:
- Docker images with large downloads or slow startup
- MCP servers with complex initialization procedures
- Network-dependent startup processes

## Configuration Examples

### Basic Timeout Configuration

```yaml
---
on:
  issues:
    types: [opened]
timeout_minutes: 15
engine:
  id: claude
  timeout: 90
  startup-timeout: 150
---

Analyze the issue and provide recommendations.
```

### Production Workflow with Strict Timeouts

```yaml
---
on:
  pull_request:
    types: [opened]
timeout_minutes: 10
engine:
  id: claude
  timeout: 45
  startup-timeout: 120
strict: true
---

Review pull request changes and add inline comments.
```

### Long-Running Research Workflow

```yaml
---
on:
  schedule:
    - cron: "0 0 * * 0"
timeout_minutes: 60
engine:
  id: claude
  timeout: 180
  startup-timeout: 180
tools:
  web-fetch:
  web-search:
---

Conduct weekly competitive analysis research.
```

## CLI Command Timeouts

The `gh aw logs` command supports a `--timeout` flag to limit execution time when downloading workflow logs.

```bash
gh aw logs --timeout 60  # Stop after 60 seconds
gh aw logs --timeout 0   # No timeout (default)
```

**Default**: No timeout (0 seconds)

**Behavior**: When timeout is reached, the command processes already-downloaded runs and returns partial results.

**MCP server**: The gh-aw MCP server uses a 50-second default timeout for log operations to prevent MCP client timeouts.

## Timeout Best Practices

### Setting Appropriate Timeouts

**Workflow timeout** (`timeout_minutes`):
- Development workflows: 10-15 minutes
- Production workflows: 5-10 minutes (with strict mode)
- Research/analysis: 30-60 minutes
- CI/CD integration: Match your existing CI timeout policies

**Tool timeout** (`engine.timeout`):
- Simple operations: 30-60 seconds
- API-heavy workflows: 90-120 seconds
- Complex analysis: 120-180 seconds
- Network-dependent operations: 120-180 seconds

**MCP startup timeout** (`engine.startup-timeout`):
- stdio MCP servers: 30-60 seconds
- Docker MCP servers: 120-180 seconds
- Large Docker images: 180-300 seconds

### Strict Mode Requirements

When using strict mode, configure explicit timeouts to enforce time limits:

```yaml
---
strict: true
timeout_minutes: 10
engine:
  id: claude
  timeout: 60
---
```

Strict mode helps ensure workflows complete within predictable time bounds for production use.

### Timeout Hierarchy

Timeouts work independently at different levels:

1. **Workflow timeout** (`timeout_minutes`): Terminates the entire job
2. **Tool timeout** (`engine.timeout`): Fails individual tool operations
3. **MCP startup timeout** (`engine.startup-timeout`): Fails MCP server initialization

A tool timeout does not terminate the workflow - the AI agent receives a timeout error and can retry or continue with other operations.

## Troubleshooting Timeout Issues

### Workflow Timing Out

**Symptoms**: Workflow job canceled with "timeout" error after `timeout_minutes` elapsed.

**Solutions**:
- Increase `timeout_minutes` if legitimate work requires more time
- Simplify workflow instructions to reduce processing time
- Use `web-fetch` and `web-search` tools efficiently to minimize network operations
- Consider splitting into multiple workflows with workflow_run triggers

### Tool Operations Timing Out

**Symptoms**: Agent reports tool execution failures, logs show timeout errors.

**Solutions**:
- Increase `engine.timeout` for complex operations
- Review network configuration - restricted domains may cause connection timeouts
- Check GitHub API rate limits - throttling may delay operations
- Verify MCP server implementations handle timeouts gracefully

### MCP Server Startup Timeout

**Symptoms**: Workflow fails during initialization with MCP server startup errors.

**Solutions**:
- Increase `engine.startup-timeout` for Docker-based MCP servers
- Pre-pull Docker images to reduce startup time:
  ```yaml
  steps:
    - name: Pre-pull MCP images
      run: docker pull ghcr.io/my-mcp-server:latest
  ```
- Use stdio transport instead of Docker when startup time is critical
- Check MCP server logs for initialization issues

### CLI Logs Timeout

**Symptoms**: `gh aw logs` command stops early with partial results.

**Solutions**:
- Use continuation parameters returned in output to fetch remaining logs
- Increase `--timeout` value for larger log downloads
- Use `--count` to limit number of workflow runs fetched
- Cache downloaded runs for faster reprocessing

## Engine-Specific Considerations

### Claude Code

**Default timeout**: 60 seconds
**Supports**: Both `timeout` and `startup-timeout` configuration

Claude Code provides detailed timeout error messages in logs showing which tool operation timed out.

### Codex

**Default timeout**: 120 seconds
**Supports**: Both `timeout` and `startup-timeout` configuration

Codex may include additional timeout configuration in the `config` field for engine-specific behavior.

### Copilot

**Default timeout**: Engine-managed (not configurable)
**Supports**: Limited timeout configuration

Copilot uses its own timeout management. Workflow-level `timeout_minutes` still applies.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration reference
- [AI Engines](/gh-aw/reference/engines/) - Engine-specific configuration options
- [Tools](/gh-aw/reference/tools/) - Tool configuration and MCP servers
- [CLI Commands](/gh-aw/tools/cli/) - Command-line timeout flags
- [Strict Mode](/gh-aw/guides/security/) - Security validation requirements
