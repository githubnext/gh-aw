---
"gh-aw": patch
---

Fix patch generation to handle underscored safe-output type names

The patch generation script now correctly searches for underscored type names (`push_to_pull_request_branch`, `create_pull_request`) to match the format used by the safe-outputs MCP server. This fixes a mismatch that was causing the `push_to_pull_request_branch` safe-output job to fail when looking for the patch file.
