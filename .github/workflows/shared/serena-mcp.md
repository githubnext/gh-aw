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
    allowed: ["*"]
steps:
  - name: Setup python
    uses: actions/setup-python@v6
    with:
      python-version: "3.13"
---

Activate the current dir as project using serena.

<!--

  # https://github.com/mcp/oraios/serena#using-docker-experimental

-->