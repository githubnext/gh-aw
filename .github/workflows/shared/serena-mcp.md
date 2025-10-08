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
  - name: Setup python
    uses: actions/setup-python@v6
    with:
      python-version: "3.13"
  - name: Install uv
    uses: astral-sh/setup-uv@v6
  - name: Verify uv
    run: uv --version
  - name: Setup go
    uses: actions/setup-go@v4
  - name: Install Go language service
    run: go install golang.org/x/tools/gopls@latest
  - name: Check gopls version
    run: gopls version
---

Activate the current dir as project using serena.

<!--

  # https://github.com/mcp/oraios/serena#using-docker-experimental

-->