<mcp-clis>
CLI servers are available on `PATH`:
__GH_AW_MCP_CLI_SERVERS_LIST__
Use `<server> --help` for tool names, parameters, and examples before calling any command.
To pass many arguments safely, pipe a JSON object on stdin with `printf` and pass `.` as the payload sentinel: `printf '%s\n' '{"param":"value","count":1}' | <server> <tool> .`
</mcp-clis>
