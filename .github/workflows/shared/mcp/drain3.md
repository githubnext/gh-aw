---
mcp-servers:
  drain3:
    type: http
    url: http://localhost:8766
    allowed:
      - index_file
      - query_file
      - list_templates
steps:
  - name: Set up Python
    uses: actions/setup-python@v5
    with:
      python-version: '3.11'
  - name: Install Drain3 dependencies
    run: |
      pip install fastmcp drain3
  - name: Copy Drain3 MCP server script
    run: |
      mkdir -p /tmp/gh-aw/mcp-servers/drain3/
      cp .github/workflows/shared/mcp/drain3_server.py /tmp/gh-aw/mcp-servers/drain3/
      chmod +x /tmp/gh-aw/mcp-servers/drain3/drain3_server.py
  - name: Start Drain3 MCP server
    run: |
      set -e
      fastmcp run /tmp/gh-aw/mcp-servers/drain3/drain3_server.py > /tmp/gh-aw/mcp-servers/drain3/server.log 2>&1 &
      MCP_PID=$!
      
      # Wait for server to start
      sleep 3
      
      # Check if server is still running
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "Drain3 MCP server failed to start"
        echo "Server logs:"
        cat /tmp/gh-aw/mcp-servers/drain3/server.log || true
        exit 1
      fi
      
      # Check if server is listening on port 8766
      if ! netstat -tln | grep -q ":8766 "; then
        echo "Drain3 MCP server not listening on port 8766"
        echo "Server logs:"
        cat /tmp/gh-aw/mcp-servers/drain3/server.log || true
        exit 1
      fi
      
      echo "Drain3 MCP server started successfully with PID $MCP_PID"
    env:
      PORT: "8766"
      HOST: "0.0.0.0"
      STATE_DIR: "${{ github.workspace }}/.drain3"
---

<!--

Drain3 MCP Server
Log template mining and pattern extraction tool

Drain3 is an online log template miner that extracts structured patterns from 
unstructured log files. This MCP server provides tools for indexing log files, 
querying patterns, and listing extracted templates.

Documentation: https://github.com/logpai/Drain3

This shared configuration provides a local HTTP MCP server that runs Drain3
for log analysis and template extraction. The server uses streaming JSONL
responses for progressive results.

Available tools:
  - index_file: Stream-mine templates from a log file and persist snapshot
    Parameters:
      - path: Path to the log file to analyze
      - encoding: File encoding (default: utf-8)
      - max_lines: Maximum lines to process (optional)
    Returns: Streaming JSONL events (start, progress, template, summary)

  - query_file: Match a log line against previously indexed templates
    Parameters:
      - path: Path to the indexed log file
      - text: Log line text to match
    Returns: Matching cluster information (cluster_id, size, template)

  - list_templates: List all extracted templates from an indexed file
    Parameters:
      - path: Path to the indexed log file
      - limit: Maximum number of templates to return (optional)
    Returns: Streaming JSONL events for each template

Configuration:
  The server can be configured via environment variables in the workflow:
  - DRAIN3_SIM_TH: Similarity threshold (default: 0.4)
  - DRAIN3_DEPTH: Tree depth (default: 4)
  - DRAIN3_MAX_CHILDREN: Maximum children per node (default: 100)
  - DRAIN3_MAX_CLUSTERS: Maximum clusters (default: 0 = unlimited)
  - STREAM_FLUSH_EVERY: Progress event frequency (default: 500 lines)
  - STREAM_SLEEP: Throttle between flushes in seconds (default: 0)

Setup:
  1. Include in Your Workflow:
     imports:
       - shared/mcp/drain3.md

  2. The server will be automatically installed and started on localhost:8766

Example Usage:
  Analyze GitHub Actions workflow logs to identify common error patterns
  and failure templates. Index a log file, then query specific error messages
  to find which cluster they belong to.

  ```
  Use the drain3 tool to index the workflow log file at /tmp/workflow.log
  and extract error patterns. Then query a specific error message to find
  its template cluster.
  ```

Connection Type:
  This configuration uses a local HTTP MCP server running Python with FastMCP 2.0.
  The server is launched using the `fastmcp run` command which handles HTTP 
  transport automatically. Responses stream as JSONL (JSON Lines) for progressive results.

State Persistence:
  Drain3 snapshots are stored in ${{ github.workspace }}/.drain3/ directory.
  Each indexed file gets its own snapshot file for quick reloading.

Troubleshooting:
  Server Failed to Start:
  - Verify Python 3.11+ is available
  - Check that port 8766 is not in use
  - Review server logs for dependency installation issues
  
  Index/Query Errors:
  - Ensure file paths are absolute or relative to workspace
  - Check that the file was indexed before querying
  - Verify file permissions are readable

Usage:
  imports:
    - shared/mcp/drain3.md

-->
