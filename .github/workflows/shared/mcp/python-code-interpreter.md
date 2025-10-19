---
mcp-servers:
  python-code-interpreter:
    container: "python-code-interpreter:latest"
    args:
      - "-v"
      - "${{ github.workspace }}:/workspace"
    allowed: ["*"]
steps:
  - name: Build Python code interpreter Docker image
    run: |
      cd .github/workflows/shared/mcp
      docker build -f python-code-interpreter.Dockerfile -t python-code-interpreter:latest .
---

<!--

Python Code Interpreter MCP Server
Execute Python data analysis code in isolated environments

This MCP server provides a Python code interpreter that executes data analysis
code in isolated folders. Each request runs in its own directory with optional
file copying support for data analysis workflows.

Documentation: https://github.com/jlowin/fastmcp

**IMPORTANT - Sandbox Environment:**
You are running in a restricted sandbox environment where direct bash and Python 
execution is locked down for security. You MUST use the `run_python_query` MCP 
tool to execute any Python code. Do not attempt to use bash's `python` command 
or local Python execution - they will not work. All Python code execution must 
go through the MCP server.

Features:
- Isolated execution: Each request runs in /app/runs/<uuid>
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
  The server runs as a Docker container with Python 3.11 and executes code using
  stdio transport (FastMCP default). The workspace is mounted to /workspace for
  file access.

Setup:
  1. Include in Your Workflow:
     imports:
       - shared/mcp/python-code-interpreter.md

  2. The Docker image will be built automatically and the server started

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
  This configuration uses a Docker container with stdio transport (FastMCP default).
  The server runs via `mcp.run()` and communicates through standard input/output.

Troubleshooting:
  Docker Build Failed:
  - Verify Docker is available in the runner
  - Check network connectivity for dependency downloads
  - Review Docker build logs
  
  Execution Errors:
  - Ensure file paths are accessible from within the container
  - Use /workspace prefix for files in the GitHub workspace
  - Check that required Python libraries are available
  - Verify file permissions are readable

Usage:
  imports:
    - shared/mcp/python-code-interpreter.md

-->
