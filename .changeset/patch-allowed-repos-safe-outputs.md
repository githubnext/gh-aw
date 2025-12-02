---
"gh-aw": patch
---

Add support for `allowed-repos` in `create-issue` and `create-discussion`
safe-outputs. Agent outputs may now include an optional `repo` field to
target a repository from the configured `allowed-repos`. Temporary IDs
are now resolved to `(repo, number)` pairs while remaining backward
compatible with the legacy single-repo format.

This is an internal enhancement to expand safe-outputs to support
creating issues/discussions across multiple repositories.

