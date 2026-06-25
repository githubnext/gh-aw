---
"gh-aw": patch
---
Improve `gh aw update --org` mode: show `[n/total]` per-repository progress, skip parse/preview errors instead of aborting the run, stop gracefully on Ctrl-C while still printing the gathered report, and short-circuit when the GitHub API budget reaches a critical level (limit minus 1000 API units). Rate-limit telemetry is now logged for debugging. Source fields without an explicit branch ref now resolve the repository's actual default branch via the GitHub API instead of assuming `main`.
