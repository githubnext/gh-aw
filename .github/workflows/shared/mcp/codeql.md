---
# CodeQL MCP Server
# MCP server that wraps the CodeQL query server for semantic code analysis
#
# Documentation: https://github.com/JordyZomer/codeql-mcp
#
# Prerequisites:
#   - CodeQL CLI must be installed
#   - CodeQL database must be created for the target repository
#
# Available tools:
#   - register_database: Register a CodeQL database given a path
#   - evaluate_query: Run a CodeQL query on a given database
#   - quick_evaluate: Quick-evaluate a class or predicate in a CodeQL query
#   - decode_bqrs: Decode CodeQL results to CSV or JSON format
#   - find_class_position: Find position of a class for quick evaluation
#   - find_predicate_position: Find position of a predicate for quick evaluation
#
# Setup Requirements:
#   1. CodeQL CLI installed in the workflow environment
#   2. CodeQL database created for the repository
#   3. Python dependencies: fastmcp, httpx
#
# Usage:
#   imports:
#     - shared/mcp/codeql.md

mcp-servers:
  codeql:
    type: http
    url: http://localhost:8000
    allowed: ["*"]

steps:
  - name: Install CodeQL CLI
    run: |
      set -e
      echo "Installing CodeQL CLI..."
      
      # Download and install CodeQL CLI
      CODEQL_VERSION="v2.19.3"
      CODEQL_URL="https://github.com/github/codeql-cli-binaries/releases/download/${CODEQL_VERSION}/codeql-linux64.zip"
      
      # Download CodeQL
      curl -L -o /tmp/codeql.zip "${CODEQL_URL}"
      
      # Extract to a known location
      sudo unzip -q /tmp/codeql.zip -d /usr/local/
      
      # Add to PATH
      echo "/usr/local/codeql" >> $GITHUB_PATH
      
      # Verify installation
      /usr/local/codeql/codeql --version
      
      echo "CodeQL CLI installed successfully"

  - name: Install Python dependencies for CodeQL MCP server
    run: |
      set -e
      echo "Installing Python dependencies for CodeQL MCP server..."
      
      # Install required Python packages
      pip install fastmcp httpx
      
      echo "Python dependencies installed successfully"

  - name: Clone CodeQL MCP server
    run: |
      set -e
      echo "Cloning CodeQL MCP server repository..."
      
      # Clone the MCP server repository
      git clone https://github.com/JordyZomer/codeql-mcp.git /tmp/codeql-mcp
      
      echo "CodeQL MCP server cloned successfully"

  - name: Start CodeQL MCP server
    run: |
      set -e
      
      # Start the CodeQL MCP server in the background
      cd /tmp/codeql-mcp
      python3 server.py &
      MCP_PID=$!
      
      # Robust health check with TCP connection test
      echo "Waiting for CodeQL MCP server to start (PID: $MCP_PID)..."
      for i in {1..30}; do
        # Check if process is still running
        if ! kill -0 $MCP_PID 2>/dev/null; then
          echo "Error: CodeQL MCP server process died unexpectedly"
          exit 1
        fi
        
        # Try to connect to the server port
        if timeout 1 bash -c "echo > /dev/tcp/localhost/8000" 2>/dev/null; then
          echo "CodeQL MCP server is accepting connections on port 8000"
          echo "CodeQL MCP server started successfully with PID $MCP_PID"
          exit 0
        fi
        
        echo "Waiting for server to accept connections... (attempt $i/30)"
        sleep 2
      done
      
      echo "Error: CodeQL MCP server failed to accept connections after 60 seconds"
      exit 1
---

## CodeQL MCP Server

CodeQL is a semantic code analysis engine that helps identify vulnerabilities and code quality issues. This MCP server wraps the CodeQL query server to enable AI agents to interact with CodeQL through structured commands.

### Available Tools

The CodeQL MCP server provides the following tools for semantic code analysis:

1. **register_database** - Register a CodeQL database given a path
   - Parameters: `db_path` (string)
   - Returns: Confirmation message
   - Example: Register a database at `/path/to/database`

2. **evaluate_query** - Run a full CodeQL query on a database
   - Parameters: `query_path` (string), `db_path` (string), `output_path` (string, default: `/tmp/eval.bqrs`)
   - Returns: Path to the results file
   - Example: Run a security query to find SQL injection vulnerabilities

3. **quick_evaluate** - Quick-evaluate a specific class or predicate in a CodeQL query
   - Parameters: `file` (string), `db` (string), `symbol` (string), `output_path` (string, default: `/tmp/quickeval.bqrs`)
   - Returns: Path to the results file
   - Example: Evaluate a specific predicate without running the full query

4. **decode_bqrs** - Decode CodeQL binary results to human-readable format
   - Parameters: `bqrs_path` (string), `fmt` (string: "csv" or "json")
   - Returns: Decoded results
   - Example: Convert query results to JSON for further processing

5. **find_class_position** - Find the position of a class in a CodeQL file
   - Parameters: `file` (string), `name` (string)
   - Returns: Object with `start_line`, `start_col`, `end_line`, `end_col`
   - Example: Locate a class definition for quick evaluation

6. **find_predicate_position** - Find the position of a predicate in a CodeQL file
   - Parameters: `file` (string), `name` (string)
   - Returns: Object with `start_line`, `start_col`, `end_line`, `end_col`
   - Example: Locate a predicate definition for quick evaluation

### Basic Usage

The MCP server exposes CodeQL functionality through its MCP tools interface. When using CodeQL in your workflow, you can:

1. **Register databases**: Point CodeQL to the database for your repository
2. **Run queries**: Execute full queries or quick-evaluate specific symbols
3. **Analyze results**: Decode and process query results in CSV or JSON format
4. **Navigate code**: Find positions of classes and predicates in CodeQL files

### Workflow Example

```markdown
---
on: workflow_dispatch
permissions:
  security-events: write
  contents: read
engine: copilot
imports:
  - shared/mcp/codeql.md
---

# CodeQL Security Analysis

Analyze the repository for security vulnerabilities using CodeQL.

1. Create a CodeQL database for the repository
2. Register the database with the MCP server
3. Run security queries to identify vulnerabilities
4. Decode and analyze the results
5. Generate a security report
```

### Creating a CodeQL Database

Before using the CodeQL MCP server, you need to create a database for your repository:

```yaml
steps:
  - name: Create CodeQL database
    run: |
      # Create database for the repository
      codeql database create /tmp/codeql-db \
        --language=<language> \
        --source-root=${{ github.workspace }}
      
      # The database is now available at /tmp/codeql-db
```

Replace `<language>` with your repository's primary language (e.g., `javascript`, `python`, `java`, `go`, `cpp`, `csharp`, `ruby`).

### Security Considerations

- **Database Creation**: CodeQL databases can be large; consider caching them for repeated use
- **Query Selection**: Use official CodeQL security queries from the CodeQL query repository
- **Results Handling**: CodeQL results may contain sensitive information; handle with care
- **Network Access**: The MCP server runs locally (localhost:8000) with no external network access

### More Information

- **CodeQL Documentation**: https://codeql.github.com/docs/
- **CodeQL CLI**: https://github.com/github/codeql-cli-binaries
- **CodeQL MCP Server**: https://github.com/JordyZomer/codeql-mcp
- **Query Writing Guide**: https://codeql.github.com/docs/writing-codeql-queries/
- **Security Queries**: https://github.com/github/codeql
