---
mcp-servers:
  gh-aw:
    command: "./gh-aw"
    args: ["mcp-server"]
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
steps:
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  - name: Install dependencies
    run: make deps-dev
  - name: Install binary as 'gh-aw'
    run: make install
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---
