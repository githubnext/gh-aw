---
name: Test JSON Schema Minify
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
timeout_minutes: 5
imports:
  - shared/json-schema-minify.md
tools:
  bash: ["cat", "echo", "/tmp/gh-aw/jq/json-schema-minify.sh"]
---

# Test JSON Schema Minify

Test the JSON schema minifier utility.

1. Create a test JSON file with complex structure
2. Run the json-schema-minify.sh script on it
3. Verify the output shows only types and structure
4. Report the results
