---
"gh-aw": patch
---

Fixed a runtime regression where valid zero-argument custom safe-output tools were incorrectly treated as schema-discovery probes. The MCP CLI bridge now inspects the tool's `inputSchema.required` array before deciding whether to show help: it only shows help when required fields are declared, allowing zero-input and optional-only custom safe-output jobs to proceed to `tools/call` with `{}` as expected.
