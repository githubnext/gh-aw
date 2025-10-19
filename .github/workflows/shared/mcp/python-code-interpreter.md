---
mcp-servers:
  python-code-interpreter:
    type: http
    url: http://localhost:9753/mcp
    allowed:
      - run_python_query
steps:
  - name: Set up Python
    uses: actions/setup-python@v5
    with:
      python-version: '3.11'
  - name: Install Python code interpreter dependencies
    run: |
      pip install fastmcp pandas numpy scipy matplotlib seaborn plotly
  - name: Copy Python code interpreter MCP server script
    run: |
      mkdir -p /tmp/gh-aw/mcp-servers/python-code-interpreter/
      cp .github/workflows/shared/mcp/python-code-interpreter_server.py /tmp/gh-aw/mcp-servers/python-code-interpreter/
      chmod +x /tmp/gh-aw/mcp-servers/python-code-interpreter/python-code-interpreter_server.py
  - name: Start Python code interpreter MCP server
    run: |
      set -e
      mkdir -p /tmp/gh-aw/mcp-logs/python-code-interpreter/
      python /tmp/gh-aw/mcp-servers/python-code-interpreter/python-code-interpreter_server.py > /tmp/gh-aw/mcp-logs/python-code-interpreter/server.log 2>&1 &
      MCP_PID=$!
      
      # Wait for server to start
      sleep 3
      
      # Check if server is still running
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "Python code interpreter MCP server failed to start"
        echo "Server logs:"
        cat /tmp/gh-aw/mcp-logs/python-code-interpreter/server.log || true
        exit 1
      fi
      
      # Check if server is listening on port 9753
      if ! netstat -tln | grep -q ":9753 "; then
        echo "Python code interpreter MCP server not listening on port 9753"
        echo "Server logs:"
        cat /tmp/gh-aw/mcp-logs/python-code-interpreter/server.log || true
        exit 1
      fi
      
      # Test HTTP endpoint with curl
      echo "Testing HTTP endpoint with curl..."
      if curl -v -X GET http://localhost:9753/mcp 2>&1 | tee /tmp/gh-aw/mcp-logs/python-code-interpreter/curl-test.log; then
        echo "✓ HTTP endpoint responded"
      else
        echo "✗ HTTP endpoint did not respond"
        echo "Server logs:"
        cat /tmp/gh-aw/mcp-logs/python-code-interpreter/server.log || true
        echo "Curl test logs:"
        cat /tmp/gh-aw/mcp-logs/python-code-interpreter/curl-test.log || true
        exit 1
      fi
      
      echo "Python code interpreter MCP server started successfully with PID $MCP_PID"
      echo "Server logs (first 50 lines):"
      head -n 50 /tmp/gh-aw/mcp-logs/python-code-interpreter/server.log || true
    env:
      PORT: "9753"
---

<!--

Python Code Interpreter MCP Server
Execute Python data analysis code in isolated environments

This MCP server provides a Python code interpreter that executes data analysis
code in isolated folders. Each request runs in its own directory with optional
file copying support for data analysis workflows.

Documentation: https://github.com/jlowin/fastmcp

Features:
- Isolated execution: Each request runs in /tmp/gh-aw/python-runs/<uuid>
- File support: Copy input files into the run directory
- Real-time streaming: See execution output as it happens
- Output collection: All generated files are tracked and returned
- Pre-installed libraries: pandas, numpy, scipy, matplotlib, seaborn, plotly

Available tools:
  - run_python_query: Execute Python data-analysis code in an isolated folder
    Parameters:
      - code: Python code to execute
      - files: Optional list of file paths to copy into the run directory
    Returns: Run metadata including run_id, run_path, exit_code, and list of files

Configuration:
  The server runs on port 9753 and executes Python 3.11+ code with common
  data analysis libraries pre-installed.

Setup:
  1. Include in Your Workflow:
     imports:
       - shared/mcp/python-code-interpreter.md

  2. The server will be automatically installed and started on localhost:9753

Example Usage:
  Generate a histogram plot of file sizes in the repository:

  ```
  Use the python-code-interpreter tool to analyze file sizes in the repository.
  First, collect all file paths and sizes, then create a histogram plot
  showing the distribution of file sizes. Save the plot as histogram.png.
  ```

  The agent can reference files by their basename (e.g., "data.csv") after
  providing the full path in the files parameter.

Connection Type:
  This configuration uses a local HTTP MCP server running Python with FastMCP 2.0.
  The server runs with transport="http" and responses stream for real-time output.

Troubleshooting:
  Server Failed to Start:
  - Verify Python 3.11+ is available
  - Check that port 9753 is not in use
  - Review server logs for dependency installation issues
  
  Execution Errors:
  - Ensure file paths are absolute when using the files parameter
  - Check that required Python libraries are available
  - Verify file permissions are readable

Usage:
  imports:
    - shared/mcp/python-code-interpreter.md

-->
