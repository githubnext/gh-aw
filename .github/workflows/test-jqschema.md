---
name: Test jqschema
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
timeout_minutes: 5
imports:
  - shared/jqschema.md
tools:
  github:
    toolsets: [repos]
  bash: ["cat", "echo", "/tmp/gh-aw/jqschema.sh"]
---

# Test jqschema

Test the jqschema utility.

1. Create a test JSON file with complex structure
2. Run the jqschema.sh script on it
3. Verify the output shows only types and structure
4. Report the results
