---
"gh-aw": patch
---

Fix `gh-aw` binary detection in generated MCP setup steps so workflow runs do not fail under `bash -e` and `set -o pipefail` when the extension binary is installed outside `$PATH`.
