---
mcp-servers:
  arxiv:
    container: "mcp/arxiv-mcp-server"
    allowed:
      - search_arxiv
      - get_paper_details
      - get_paper_pdf
---

<!--

# arXiv MCP Server
# Access to arXiv research papers
#
# Provides access to arXiv's extensive research paper repository
# Documentation: https://hub.docker.com/r/mcp/arxiv-mcp-server
#
# Available tools:
#   - search_arxiv: Search for papers on arXiv by keywords, authors, or topics
#   - get_paper_details: Get detailed metadata about a specific arXiv paper
#   - get_paper_pdf: Retrieve the PDF content of an arXiv paper
#
# Usage:
#   imports:
#     - shared/arxiv-mcp.md

-->
