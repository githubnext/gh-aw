---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
---

# Test Copilot Imports

This is a test workflow to verify that import directives with cycles are handled correctly.

{{#runtime-import shared/keep-it-short.md}}

{{#runtime-import shared/use-emojis.md}}

Process the issue and respond with a helpful comment.
