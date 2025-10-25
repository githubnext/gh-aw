---
steps:
  - name: Download workflow logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e
      echo "Downloading workflow logs to /tmp/gh-aw/aw-mcp/logs..."
      mkdir -p /tmp/gh-aw/aw-mcp/logs
      
      # Download logs from the last 7 days by default
      # Modify the parameters as needed for your workflow
      ./gh-aw logs --start-date -7d -o /tmp/gh-aw/aw-mcp/logs
      
      echo "Logs downloaded successfully"
      ls -lh /tmp/gh-aw/aw-mcp/logs
---

This shared configuration pre-downloads workflow logs before the AI agent runs.

## Usage

Import this shared configuration in your workflow:

```yaml
imports:
  - shared/predownload-logs.md
```

## Customization

You can customize the log download by modifying the parameters in your workflow's `steps:` section:

```yaml
steps:
  - name: Download workflow logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Download logs for a specific workflow
      ./gh-aw logs my-workflow --start-date -7d -o /tmp/gh-aw/aw-mcp/logs
      
      # Or download logs from the last 30 days
      ./gh-aw logs --start-date -30d -o /tmp/gh-aw/aw-mcp/logs
      
      # Or download logs with specific filters
      ./gh-aw logs --engine claude --count 50 -o /tmp/gh-aw/aw-mcp/logs
```

## Integration with agentic-workflows MCP Tool

When used together with the `agentic-workflows` tool import, the logs are pre-downloaded
and available for the AI agent to analyze without needing to download them during execution.

This improves performance and reduces the time the AI agent spends waiting for downloads.
