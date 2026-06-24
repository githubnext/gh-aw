---
"gh-aw": patch
---

perf: parallelize audit analysis tasks to reduce latency for long-running workflows

All independent analysis operations in `AuditWorkflowRun` (log metrics
extraction, firewall log analysis, token usage, job details, missing tools,
etc.) now execute concurrently using goroutines and `sync.WaitGroup`. This
eliminates sequential I/O wait time that previously scaled with log volume,
bringing the audit tool within the <30 s performance target for complex
workflows such as the 32-turn Static Analysis Report that previously took
over 72 s.
