---
"gh-aw": patch
---

Load safe outputs tool definitions at runtime from `actions/setup` via `tools_meta.json` instead of inlining filtered JSON into each compiled lock file, reducing lock-file size and avoiding full workflow recompiles when tool descriptions change.
