---
"githubnext/gh-aw": patch
---

Organize MCP server shared workflows into dedicated mcp/ subdirectory with cleaner naming

Reorganizes shared workflow files that expose MCP servers into `.github/workflows/shared/mcp/` subdirectory for better discoverability and maintainability. Files are renamed to remove redundant `-mcp` suffix since the directory name already indicates they are MCP server configurations. Updates all import paths across workflows, documentation, and templates. Includes bug fix to `isWorkflowSpec()` function to correctly handle the new `shared/` prefix in import paths.
