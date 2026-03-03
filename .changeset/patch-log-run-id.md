---
"gh-aw": patch
---

Ensures workflow dispatch logs the run ID by using `return_run_details` (with a retry for older GHES) and parses `gh workflow run` output so the CLI keeps working if the new field is missing.
