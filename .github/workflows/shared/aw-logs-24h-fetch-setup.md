---
# Pre-fetch last 24 hours of agentic workflow logs for analysis
# Saves logs to /tmp/gh-aw/aw-mcp/logs/

tools:
  agentic-workflows:
  cache-memory: true
  timeout: 300

steps:
  - name: Download logs from last 24 hours
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: ./gh-aw logs --start-date -1d -o /tmp/gh-aw/aw-mcp/logs
---
