---
on: 
  workflow_dispatch:
name: Dev
description: Test upload-asset with Python graph generation
timeout-minutes: 5
strict: true
engine: copilot

permissions:
  contents: read
  issues: read

sandbox:
  mcp:
    port: 8080

tools:
  github:
    toolsets: [issues]

safe-outputs:
  upload-asset:
    allowed-exts: [".png", ".jpg"]
    max: 5
  create-issue:
    title-prefix: "[Dev Test] "
    max: 1

imports:
  - shared/gh.md
  - shared/python-dataviz.md
---

# Test Upload Asset with Python Graph Generation

Create a dummy graph using Python and matplotlib, then upload it as an asset.

**Requirements:**
1. Use Python to create a simple graph (e.g., a sine wave or bar chart) using matplotlib
2. Save the graph as a PNG file to /tmp/graph.png
3. Use the `upload_asset` tool to upload the graph
4. The tool should return a URL where the graph can be accessed
5. Create an issue that includes the graph using markdown image syntax
6. Verify that:
   - The graph file was created successfully
   - The asset was uploaded and a URL was returned
   - The issue was created with the embedded graph image
