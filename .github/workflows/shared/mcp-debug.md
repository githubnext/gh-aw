---
mcp-servers:
  gh-aw:
    type: http
    url: http://localhost:8765
    allowed:
      - mcp-inspect
safe-outputs:
  jobs:
    report-diagnostics-to-pull-request:
      description: "Post MCP diagnostic findings as a comment on the pull request associated with the triggering branch"
      runs-on: ubuntu-latest
      output: "Diagnostic report posted to pull request successfully!"
      inputs:
        message:
          description: "The diagnostic message to post as a PR comment"
          required: true
          type: string
      permissions:
        contents: read
        pull-requests: write
      steps:
        - name: Checkout repository
          uses: actions/checkout@v4
        - name: Post diagnostic report to pull request
          uses: actions/github-script@v8
          with:
            script: |
              const fs = require('fs');
              const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === 'true';
              const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
              
              // Read and parse agent output
              if (!outputContent) {
                core.info('No GITHUB_AW_AGENT_OUTPUT environment variable found');
                return;
              }
              
              let agentOutputData;
              try {
                const fileContent = fs.readFileSync(outputContent, 'utf8');
                agentOutputData = JSON.parse(fileContent);
              } catch (error) {
                core.setFailed(`Error reading or parsing agent output: ${error instanceof Error ? error.message : String(error)}`);
                return;
              }
              
              if (!agentOutputData.items || !Array.isArray(agentOutputData.items)) {
                core.info('No valid items found in agent output');
                return;
              }
              
              // Filter for report_diagnostics_to_pull_request items
              const diagnosticItems = agentOutputData.items.filter(item => item.type === 'report_diagnostics_to_pull_request');
              
              if (diagnosticItems.length === 0) {
                core.info('No report_diagnostics_to_pull_request items found in agent output');
                return;
              }
              
              core.info(`Found ${diagnosticItems.length} report_diagnostics_to_pull_request item(s)`);
              
              // Get the current branch
              const ref = context.ref;
              const branch = ref.replace('refs/heads/', '');
              core.info(`Current branch: ${branch}`);
              
              // Find pull requests associated with this branch
              let pullRequests;
              try {
                const { data } = await github.rest.pulls.list({
                  owner: context.repo.owner,
                  repo: context.repo.repo,
                  head: `${context.repo.owner}:${branch}`,
                  state: 'open'
                });
                pullRequests = data;
              } catch (error) {
                core.setFailed(`Failed to list pull requests: ${error instanceof Error ? error.message : String(error)}`);
                return;
              }
              
              if (pullRequests.length === 0) {
                core.warning(`No open pull requests found for branch: ${branch}`);
                core.info('Diagnostic report cannot be posted without an associated pull request');
                return;
              }
              
              const pullRequest = pullRequests[0];
              const prNumber = pullRequest.number;
              core.info(`Found pull request #${prNumber} for branch ${branch}`);
              
              // Process each diagnostic item
              for (let i = 0; i < diagnosticItems.length; i++) {
                const item = diagnosticItems[i];
                const message = item.message;
                
                if (!message) {
                  core.warning(`Item ${i + 1}: Missing message field, skipping`);
                  continue;
                }
                
                if (isStaged) {
                  let summaryContent = "## 🎭 Staged Mode: Diagnostic Report Preview\n\n";
                  summaryContent += "The following diagnostic report would be posted to the pull request if staged mode was disabled:\n\n";
                  summaryContent += `**Pull Request:** #${prNumber}\n`;
                  summaryContent += `**Branch:** ${branch}\n\n`;
                  summaryContent += `**Diagnostic Message:**\n\n${message}\n\n`;
                  await core.summary.addRaw(summaryContent).write();
                  core.info("📝 Diagnostic report preview written to step summary");
                  continue;
                }
                
                core.info(`Posting diagnostic report ${i + 1}/${diagnosticItems.length} to PR #${prNumber}`);
                
                try {
                  const { data: comment } = await github.rest.issues.createComment({
                    owner: context.repo.owner,
                    repo: context.repo.repo,
                    issue_number: prNumber,
                    body: message
                  });
                  
                  core.info(`✅ Diagnostic report ${i + 1} posted successfully`);
                  core.info(`Comment ID: ${comment.id}`);
                  core.info(`Comment URL: ${comment.html_url}`);
                } catch (error) {
                  core.setFailed(`Failed to post diagnostic report ${i + 1}: ${error instanceof Error ? error.message : String(error)}`);
                  return;
                }
              }
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
- Safe-output job for posting diagnostic reports to pull requests

### Safe Job: report-diagnostics-to-pull-request

The `report_diagnostics_to_pull_request` safe-job allows agentic workflows to post MCP diagnostic findings as comments on the pull request associated with the triggering branch.

**Purpose**: Enable AI agents to report MCP debugging findings directly on pull requests for collaborative review and resolution.

**Agent Output Format:**

The agent should output JSON with items of type `report_diagnostics_to_pull_request`:

```json
{
  "items": [
    {
      "type": "report_diagnostics_to_pull_request",
      "message": "## MCP Diagnostic Report\n\n**Issue**: Server failed to start\n\n**Analysis**: Missing dependency..."
    }
  ]
}
```

**Behavior:**
- Automatically resolves the pull request associated with the current branch
- Posts the diagnostic message as a comment on that PR
- If no open PR is found for the branch, a warning is logged and the job completes without error
- Supports staged mode for preview without posting

**Required Permissions:**
- `contents: read` - To checkout the repository
- `pull-requests: write` - To post comments on pull requests

**Staged Mode Support:**

When `staged: true` is set in the workflow's safe-outputs configuration, diagnostic reports will be previewed in the step summary instead of being posted to the pull request.

**Example Usage in Workflow:**

```markdown
Use the mcp-inspect tool to diagnose the drain3 server.
Post a summary of your findings using the report_diagnostics_to_pull_request output type.
Include the specific error, root cause analysis, and recommended fixes.
```

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
