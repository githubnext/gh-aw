---
mcp-servers:
  tavily:
    type: http
    url: "https://mcp.tavily.com/mcp/"
    headers:
      Authorization: "Bearer ${{ secrets.TAVILY_API_KEY }}"
    # Security decision (2026-05-19): keep wildcard pending follow-up inventory of Tavily tool variants.
    # Multiple production workflows depend on this integration for broad web research tasks.
    allowed: ["*"]
---
