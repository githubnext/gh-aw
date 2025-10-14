---
mcp-servers:
  drain3:
    container: "mcp/drain3"
    allowed:
      - parse_log
      - get_clusters
---

## Drain3 MCP Server

Drain3 is an online log template miner that extracts patterns from log messages. This MCP server wraps Drain3 to provide log parsing capabilities to agentic workflows.

### Available Tools

- **`parse_log`**: Parse a log line and return the mined template with cluster information
  - Input: `log_line` (string) - The log line to parse
  - Output: JSON with `cluster_id`, `template`, and `change_type`
  
- **`get_clusters`**: Get information about all discovered log clusters and their templates
  - Output: JSON with `total_clusters` and list of clusters with their templates and sizes

### How It Works

Drain3 uses a fixed depth tree structure to efficiently parse log messages and extract templates. It automatically:
- Identifies log patterns by clustering similar log messages
- Extracts variable parts (like IDs, timestamps, numbers) and replaces them with wildcards (`<*>`)
- Groups log messages into clusters based on their templates
- Persists learned patterns across runs

**Example:**

Given log lines:
```
INFO user alice logged in
INFO user bob logged in
ERROR connection timeout after 30 seconds
ERROR connection timeout after 45 seconds
```

Drain3 will extract templates:
```
Template 1: "INFO user <*> logged in" (2 occurrences)
Template 2: "ERROR connection timeout after <*> seconds" (2 occurrences)
```

### Usage Example

```yaml
---
on: workflow_dispatch
engine: claude
imports:
  - shared/mcp/drain3.md
---

# Log Pattern Analyzer

Analyze the following log lines and extract common patterns:

```
2024-01-15 10:23:45 INFO Starting service on port 8080
2024-01-15 10:23:46 INFO Starting service on port 9090
2024-01-15 10:24:12 ERROR Failed to connect to database: timeout
2024-01-15 10:24:15 ERROR Failed to connect to database: connection refused
```

Use the parse_log tool to process each log line and identify patterns.
Then use get_clusters to see all discovered templates.
```

### Reference

- Drain3 GitHub: https://github.com/logpai/Drain3
- Paper: "Drain: An Online Log Parsing Approach with Fixed Depth Tree" (IEEE ICWS 2017)

### Building the Docker Image

To build the Docker image locally:

```bash
cd .github/workflows/shared/mcp/drain3
docker build -t mcp/drain3 .
```

### Testing the Server

Test the MCP server locally:

```bash
# Run the server
docker run -i mcp/drain3

# Or test with a sample log line (requires MCP client)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"parse_log","arguments":{"log_line":"INFO user bob logged in"}}}' | docker run -i mcp/drain3
```
