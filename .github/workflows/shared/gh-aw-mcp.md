---
mcp-servers:
  gh-aw:
    command: "./gh-aw"
    args: ["mcp-server"]
    env:
      GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  - name: Install dependencies
    run: make deps-dev
  - name: Build gh-aw CLI
    run: make build
  - name: Install binary as 'gh-aw'
    run: make install
---
