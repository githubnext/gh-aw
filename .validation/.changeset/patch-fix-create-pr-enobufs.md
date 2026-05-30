---
"gh-aw": patch
---

Fixed `create_pull_request` failing with `spawnSync git ENOBUFS` on large diffs (e.g. 47+ changed files).

The `execGitSync` helper in `git_helpers.cjs` used Node.js `spawnSync` without an explicit `maxBuffer`, defaulting to ~1 MB. When `git format-patch --stdout` produced output exceeding that limit, all patch generation strategies silently failed with a misleading "No changes to commit" error.

The fix:
- Set `maxBuffer: 100 * 1024 * 1024` (100 MB) as the default in `execGitSync`, matching the `max_patch_size` headroom and consistent with other handlers in the codebase (e.g. MCP handlers use 10 MB).
- Detect `ENOBUFS` errors and throw an actionable error message that surfaces the real cause instead of the generic "no commits found" fallback.
- Callers can still override `maxBuffer` via the options spread.
