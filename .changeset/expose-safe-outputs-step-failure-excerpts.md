---
"gh-aw": minor
---

Made `safe_outputs` job failures diagnosable from `gh aw audit` and the `agenticworkflows` MCP tools. Failed steps in the audit report now carry an `error_excerpt` field containing a truncated excerpt of their GitHub Actions step log (preferring `##[error]` annotations, falling back to the tail of the log), and a new `safe-outputs` artifact set downloads the `safe-outputs-items` artifact (`safe-output-items.jsonl`, `temporary-id-map.json`) without requiring `--artifacts all`.
