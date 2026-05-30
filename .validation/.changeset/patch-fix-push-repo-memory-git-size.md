---
"gh-aw": patch
---

Fix `push_repo_memory` tool false-positive size limit errors. The tool was including `.git` directory contents (git object pack files) when calculating total memory size, causing workflows with small actual memory files (~500 bytes) to report inflated sizes (~30 KB) and hit the configured limit. The fix excludes the `.git` directory from the size scan, measuring only actual working tree files.
