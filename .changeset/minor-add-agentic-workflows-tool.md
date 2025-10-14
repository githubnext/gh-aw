---
"gh-aw": minor
---

Add builtin "agentic-workflows" tool for workflow introspection and analysis

Adds a new builtin tool that enables AI agents to analyze GitHub Actions workflow traces and improve workflows based on execution history. The tool exposes the `gh aw mcp-server` command as an MCP server, providing agents with four powerful capabilities:

- **status** - Check compilation status and GitHub Actions state of all workflows
- **compile** - Programmatically compile markdown workflows to YAML
- **logs** - Download and analyze workflow run logs with filtering options
- **audit** - Investigate specific workflow run failures with detailed diagnostics

When enabled in a workflow's frontmatter, the tool automatically installs the gh-aw extension and configures the MCP server for all supported engines (Claude, Copilot, Custom, Codex). This enables continuous workflow improvement driven by AI analysis of actual execution data.
