---
mcp-servers:
  gh-aw:
    type: http
    url: http://localhost:8765
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  - name: Install dependencies
    run: make deps-dev
  - name: Install binary as 'gh-aw'
    run: make build
  - name: Start MCP server
    run: ./gh-aw mcp-server --port 8765 &
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---
