---
"gh-aw": minor
---

Add `gh aw logs episode <run-id-or-url>` and MCP `logs.episode_run` for bounded
canonical episode retrieval. Follow recorded parents and roots and discover
members only in explicitly allowed repositories and a fixed time window.
Candidate-run, REST-request, and timeout caps cover the whole collection;
missing evidence and exhausted limits qualify coverage without claiming
complete membership. Compact artifacts are cached by host, repository, run,
attempt, and immutable artifact ID.

Episode outcome reports read recorded run artifact paths beneath `--output`,
so isolated retrieval caches remain available for offline cost accounting.
