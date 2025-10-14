#!/usr/bin/env python3
"""
Minimal MCP (Model Context Protocol) server wrapping Drain3.
Drain3 is a log template miner that extracts patterns from log messages.
"""

import sys
import json
import asyncio
from typing import Any

from drain3 import TemplateMiner
from drain3.file_persistence import FilePersistence
from drain3.template_miner_config import TemplateMinerConfig

try:
    from mcp.server import Server
    from mcp.types import Tool, TextContent
except ImportError:
    print("Error: modelcontextprotocol package not found. Install with: pip install mcp", file=sys.stderr)
    sys.exit(1)

# Initialize Drain3 template miner with file persistence
cfg = TemplateMinerConfig()
persistence = FilePersistence("/tmp/drain3_state.bin")
miner = TemplateMiner(persistence_handler=persistence, config=cfg)

# Create MCP server
server = Server(name="drain3", version="0.1.0")


@server.list_tools()
async def list_tools() -> list[Tool]:
    """List available tools for the Drain3 MCP server."""
    return [
        Tool(
            name="parse_log",
            description="Parse a log line and return mined template + cluster info. Drain3 automatically identifies log patterns and extracts structured templates.",
            inputSchema={
                "type": "object",
                "properties": {
                    "log_line": {
                        "type": "string",
                        "description": "The log line to parse and extract patterns from"
                    }
                },
                "required": ["log_line"],
            },
        ),
        Tool(
            name="get_clusters",
            description="Get information about all discovered log clusters and their templates.",
            inputSchema={
                "type": "object",
                "properties": {},
            },
        ),
    ]


@server.call_tool()
async def call_tool(name: str, arguments: Any) -> list[TextContent]:
    """Handle tool calls for the Drain3 MCP server."""
    if name == "parse_log":
        log_line = arguments.get("log_line")
        if not log_line:
            return [TextContent(type="text", text=json.dumps({"error": "Missing 'log_line' argument"}))]
        
        # Process log line with Drain3
        result = miner.add_log_message(log_line)
        
        # Format output
        output = {
            "cluster_id": result["cluster_id"],
            "template": result["template_mined"],
            "change_type": result["change_type"],
        }
        
        return [TextContent(type="text", text=json.dumps(output, indent=2))]
    
    elif name == "get_clusters":
        # Get all clusters
        clusters = []
        for cluster_id, cluster in miner.drain.clusters.items():
            clusters.append({
                "cluster_id": cluster_id,
                "template": cluster.get_template(),
                "size": cluster.size,
            })
        
        output = {
            "total_clusters": len(clusters),
            "clusters": clusters,
        }
        
        return [TextContent(type="text", text=json.dumps(output, indent=2))]
    
    else:
        return [TextContent(type="text", text=json.dumps({"error": f"Unknown tool: {name}"}))]


async def main():
    """Run the MCP server using stdio transport."""
    from mcp.server.stdio import stdio_server
    
    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            server.create_initialization_options()
        )


if __name__ == "__main__":
    asyncio.run(main())
