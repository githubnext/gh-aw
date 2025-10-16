---
on:
  workflow_dispatch:
    inputs:
      log_file_path:
        description: 'Path to log file in the repository'
        required: false
        type: string
        default: '.github/workflows/ci.yml'

permissions:
  contents: read

engine: claude

imports:
  - shared/mcp/drain3.md

timeout_minutes: 10
---

# Drain3 Log Pattern Mining Example

This workflow demonstrates how to use the Drain3 MCP server for log pattern extraction.

## Available Tools

The drain3 MCP server provides three tools:

1. **index_file** - Analyze a log file and extract patterns
   - Streams JSONL events with progress updates
   - Creates a persistent snapshot for future queries
   
2. **query_file** - Match a specific log line against extracted patterns
   - Returns the matching cluster information
   
3. **list_templates** - List all extracted log templates
   - Useful for reviewing what patterns were found

## Example Task

Analyze the log file at: "${{ github.event.inputs.log_file_path }}"

Steps:
1. Create a sample log file if needed (or use the provided path)
2. Use `index_file` to analyze the log and extract templates
3. Use `list_templates` to see what patterns were found
4. Try `query_file` with a specific log line to see which pattern it matches

Provide a summary of the analysis results.
