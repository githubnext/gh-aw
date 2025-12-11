---
permissions:
  actions: read
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
    run: |
      # Check if gh-aw extension is already installed
      if gh extension list | grep -q "githubnext/gh-aw"; then
        echo "gh-aw extension already installed, upgrading..."
        gh extension upgrade gh-aw || true
      else
        echo "Installing gh-aw extension..."
        make install
      fi
      gh aw --version
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
  - name: Start MCP server
    run: |
      set -e
      ./gh-aw mcp-server --cmd ./gh-aw --port 8765 &
      MCP_PID=$!
      
      # Wait a moment for server to start
      sleep 2
      
      # Check if server is still running
      if ! kill -0 $MCP_PID 2>/dev/null; then
        echo "MCP server failed to start"
        exit 1
      fi
      
      echo "MCP server started successfully with PID $MCP_PID"
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---
