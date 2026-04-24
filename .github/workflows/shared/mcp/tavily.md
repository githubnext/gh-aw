---
mcp-servers:
  tavily:
    type: http
    container: "node:lts-alpine"
    url: "https://mcp.tavily.com/mcp/"
    headers:
      Authorization: "Bearer ${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
---
