---
on:
  workflow_dispatch:
permissions: read-all
engine: claude
safe-outputs:
  staged: true
  upload-asset:
  create-issue:
    title-prefix: "[test] "
tools:
  playwright:
    allowed_domains: ["example.com"]
    args: ["--browser", "chromium"]
---

# Test Playwright Args Configuration

This workflow tests the `args` field for Playwright MCP server configuration.

The workflow is configured with:
- `allowed_domains: ["example.com"]` to access example.com
- `args: ["--browser", "chromium"]` to explicitly specify the browser

Please perform the following tasks:

1. Use Playwright to navigate to `https://example.com`
2. Wait for the page to fully load
3. Take a screenshot of the page
4. Save the screenshot to `/tmp/gh-aw/example-screenshot.png`
5. Use the `upload asset` tool to upload the screenshot
6. Create an issue with the screenshot URL and confirm that Playwright is working with the custom args

The args field allows passing additional command-line arguments to the Playwright MCP server, such as browser selection or custom flags for testing scenarios.
