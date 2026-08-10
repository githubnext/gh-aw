---
# arXiv MCP Server
# SECURITY: docker.io/mcp/arxiv-mcp-server has Critical/High CVEs with no upstream fix available (issue #51713).
# The container definition has been removed until a patched image is published upstream.
# To re-enable, restore the mcp-servers block and update the pinned digest in actions-lock.json.
#
# Available tools (when enabled):
#   - search_arxiv: Search for papers on arXiv by keywords, authors, or topics
#   - get_paper_details: Get detailed metadata about a specific arXiv paper
#   - get_paper_pdf: Retrieve the PDF content of an arXiv paper
#
# Usage:
#   imports:
#     - shared/mcp/arxiv.md
---
