---
"gh-aw": patch
---

Escape single quotes and backslashes when embedding JSON into shell environment
variables to prevent shell injection. This fixes a code-scanning finding
(`go/unsafe-quoting`) by properly escaping backslashes and single quotes
before inserting JSON into a single-quoted shell string.

Files changed:
- `pkg/workflow/update_project_job.go` (apply POSIX-compatible escaping)

This is an internal security fix and does not change the public CLI API.

