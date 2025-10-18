---
tools:
  agentic-workflows: {}
---

## MCP Server Debugging Assistant

This shared workflow provides tools and procedures to debug MCP (Model Context Protocol) server configurations and diagnose runtime issues.

### Available Tools

**Agentic Workflows Tool**: Enables introspection of workflow configurations and MCP server status using:
- `status` - Show status of workflow files in the repository
- `compile` - Compile markdown workflows to YAML
- `logs` - Download and analyze workflow run logs
- `audit` - Investigate workflow run failures and generate reports

### MCP Logs Location

MCP server logs are stored at `/tmp/gh-aw/mcp-logs/` with the following structure:

```
/tmp/gh-aw/mcp-logs/
├── <server-name>/
│   ├── server.log        # Main server startup and runtime logs
│   ├── curl-test.log     # HTTP endpoint connectivity tests (for HTTP servers)
│   └── <other>.log       # Additional server-specific logs
```

These logs contain valuable diagnostic information about:
- Server startup failures
- Dependency installation issues
- Port binding problems
- Configuration errors
- Runtime exceptions

### MCP Server Debugging Procedure

When you encounter issues with an MCP server, follow this systematic debugging procedure:

#### Step 1: Report the Error

Document the error you're experiencing:
- What operation failed?
- Which MCP server is affected?
- What error message(s) did you receive?
- What were you trying to accomplish?

#### Step 2: Read the Server Logs

Check the MCP server logs to understand what went wrong:

```bash
# Read the main server log
cat /tmp/gh-aw/mcp-logs/<server-name>/server.log

# For HTTP servers, check the curl test log
cat /tmp/gh-aw/mcp-logs/<server-name>/curl-test.log

# List all available logs for a server
ls -la /tmp/gh-aw/mcp-logs/<server-name>/
```

Look for:
- Python/Node.js import errors indicating missing dependencies
- Port binding errors (e.g., "Address already in use")
- Configuration validation errors
- Stack traces showing the failure point

#### Step 3: Use MCP Inspect for HTTP Servers

For HTTP-based MCP servers, use the `gh aw mcp inspect` command to diagnose connectivity and tool availability:

```bash
# Inspect all MCP servers in a workflow
gh aw mcp inspect <workflow-name>

# Inspect a specific MCP server
gh aw mcp inspect <workflow-name> --server <server-name>

# Get detailed information about a specific tool
gh aw mcp inspect <workflow-name> --server <server-name> --tool <tool-name>

# Verbose output with connection details
gh aw mcp inspect <workflow-name> --server <server-name> -v
```

The inspect command will:
- Verify HTTP endpoint connectivity
- List available tools and their status
- Show tool schemas and parameters
- Identify configuration problems

#### Step 4: Analyze and Report Findings

Based on the logs and inspection results, determine:
1. **Root cause**: What specifically failed?
2. **Impact**: Which tools or operations are affected?
3. **Workaround**: Can you proceed without this server?
4. **Fix**: What changes are needed to resolve the issue?

### Common MCP Server Issues

#### Issue: Server Failed to Start
**Symptoms**: Server process exits immediately after starting

**Debug steps**:
1. Check `server.log` for startup errors
2. Look for missing dependencies (ModuleNotFoundError, ImportError)
3. Verify port is not already in use (`netstat -tln | grep <port>`)
4. Check Python/Node version compatibility

**Common fixes**:
- Install missing packages: `pip install <package>` or `npm install <package>`
- Change server port in configuration
- Kill conflicting process on the port

#### Issue: HTTP Endpoint Not Responding
**Symptoms**: Connection refused errors, curl failures

**Debug steps**:
1. Check if server process is running: `ps aux | grep <server-name>`
2. Review `curl-test.log` for connection details
3. Verify server is listening: `netstat -tln | grep <port>`
4. Check firewall/network settings

**Common fixes**:
- Increase startup wait time in workflow configuration
- Verify HOST and PORT environment variables
- Check server transport mode (http vs stdio)

#### Issue: Tools Not Available
**Symptoms**: MCP tools show as "not allowed" or unavailable

**Debug steps**:
1. Use `gh aw mcp inspect` to list available tools
2. Compare with workflow's `allowed` list
3. Check tool registration in server code
4. Verify server initialized successfully

**Common fixes**:
- Add missing tools to `allowed` list in workflow configuration
- Fix tool registration errors in server code
- Restart server after configuration changes

### Integration with Workflows

To enable MCP debugging in your workflow, add this import:

```yaml
imports:
  - shared/mcp-debug.md
```

This provides:
- Access to agentic-workflows tool for workflow introspection
- Documentation for MCP debugging procedures
- Reference for common issues and solutions

### Best Practices

1. **Always check logs first**: MCP server logs contain the most direct information about failures
2. **Use inspect for HTTP servers**: The `gh aw mcp inspect` command provides real-time diagnostics
3. **Document your findings**: Include log excerpts and error messages in issues
4. **Test incrementally**: Debug one server at a time
5. **Verify dependencies**: Ensure all required packages are installed before starting servers
6. **Check ports**: Avoid port conflicts by verifying ports are free before starting servers

### Example Debugging Session

```bash
# 1. Server failed - check what went wrong
cat /tmp/gh-aw/mcp-logs/drain3/server.log
# Output: ModuleNotFoundError: No module named 'fastmcp'

# 2. Install missing dependency
pip install fastmcp

# 3. Restart and verify
# (server restarts automatically in workflow)
gh aw mcp inspect dev --server drain3
# Output: ✓ Successfully connected, 3 tools available

# 4. Verify specific tool
gh aw mcp inspect dev --server drain3 --tool index_file
# Output: Tool details showing correct configuration
```
