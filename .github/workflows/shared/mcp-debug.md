---
mcp-servers:
  gh-aw:
    type: http
    url: http://localhost:8765
    allowed:
      - mcp-inspect
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  - name: Install dependencies
    run: make deps-dev
  - name: Install binary as 'gh-aw'
    run: make build
  - name: Start MCP server
    run: |
      set -e
      ./gh-aw mcp-server --cmd ./gh-aw --port 8765 &
      MCP_PID=$!
      
      # Wait a moment for server to start
      sleep 2
      
      # Check if server is still running
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "MCP server failed to start"
        exit 1
      fi
      
      echo "MCP server started successfully with PID $MCP_PID"
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---

## MCP Server Debugging Assistant

This shared workflow provides tools and procedures to debug MCP (Model Context Protocol) server configurations and diagnose runtime issues.

### Available Tools

**mcp-inspect Tool**: Use the `mcp-inspect` tool to inspect MCP server configurations and diagnose connectivity issues.

The tool can:
- List all workflows with MCP servers
- Inspect specific MCP servers in a workflow
- Show available tools and their status
- Display tool schemas and parameters
- Verify HTTP endpoint connectivity

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

#### Step 3: Use mcp-inspect Tool for HTTP Servers

For HTTP-based MCP servers, use the `mcp-inspect` tool to diagnose connectivity and tool availability.

**Tool Parameters:**
- `workflow_file`: The workflow file to inspect (e.g., "dev" or "audit-workflows")
- `server`: (Optional) Filter to inspect only the specified MCP server
- `tool`: (Optional) Show detailed information about a specific tool

**Usage Examples:**

To inspect all MCP servers in the current workflow:
```
Use mcp-inspect tool with workflow_file parameter set to the current workflow name
```

To inspect a specific MCP server:
```
Use mcp-inspect tool with workflow_file="<workflow-name>" and server="<server-name>"
```

To get detailed information about a specific tool:
```
Use mcp-inspect tool with workflow_file="<workflow-name>", server="<server-name>", and tool="<tool-name>"
```

The mcp-inspect tool will:
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

**IMPORTANT**: When reporting your findings, include:
- **Detailed description** of what you discovered during investigation
- **Specific error messages** or symptoms found in logs
- **Root cause analysis** explaining why the failure occurred
- **Evidence** from log files or diagnostic tools that support your conclusion
- **Recommendations** for fixing the issue based on your analysis

Create a clear summary of your investigation that includes all diagnostic steps you performed and their results.

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
1. Use the `mcp-inspect` tool to list available tools
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
- Access to mcp-inspect tool for MCP server diagnostics
- Documentation for MCP debugging procedures
- Reference for common issues and solutions

### Best Practices

1. **Always check logs first**: MCP server logs contain the most direct information about failures
2. **Use mcp-inspect tool for HTTP servers**: The tool provides real-time diagnostics for MCP server connectivity
3. **Document your findings**: Include log excerpts and error messages in issues
4. **Test incrementally**: Debug one server at a time
5. **Verify dependencies**: Ensure all required packages are installed before starting servers
6. **Check ports**: Avoid port conflicts by verifying ports are free before starting servers

### Example Debugging Session

```
# 1. Server failed - check what went wrong
Read /tmp/gh-aw/mcp-logs/drain3/server.log
# Output: ModuleNotFoundError: No module named 'fastmcp'

# 2. Document the error and check if dependencies need to be installed
# Note: In a workflow context, dependency installation should be added to the workflow steps

# 3. After server restarts, verify connectivity
Use mcp-inspect tool with workflow_file="dev" and server="drain3"
# Output: ✓ Successfully connected, 3 tools available

# 4. Verify specific tool details
Use mcp-inspect tool with workflow_file="dev", server="drain3", and tool="index_file"
# Output: Tool details showing correct configuration
```
