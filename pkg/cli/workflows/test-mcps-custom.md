---
on: issues
permissions:
  contents: read
  issues: write
engine: claude
mcp-servers:
  # New direct field format - stdio with command
  my-stdio-server:
    type: stdio
    command: "python"
    args: ["-m", "my_server"]
    env:
      API_KEY: "secret123"
    registry: "https://registry.example.com/servers/my-stdio-server"
    proxy-args: ["--custom-arg"]
    allowed: ["process_data", "get_info"]
    
  # New direct field format - http with url
  my-http-server:
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_TOKEN }}"
    registry: "https://registry.example.com/servers/my-http-server"
    allowed: ["fetch_data"]
    
  # Type inference - local type alias
  local-server:
    type: local
    command: "local-tool"
    args: ["--local"]
    allowed: ["local_action"]
    
  # Type inference - no type specified, inferred from command
  inferred-stdio:
    command: "inferred-server"
    args: ["--mode", "stdio"]
    allowed: ["inferred_tool"]
    
  # Type inference - no type specified, inferred from url  
  inferred-http:
    url: "https://inferred.api.com/mcp"
    allowed: ["inferred_http_tool"]
---

# Test Workflow

Test workflow with new MCP configuration format.
