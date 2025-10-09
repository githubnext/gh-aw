---  
mcp-servers:
  serena:
    command: "uvx"
    args:
      - "--from"
      - "git+https://github.com/oraios/serena"
      - "serena"
      - "start-mcp-server"
      - "--context"
      - "codex"
      - "--project"
      - "${{ github.workspace }}"
    allowed: ["*"]
steps:
  - name: Verify uv
    run: uv --version
  - name: Install Go language service
    run: go install golang.org/x/tools/gopls@latest
  - name: Check gopls version
    run: gopls version
---

## Serena configuration

The active workspaces is ${{ github.workspace }}. You should configure the Serena memory at the cache-memory folder (/tmp/gh-aw/cache-memory/serena).

<!--

  # https://github.com/mcp/oraios/serena#using-docker-experimental

-->