---
"gh-aw": patch
---

Support custom AWF installation path in firewall configuration. Adds support for specifying a custom AWF binary path in the workflow frontmatter `network.firewall.path` so users can validate and use their own AWF binary instead of downloading releases from GitHub.

When `path` is set, the `version` field is ignored and AWF download is skipped.

Affected files: `pkg/workflow/firewall.go`, `pkg/parser/schemas/main_workflow_schema.json`, `pkg/workflow/frontmatter_extraction.go`, `pkg/workflow/copilot_engine.go`.

