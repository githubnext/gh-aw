---
"gh-aw": patch
---

Resolve the Copilot CLI config directory from `$HOME` instead of a hard-coded `/home/runner` so workflows using the `copilot` engine work on self-hosted and containerized runners where `HOME` is not `/home/runner`. Affects the generated `mkdir`, `XDG_CONFIG_HOME`, `GH_AW_MCP_CONFIG`, settings-file write/cleanup, MCP gateway converter, and detection-job MCP cleanup. Lock files (`.lock.yml`) will see mechanical diffs on the next `gh aw compile`; no behavior change on standard GitHub-hosted runners.
