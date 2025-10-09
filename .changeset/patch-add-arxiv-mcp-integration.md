---
"githubnext/gh-aw": patch
---

Add arXiv MCP server integration to scout workflow with custom cache-memory configuration

This adds support for the arXiv MCP server to enhance the scout workflow's research capabilities with access to academic research papers and preprints. The integration includes:

- New shared arXiv MCP configuration file (`shared/arxiv-mcp.md`)
- Updated scout workflow to import and use arXiv MCP tools
- Custom cache-memory configuration with "arxiv" key and 60-day retention for persistent memory storage
- Support for searching arXiv papers, retrieving paper details, and accessing PDFs
