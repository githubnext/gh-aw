---
"gh-aw": patch
---
Fix `create_pull_request` 100-file limit to count unique files (not raw `diff --git` headers, which inflate the count when a single push contains multiple commits touching the same files), and add a configurable `max-patch-files` top-level safe-outputs option (default 100) so long-running branches can opt into a higher limit.
