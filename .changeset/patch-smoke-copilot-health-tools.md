---
"gh-aw": patch
---

Add firewall health endpoint test and display the list of available tools
to the `smoke-copilot` workflow. This adds non-breaking test checks that
curl the (redacted) firewall health endpoint and prints the HTTP status
code, and ensures the workflow displays the available tools for debugging.

These are test and workflow changes only and do not modify the CLI API.

